package initproject

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	goformation "github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/batch"
	"github.com/awslabs/goformation/v7/cloudformation/events"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

// initializeDedicatedInfrastructure initializes the dedicated infrastructure components required by the project.
func initializeDedicatedInfrastructure(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	if projectDoc.DedicatedInfrastructure.StackName != "" {
		return fmt.Errorf("failed to create dedicated stack; project already holds a stack")
	}

	stackTemplate := goformation.NewTemplate()

	stackTemplate.Resources["BuildRuleRole"] = &iam.Role{
		RoleName:    aws.String(fmt.Sprintf("battleshiper-project-build-rule-role-%s", projectDoc.Name)),
		Description: aws.String(fmt.Sprintf("role to invoke the targets specified in the associated build rule")),
		AssumeRolePolicyDocument: map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				map[string]interface{}{
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Service": "events.amazonaws.com",
					},
					"Action": "sts:AssumeRole",
				},
			},
		},
		ManagedPolicyArns: []string{
			eventCtx.BuildJobConfiguration.BuildJobQueuePolicyArn,
		},
	}

	stackTemplate.Resources["BuildRule"] = &events.Rule{
		EventBusName: aws.String(eventCtx.BuildJobConfiguration.BuildEventbusName),
		Name:         aws.String(fmt.Sprintf("battleshiper-project-build-rule-%s", projectDoc.Name)),
		Description:  aws.String(fmt.Sprintf("triggers the associated build targets", projectDoc.Name)),
		State:        aws.String("ENABLED"),
		EventPattern: map[string]interface{}{
			"source": []string{
				eventCtx.BuildJobConfiguration.BuildEventSource,
			},
			"detail-type": []string{
				fmt.Sprintf("%s.%s", eventCtx.BuildJobConfiguration.BuildEventAction, projectDoc.Name),
			},
		},
		Targets: []events.Rule_Target{
			events.Rule_Target{
				Arn:     eventCtx.BuildJobConfiguration.BuildJobQueueArn,
				Id:      "battleshiper-project-build-queue",
				RoleArn: aws.String(goformation.GetAtt("BuildRuleRole", "Arn")),
				BatchParameters: &events.Rule_BatchParameters{
					JobDefinition: goformation.Ref("BuildJobDefinition"),
					JobName:       fmt.Sprintf("battleshiper-project-build-job-%s", projectDoc.Name),
				},
				InputTransformer: &events.Rule_InputTransformer{
					InputPathsMap: map[string]string{
						"EXECUTION_IDENTIFIER":    "$.detail.execution_identifier",
						"REPOSITORY_URL":          "$.detail.repository_url",
						"BUILD_COMMAND":           "$.detail.build_command",
						"BUILD_ASSET_BUCKET_PATH": "$.detail.build_asset_bucket_path",
						"DEPLOY_EVENTBUS":         "$.detail.deploy_endpoint.eventbus",
						"DEPLOY_SOURCE":           "$.detail.deploy_endpoint.source",
						"DEPLOY_ACTION":           "$.detail.deploy_endpoint.action",
						"DEPLOY_TICKET":           "$.detail.deploy_endpoint.ticket",
					},
					InputTemplate: "{\"containerOverrides\": {\"environment\": [{" +
						"\"EXECUTION_IDENTIFIER\": \"EXECUTION_IDENTIFIER\"," +
						"\"REPOSITORY_URL\": \"REPOSITORY_URL\"," +
						"\"BUILD_COMMAND\": \"BUILD_COMMAND\"," +
						"\"BUILD_ASSET_BUCKET_PATH\": \"BUILD_ASSET_BUCKET_PATH\"," +
						"\"DEPLOY_EVENTBUS\": \"DEPLOY_EVENTBUS\"," +
						"\"DEPLOY_SOURCE\": \"DEPLOY_SOURCE\"," +
						"\"DEPLOY_ACTION\": \"DEPLOY_ACTION\"," +
						"\"DEPLOY_TICKET\": \"DEPLOY_TICKET\"," +
						"}]}}",
				},
			},
		},
	}

	// TODO: Add Job execution ROLE (can only send detail-event battleshiper.deploy with source ch.megakuul.battleshiper.projectDoc.Name)

	stackTemplate.Resources["BuildJobDefinition"] = &batch.JobDefinition{
		JobDefinitionName: aws.String(fmt.Sprintf("battleshiper-project-build-job-%s", projectDoc.Name)),
		Type:              "container",
		ContainerProperties: &batch.JobDefinition_ContainerProperties{
			Image:  projectDoc.BuildImage,
			Vcpus:  aws.Int(eventCtx.BuildJobConfiguration.BuildJobVCPUS),
			Memory: aws.Int(eventCtx.BuildJobConfiguration.BuildJobMemory),
			Command: []string{
				"sh", "-c",
			},
			NetworkConfiguration: &batch.JobDefinition_NetworkConfiguration{
				AssignPublicIp: aws.String("ENABLED"),
			},
		},
		Timeout: &batch.JobDefinition_Timeout{
			AttemptDurationSeconds: aws.Int(int(eventCtx.BuildJobConfiguration.BuildJobTimeout.Seconds())),
		},
	}

	// - create stack
	// - add event rule sending the request to the batch
	// - setup aws batch with permissions to insert to the asset-bucket and to send to battleshiper.deploy

	stackName := fmt.Sprintf("battleshiper-project-stack-%s", projectDoc.Name)
	stackBody, err := stackTemplate.JSON()
	if err != nil {
		return fmt.Errorf("failed to parse cloudformation stack body")
	}
	_, err = eventCtx.CloudformationClient.CreateStack(transportCtx, &cloudformation.CreateStackInput{
		StackName:    aws.String(stackName),
		TemplateBody: aws.String(string(stackBody)),
	})
	if err != nil {
		return fmt.Errorf("failed to create cloudformation stack: %v", err)
	}

	projectCollection := eventCtx.Database.Collection(project.PROJECT_COLLECTION)

	result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
		"$set": bson.M{
			"dedicated_infrastructure.stack_name": stackName,
		},
	})
	if err != nil || result.MatchedCount < 1 {
		_, err := eventCtx.CloudformationClient.DeleteStack(transportCtx, &cloudformation.DeleteStackInput{
			StackName:    aws.String(stackName),
			DeletionMode: types.DeletionModeStandard,
		})
		if err != nil {
			log.Printf("ERROR RUNTIME: failed to delete stack '%s'. failed to reference stack in database; stack is leaking.\n", stackName)
		}
		return fmt.Errorf("failed to update project on database")
	}

	waiter := cloudformation.NewStackCreateCompleteWaiter(eventCtx.CloudformationClient)
	err = waiter.Wait(transportCtx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}, eventCtx.DeploymentTimeout)
	if err != nil {
		return fmt.Errorf("failed to apply cloudformation stack: %v", err)
	}

	return nil
}
