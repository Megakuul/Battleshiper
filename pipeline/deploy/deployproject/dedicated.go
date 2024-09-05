package deployproject

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	goformation "github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/batch"
	"github.com/awslabs/goformation/v7/cloudformation/events"
	"github.com/awslabs/goformation/v7/cloudformation/iam"

	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

func updateDedicatedInfrastructure(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	_, err := eventCtx.CloudformationClient.GetTemplate(transportCtx, &cloudformation.GetTemplateInput{
		StackName: aws.String(projectDoc.DedicatedInfrastructure.StackName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudformation stack: %v", err)
	}
}

func addServerSystem(stackTemplate *goformation.Template, eventCtx *eventcontext.Context, projectDoc *project.Project) {
	const BUILD_JOB_ROLE string = "BuildJobRole"
	stackTemplate.Resources[BUILD_JOB_ROLE] = &iam.Role{
		RoleName:    aws.String(fmt.Sprintf("battleshiper-project-build-job-role-%s", projectDoc.Name)),
		Description: aws.String("role associated with the project build job"),
		AssumeRolePolicyDocument: map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Service": "batch.amazonaws.com",
					},
					"Action": "sts:AssumeRole",
				},
			},
		},
		Policies: []iam.Role_Policy{
			{
				PolicyName: fmt.Sprintf("battleshiper-build-asset-bucket-access-%s", projectDoc.Name),
				PolicyDocument: map[string]interface{}{
					"Version": "2012-10-17",
					"Statement": []map[string]interface{}{
						{
							"Effect":   "Allow",
							"Action":   "s3:PutObject",
							"Resource": fmt.Sprintf("arn:aws:s3:::%s/*", projectDoc.SharedInfrastructure.BuildAssetBucketPath),
						},
					},
				},
			},
		},
	}

	const BUILD_JOB_DEFINITION string = "BuildJobDefinition"
	stackTemplate.Resources[BUILD_JOB_DEFINITION] = &batch.JobDefinition{
		JobDefinitionName: aws.String(fmt.Sprintf("battleshiper-project-build-job-%s", projectDoc.Name)),
		Type:              "container",
		ContainerProperties: &batch.JobDefinition_ContainerProperties{
			Image:      projectDoc.BuildImage,
			Vcpus:      aws.Int(eventCtx.BuildConfiguration.BuildJobVCPUS),
			Memory:     aws.Int(eventCtx.BuildConfiguration.BuildJobMemory),
			JobRoleArn: aws.String(goformation.Ref(BUILD_JOB_ROLE)),
			Environment: []batch.JobDefinition_Environment{
				{
					Name:  aws.String("BUILD_ASSET_BUCKET_PATH"),
					Value: aws.String(projectDoc.SharedInfrastructure.BuildAssetBucketPath),
				},
			},
			Command: []string{
				"/bin/sh", "-c",
				"git clone $REPOSITORY_URL . &&",
				"$BUILD_COMMAND &&",
				"aws s3 cp $OUTPUT_DIRECTORY s3://$BUILD_ASSET_BUCKET_PATH --recursive",
			},
			NetworkConfiguration: &batch.JobDefinition_NetworkConfiguration{
				AssignPublicIp: aws.String("ENABLED"),
			},
		},
		Timeout: &batch.JobDefinition_Timeout{
			AttemptDurationSeconds: aws.Int(int(eventCtx.BuildConfiguration.BuildJobTimeout.Seconds())),
		},
	}

	const BUILD_RULE_ROLE string = "BuildRuleRole"
	stackTemplate.Resources[BUILD_RULE_ROLE] = &iam.Role{
		RoleName:    aws.String(fmt.Sprintf("battleshiper-project-build-rule-role-%s", projectDoc.Name)),
		Description: aws.String("role to invoke the targets specified in the associated build rule"),
		AssumeRolePolicyDocument: map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Service": "events.amazonaws.com",
					},
					"Action": "sts:AssumeRole",
				},
			},
		},
		ManagedPolicyArns: []string{
			eventCtx.BuildConfiguration.BuildJobQueuePolicyArn,
		},
	}

	const BUILD_RULE string = "BuildRule"
	stackTemplate.Resources[BUILD_RULE] = &events.Rule{
		EventBusName: aws.String(eventCtx.BuildConfiguration.BuildEventbusName),
		Name:         aws.String(fmt.Sprintf("battleshiper-project-build-rule-%s", projectDoc.Name)),
		Description:  aws.String("triggers the associated build targets"),
		State:        aws.String("ENABLED"),
		EventPattern: map[string]interface{}{
			"source": []string{
				eventCtx.BuildConfiguration.BuildEventSource,
			},
			"detail-type": []string{
				fmt.Sprintf("%s.%s", eventCtx.BuildConfiguration.BuildEventAction, projectDoc.Name),
			},
		},
		Targets: []events.Rule_Target{
			{
				Arn:     eventCtx.BuildConfiguration.BuildJobQueueArn,
				Id:      "battleshiper-project-build-queue",
				RoleArn: aws.String(goformation.GetAtt(BUILD_RULE_ROLE, "Arn")),
				BatchParameters: &events.Rule_BatchParameters{
					JobDefinition: goformation.Ref(BUILD_JOB_DEFINITION),
					JobName:       fmt.Sprintf("battleshiper-project-build-job-%s", projectDoc.Name),
				},
				InputTransformer: &events.Rule_InputTransformer{
					InputPathsMap: map[string]string{
						"TMPL_EXECUTION_IDENTIFIER": "$.detail.execution_identifier",
						"TMPL_DEPLOY_TICKET":        "$.detail.deploy_ticket",
						"TMPL_REPOSITORY_URL":       "$.detail.repository_url",
						"TMPL_BUILD_COMMAND":        "$.detail.build_command",
						"TMPL_OUTPUT_DIRECTORY":     "$.detail.output_directory",
					},
					InputTemplate: `{
						"parameters": {
							"executionIdentifier": "<TMPL_INTERNAL_EXECUTION_IDENTIFIER>",
							"deployTicket": 			 "<TMPL_INTERNAL_DEPLOY_TICKET>",
						},
						"containerOverrides": "environment: [
							{
								"name": "REPOSITORY_URL",
								"value": "<TMPL_REPOSITORY_URL>"
							},
							{
								"name": "BUILD_COMMAND",
								"value": "<TMPL_BUILD_COMMAND>"
							},
							{
								"name": "OUTPUT_DIRECTORY",
								"value: "<TMPL_OUTPUT_DIRECTORY>"
							}
						}] 
					}`,
				},
			},
		},
	}
}
