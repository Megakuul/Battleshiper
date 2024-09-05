package initproject

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	goformation "github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/batch"
	"github.com/awslabs/goformation/v7/cloudformation/events"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/awslabs/goformation/v7/cloudformation/logs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

// initializeDedicatedInfrastructure initializes the dedicated infrastructure components required by the project.
func initializeDedicatedInfrastructure(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) (*project.Project, error) {
	if projectDoc.DedicatedInfrastructure.StackName != "" {
		return nil, fmt.Errorf("failed to create dedicated stack; project already holds a stack")
	}

	if projectDoc.SharedInfrastructure.BuildAssetBucketPath == "" {
		return nil, fmt.Errorf("invalid build asset bucket path: empty path is not allowed")
	}

	projectDoc.DedicatedInfrastructure.EventLogGroup = fmt.Sprintf(
		"%s/%s", eventCtx.BuildConfiguration.EventLogPrefix, projectDoc.Name)
	projectDoc.DedicatedInfrastructure.BuildLogGroup = fmt.Sprintf(
		"%s/%s", eventCtx.BuildConfiguration.BuildLogPrefix, projectDoc.Name)
	projectDoc.DedicatedInfrastructure.DeployLogGroup = fmt.Sprintf(
		"%s/%s", eventCtx.BuildConfiguration.DeployLogPrefix, projectDoc.Name)
	projectDoc.DedicatedInfrastructure.ServerLogGroup = fmt.Sprintf(
		"%s/%s", eventCtx.BuildConfiguration.FunctionLogPrefix, projectDoc.Name)
	stackTemplate := goformation.NewTemplate()
	if err := addProject(stackTemplate, &eventCtx, projectDoc); err != nil {
		return nil, fmt.Errorf("failed to serialize build system blueprint")
	}

	stackName := fmt.Sprintf("battleshiper-project-stack-%s", projectDoc.Name)
	stackBody, err := stackTemplate.JSON()
	if err != nil {
		return nil, fmt.Errorf("failed to parse cloudformation stack body")
	}
	projectDoc.DedicatedInfrastructure.StackName = stackName

	_, err = eventCtx.CloudformationClient.CreateStack(transportCtx, &cloudformation.CreateStackInput{
		StackName:    aws.String(stackName),
		TemplateBody: aws.String(string(stackBody)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudformation stack: %v", err)
	}

	projectCollection := eventCtx.Database.Collection(project.PROJECT_COLLECTION)

	updatedDoc := &project.Project{}
	err = projectCollection.FindOneAndUpdate(transportCtx, bson.M{"_id": projectDoc.MongoID}, bson.M{
		"$set": bson.M{
			"dedicated_infrastructure": projectDoc.DedicatedInfrastructure,
		},
	}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updatedDoc)
	if err != nil {
		_, err := eventCtx.CloudformationClient.DeleteStack(transportCtx, &cloudformation.DeleteStackInput{
			StackName:    aws.String(stackName),
			DeletionMode: types.DeletionModeStandard,
		})
		if err != nil {
			log.Printf("ERROR RUNTIME: failed to delete stack '%s'. failed to reference stack in database; stack is leaking.\n", stackName)
		}
		return nil, fmt.Errorf("failed to update project on database")
	}

	waiter := cloudformation.NewStackCreateCompleteWaiter(eventCtx.CloudformationClient)
	err = waiter.Wait(transportCtx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}, eventCtx.DeploymentTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to apply cloudformation stack: %v", err)
	}

	return updatedDoc, nil
}

type inputEnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type inputContainerOverrides struct {
	Environment []inputEnvironmentVariable `json:"environment"`
}

type inputTransformTemplate struct {
	Parameters         event.DeployParameters  `json:"parameters"`
	ContainerOverrides inputContainerOverrides `json:"containerOverrides"`
}

func addProject(stackTemplate *goformation.Template, eventCtx *eventcontext.Context, projectDoc *project.Project) error {
	const EVENT_LOG_GROUP string = "EventLogGroup"
	stackTemplate.Resources[EVENT_LOG_GROUP] = &logs.LogGroup{
		LogGroupName:    aws.String(projectDoc.DedicatedInfrastructure.EventLogGroup),
		RetentionInDays: aws.Int(eventCtx.BuildConfiguration.LogRetentionDays),
	}

	const BUILD_LOG_GROUP string = "BuildLogGroup"
	stackTemplate.Resources[BUILD_LOG_GROUP] = &logs.LogGroup{
		LogGroupName:    aws.String(projectDoc.DedicatedInfrastructure.BuildLogGroup),
		RetentionInDays: aws.Int(eventCtx.BuildConfiguration.LogRetentionDays),
	}

	const DEPLOY_LOG_GROUP string = "DeployLogGroup"
	stackTemplate.Resources[DEPLOY_LOG_GROUP] = &logs.LogGroup{
		LogGroupName:    aws.String(projectDoc.DedicatedInfrastructure.DeployLogGroup),
		RetentionInDays: aws.Int(eventCtx.BuildConfiguration.LogRetentionDays),
	}

	const SERVER_LOG_GROUP string = "ServerLogGroup"
	stackTemplate.Resources[SERVER_LOG_GROUP] = &logs.LogGroup{
		LogGroupName:    aws.String(projectDoc.DedicatedInfrastructure.ServerLogGroup),
		RetentionInDays: aws.Int(eventCtx.BuildConfiguration.LogRetentionDays),
	}

	const BUILD_JOB_EXEC_ROLE string = "BuildJobExecRole"
	stackTemplate.Resources[BUILD_JOB_EXEC_ROLE] = &iam.Role{
		RoleName:    aws.String(fmt.Sprintf("battleshiper-project-build-job-exec-role-%s", projectDoc.Name)),
		Description: aws.String("role associated with aws batch, it is responsible to manage the running job"),
		AssumeRolePolicyDocument: map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Service": "ecs-tasks.amazonaws.com",
					},
					"Action": "sts:AssumeRole",
				},
			},
		},
		Policies: []iam.Role_Policy{
			{
				PolicyName: fmt.Sprintf("battleshiper-project-log-access-%s", projectDoc.Name),
				PolicyDocument: map[string]interface{}{
					"Version": "2012-10-17",
					"Statement": []map[string]interface{}{
						{
							"Effect": "Allow",
							"Action": []string{
								"logs:CreateLogStream",
								"logs:PutLogEvents",
							},
							"Resource": goformation.GetAtt(BUILD_LOG_GROUP, "Arn"),
						},
					},
				},
			},
		},
	}

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
			Image:            projectDoc.BuildImage,
			Vcpus:            aws.Int(eventCtx.BuildConfiguration.BuildJobVCPUS),
			Memory:           aws.Int(eventCtx.BuildConfiguration.BuildJobMemory),
			JobRoleArn:       aws.String(goformation.Ref(BUILD_JOB_ROLE)),
			ExecutionRoleArn: aws.String(goformation.Ref(BUILD_JOB_EXEC_ROLE)),
			LogConfiguration: &batch.JobDefinition_LogConfiguration{
				LogDriver: "awslogs",
				Options: map[string]string{
					"awslogs-group": projectDoc.DedicatedInfrastructure.BuildLogGroup,
				},
			},
			Environment: []batch.JobDefinition_Environment{
				{
					Name:  aws.String("BUILD_ASSET_BUCKET_PATH"),
					Value: aws.String(projectDoc.SharedInfrastructure.BuildAssetBucketPath),
				},
			},
			Command: []string{
				"/bin/sh", "-c",
				"echo \"START BUILD $EXECUTION_IDENTIFIER\" &&",
				"git clone $REPOSITORY_URL . &&",
				"$BUILD_COMMAND &&",
				"aws s3 cp $OUTPUT_DIRECTORY s3://$BUILD_ASSET_BUCKET_PATH/$EXECUTION_IDENTIFIER --recursive",
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

	inputPathsMap := map[string]string{
		"TMPL_EXECUTION_IDENTIFIER": "$.detail.execution_identifier",
		"TMPL_DEPLOY_TICKET":        "$.detail.deploy_ticket",
		"TMPL_REPOSITORY_URL":       "$.detail.repository_url",
		"TMPL_BUILD_COMMAND":        "$.detail.build_command",
		"TMPL_OUTPUT_DIRECTORY":     "$.detail.output_directory",
	}

	inputTemplate := &inputTransformTemplate{
		Parameters: event.DeployParameters{
			DeployTicket:        "<TMPL_DEPLOY_TICKET>",
			ExecutionIdentifier: "<TMPL_EXECUTION_IDENTIFIER>",
		},
		ContainerOverrides: inputContainerOverrides{
			Environment: []inputEnvironmentVariable{
				{
					Name:  "EXECUTION_IDENTIFIER",
					Value: "<TMPL_EXECUTION_IDENTIFIER>",
				},
				{
					Name:  "REPOSITORY_URL",
					Value: "<TMPL_REPOSITORY_URL>",
				},
				{
					Name:  "BUILD_COMMAND",
					Value: "<TMPL_BUILD_COMMAND>",
				},
				{
					Name:  "OUTPUT_DIRECTORY",
					Value: "<TMPL_OUTPUT_DIRECTORY>",
				},
			},
		},
	}
	inputTemplateRaw, err := json.Marshal(inputTemplate)
	if err != nil {
		return err
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
				RetryPolicy: &events.Rule_RetryPolicy{
					MaximumRetryAttempts:     aws.Int(5),
					MaximumEventAgeInSeconds: aws.Int(150),
				},
				BatchParameters: &events.Rule_BatchParameters{
					JobDefinition: goformation.Ref(BUILD_JOB_DEFINITION),
					JobName:       fmt.Sprintf("battleshiper-project-build-job-%s", projectDoc.Name),
				},
				InputTransformer: &events.Rule_InputTransformer{
					InputPathsMap: inputPathsMap,
					InputTemplate: string(inputTemplateRaw),
				},
			},
		},
	}
	return nil
}
