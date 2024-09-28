package initproject

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	goformation "github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/batch"
	"github.com/awslabs/goformation/v7/cloudformation/events"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/awslabs/goformation/v7/cloudformation/logs"
	"github.com/awslabs/goformation/v7/cloudformation/tags"
	"github.com/megakuul/battleshiper/lib/helper/database"

	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/init/eventcontext"
)

// createStack builds and deploys the initial project stack.
func createStack(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	if err := validateInfrastructureConfiguration(projectDoc); err != nil {
		return fmt.Errorf("failed to validate: %v", err)
	}

	projectDoc.DedicatedInfrastructure = generateDedicatedInfrastructure(eventCtx, projectDoc.Name)

	stackBody := goformation.NewTemplate()
	attachLogSystem(stackBody, eventCtx, projectDoc)
	if err := attachBuildSystem(stackBody, eventCtx, projectDoc); err != nil {
		return fmt.Errorf("failed to serialize build system blueprint")
	}

	stackBodyRaw, err := stackBody.JSON()
	if err != nil {
		return fmt.Errorf("failed to parse cloudformation stack body")
	}

	_, err = eventCtx.CloudformationClient.CreateStack(transportCtx, &cloudformation.CreateStackInput{
		StackName:    aws.String(projectDoc.DedicatedInfrastructure.StackName),
		RoleARN:      aws.String(eventCtx.DeploymentConfiguration.ServiceRoleArn),
		Capabilities: []types.Capability{types.CapabilityCapabilityNamedIam},
		TemplateBody: aws.String(string(stackBodyRaw)),
	})
	if err != nil {
		return fmt.Errorf("failed to create cloudformation stack: %v", err)
	}

	dedicatedInfrastructureAttributes, err := attributevalue.Marshal(&projectDoc.DedicatedInfrastructure)
	if err != nil {
		return fmt.Errorf("failed to serialize dedicated infrastructure")
	}

	_, err = database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(eventCtx.ProjectTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.Name},
		},
		AttributeNames: map[string]string{
			"#dedicated_infrastructure": "dedicated_infrastructure",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":dedicated_infrastructure": dedicatedInfrastructureAttributes,
		},
		UpdateExpr: aws.String("SET #dedicated_infrastructure = :dedicated_infrastructure"),
	})
	if err != nil {
		_, err := eventCtx.CloudformationClient.DeleteStack(transportCtx, &cloudformation.DeleteStackInput{
			StackName:    aws.String(projectDoc.DedicatedInfrastructure.StackName),
			DeletionMode: types.DeletionModeStandard,
		})
		if err != nil {
			log.Printf(
				"ERROR RUNTIME: failed to delete stack '%s'. failed to reference stack in database; stack is leaking.\n",
				projectDoc.DedicatedInfrastructure.StackName,
			)
		}
		return fmt.Errorf("failed to update project on database")
	}

	waiter := cloudformation.NewStackCreateCompleteWaiter(eventCtx.CloudformationClient)
	err = waiter.Wait(transportCtx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(projectDoc.DedicatedInfrastructure.StackName),
	}, eventCtx.DeploymentConfiguration.Timeout)
	if err != nil {
		return fmt.Errorf("stack creation failed: %v", err)
	}

	return nil
}

// validateInfrastructureConfiguration validates infrastructure options that can, if in a invalid state,
// interfer with the whole battleshiper system.
func validateInfrastructureConfiguration(projectDoc *project.Project) error {
	// if stack name is already present, overwriting can lead to a resource leak.
	if projectDoc.DedicatedInfrastructure.StackName != "" {
		return fmt.Errorf("project already holds a stack")
	}

	// if bucket path is empty, this can potentially lead to unintended behavior in the iam policy.
	if projectDoc.SharedInfrastructure.BuildAssetBucketPath == "" {
		return fmt.Errorf("invalid build asset bucket path")
	}

	return nil
}

// generateDedicatedInfrastructure generates the initial dedicated infrastructure config based on eventCtx and the projectName.
func generateDedicatedInfrastructure(eventCtx eventcontext.Context, projectName string) project.DedicatedInfrastructure {
	infrastructure := project.DedicatedInfrastructure{}

	infrastructure.EventLogGroup = fmt.Sprintf("%s/%s", eventCtx.ProjectConfiguration.EventLogPrefix, projectName)
	infrastructure.BuildLogGroup = fmt.Sprintf("%s/%s", eventCtx.ProjectConfiguration.BuildLogPrefix, projectName)
	infrastructure.EventLogGroup = fmt.Sprintf("%s/%s", eventCtx.ProjectConfiguration.DeployLogPrefix, projectName)
	infrastructure.ServerLogGroup = fmt.Sprintf("%s/%s", eventCtx.ProjectConfiguration.ServerLogPrefix, projectName)

	infrastructure.StackName = fmt.Sprintf("battleshiper-project-stack-%s", projectName)

	return infrastructure
}

// attachLogSystem adds the projects logsystem to the stack.
func attachLogSystem(stackTemplate *goformation.Template, eventCtx eventcontext.Context, projectDoc *project.Project) {
	const EVENT_LOG_GROUP string = "EventLogGroup"
	stackTemplate.Resources[EVENT_LOG_GROUP] = &logs.LogGroup{
		LogGroupName:    aws.String(projectDoc.DedicatedInfrastructure.EventLogGroup),
		RetentionInDays: aws.Int(eventCtx.ProjectConfiguration.LogRetentionDays),
	}

	const DEPLOY_LOG_GROUP string = "DeployLogGroup"
	stackTemplate.Resources[DEPLOY_LOG_GROUP] = &logs.LogGroup{
		LogGroupName:    aws.String(projectDoc.DedicatedInfrastructure.DeployLogGroup),
		RetentionInDays: aws.Int(eventCtx.ProjectConfiguration.LogRetentionDays),
	}

	const SERVER_LOG_GROUP string = "ServerLogGroup"
	stackTemplate.Resources[SERVER_LOG_GROUP] = &logs.LogGroup{
		LogGroupName:    aws.String(projectDoc.DedicatedInfrastructure.ServerLogGroup),
		RetentionInDays: aws.Int(eventCtx.ProjectConfiguration.LogRetentionDays),
	}
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

// attachBuildSystem adds the project pipeline build system to the stack.
func attachBuildSystem(stackTemplate *goformation.Template, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	const BUILD_LOG_GROUP string = "BuildLogGroup"
	stackTemplate.Resources[BUILD_LOG_GROUP] = &logs.LogGroup{
		LogGroupName:    aws.String(projectDoc.DedicatedInfrastructure.BuildLogGroup),
		RetentionInDays: aws.Int(eventCtx.ProjectConfiguration.LogRetentionDays),
	}

	const BUILD_JOB_EXEC_ROLE string = "BuildJobExecRole"
	stackTemplate.Resources[BUILD_JOB_EXEC_ROLE] = &iam.Role{
		Tags: []tags.Tag{
			{Value: "Name", Key: fmt.Sprintf("battleshiper-project-build-job-exec-role-%s", projectDoc.Name)},
		},
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
				PolicyName: fmt.Sprintf("battleshiper-pipeline-build-log-%s-exec-access", projectDoc.Name),
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
		Tags: []tags.Tag{
			{Value: "Name", Key: fmt.Sprintf("battleshiper-project-build-job-role-%s", projectDoc.Name)},
		},
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
				PolicyName: fmt.Sprintf("battleshiper-pipeline-build-asset-bucket-%s-write-access", projectDoc.Name),
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
			Vcpus:            aws.Int(eventCtx.ProjectConfiguration.BuildJobVCPUS),
			Memory:           aws.Int(eventCtx.ProjectConfiguration.BuildJobMemory),
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
				"git clone --branch $REPOSITORY_BRANCH $REPOSITORY_URL . &&",
				"$BUILD_COMMAND &&",
				"aws s3 cp $OUTPUT_DIRECTORY s3://$BUILD_ASSET_BUCKET_PATH/$EXECUTION_IDENTIFIER --recursive",
			},
			NetworkConfiguration: &batch.JobDefinition_NetworkConfiguration{
				AssignPublicIp: aws.String("ENABLED"),
			},
		},
		Timeout: &batch.JobDefinition_Timeout{
			AttemptDurationSeconds: aws.Int(int(eventCtx.ProjectConfiguration.BuildJobTimeout.Seconds())),
		},
	}

	const BUILD_RULE_ROLE string = "BuildRuleRole"
	stackTemplate.Resources[BUILD_RULE_ROLE] = &iam.Role{
		Tags: []tags.Tag{
			{Value: "Name", Key: fmt.Sprintf("battleshiper-project-build-rule-role-%s", projectDoc.Name)},
		},
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
		Policies: []iam.Role_Policy{
			{
				PolicyName: fmt.Sprintf("battleshiper-pipeline-build-queue-%s-exec-access", projectDoc.Name),
				PolicyDocument: map[string]interface{}{
					"Version": "2012-10-17",
					"Statement": []map[string]interface{}{
						{
							"Effect":   "Allow",
							"Action":   "batch:SubmitJob",
							"Resource": eventCtx.ProjectConfiguration.BuildJobQueueArn,
						},
					},
				},
			},
		},
	}

	inputPathsMap := map[string]string{
		"TMPL_EXECUTION_IDENTIFIER": "$.detail.execution_identifier",
		"TMPL_DEPLOY_TICKET":        "$.detail.deploy_ticket",
		"TMPL_REPOSITORY_URL":       "$.detail.repository_url",
		"TMPL_REPOSITORY_BRANCH":    "$.detail.repository_branch",
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
					Name:  "REPOSITORY_BRANCH",
					Value: "<TMPL_REPOSITORY_BRANCH>",
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
		EventBusName: aws.String(eventCtx.ProjectConfiguration.BuildEventbusName),
		Name:         aws.String(fmt.Sprintf("battleshiper-project-build-rule-%s", projectDoc.Name)),
		Description:  aws.String("triggers the associated build targets"),
		State:        aws.String("ENABLED"),
		EventPattern: map[string]interface{}{
			"source": []string{
				eventCtx.ProjectConfiguration.BuildEventSource,
			},
			"detail-type": []string{
				fmt.Sprintf("%s.%s", eventCtx.ProjectConfiguration.BuildEventAction, projectDoc.Name),
			},
		},
		Targets: []events.Rule_Target{
			{
				Arn:     eventCtx.ProjectConfiguration.BuildJobQueueArn,
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
