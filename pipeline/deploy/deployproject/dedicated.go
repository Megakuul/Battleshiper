package deployproject

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	goform "github.com/awslabs/goformation/v7"
	goformation "github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/awslabs/goformation/v7/cloudformation/lambda"
	"github.com/awslabs/goformation/v7/intrinsics"

	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

// validateStackState ensures that the stack is in a consistent state which can be updated.
func validateStackState(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	stackState, err := eventCtx.CloudformationClient.DescribeStacks(transportCtx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(projectDoc.DedicatedInfrastructure.StackName),
	})
	if err != nil || len(stackState.Stacks) < 1 {
		return fmt.Errorf("failed to fetch stack state: %v", err)
	}
	stack := stackState.Stacks[0]
	switch stack.StackStatus {
	case cloudformationtypes.StackStatusCreateComplete:
	case cloudformationtypes.StackStatusUpdateComplete:
	case cloudformationtypes.StackStatusUpdateRollbackComplete:
		return nil
	default:
		return fmt.Errorf("detected non-updateable stack state (%s): %s", stack.StackStatus, *stack.StackStatusReason)
	}

	return nil
}

// createChangeSet loads the current stack, builds a changeset with the new system and pushes the change set to cloudformation.
func createChangeSet(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, execId string, serverAsset ObjectDescription) (string, error) {
	stackTemplate, err := eventCtx.CloudformationClient.GetTemplate(transportCtx, &cloudformation.GetTemplateInput{
		StackName: aws.String(projectDoc.DedicatedInfrastructure.StackName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch stack template: %v", err)
	}

	// By default goformation tries to resolve intrinsic functions (e.g. Fn::GetAtt),
	// this process makes no sense in this usecase. Besides that, resolving "Fn::GetAtt" is currently not even supported.
	// Therefore a custom json handler is used, which essentially just leaves the intrinsics as they where (expected behavior).
	stackBody, err := goform.ParseJSONWithOptions([]byte(*stackTemplate.TemplateBody), &intrinsics.ProcessorOptions{
		EvaluateConditions: false,
		IntrinsicHandlerOverrides: map[string]intrinsics.IntrinsicHandler{
			"Fn::Base64":      intrinsicJsonHandler,
			"Fn::And":         intrinsicJsonHandler,
			"Fn::Equals":      intrinsicJsonHandler,
			"Fn::If":          intrinsicJsonHandler,
			"Fn::Not":         intrinsicJsonHandler,
			"Fn::Or":          intrinsicJsonHandler,
			"Fn::FindInMap":   intrinsicJsonHandler,
			"Fn::GetAtt":      intrinsicJsonHandler,
			"Fn::GetAZs":      intrinsicJsonHandler,
			"Fn::ImportValue": intrinsicJsonHandler,
			"Fn::Join":        intrinsicJsonHandler,
			"Fn::Select":      intrinsicJsonHandler,
			"Fn::Split":       intrinsicJsonHandler,
			"Fn::Sub":         intrinsicJsonHandler,
			"Ref":             intrinsicJsonHandler,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to parse stack template: %v", err)
	}

	attachServerSystem(stackBody, eventCtx, projectDoc, serverAsset.SourceBucket, serverAsset.SourceKey)

	stackBodyRaw, err := stackBody.JSON()
	if err != nil {
		return "", fmt.Errorf("failed to serialize stack template: %v", err)
	}

	changeSetName := fmt.Sprintf("deployment-%s", execId)
	_, err = eventCtx.CloudformationClient.CreateChangeSet(transportCtx, &cloudformation.CreateChangeSetInput{
		StackName:     aws.String(projectDoc.DedicatedInfrastructure.StackName),
		ChangeSetName: aws.String(changeSetName),
		TemplateBody:  aws.String(string(stackBodyRaw)),
		Capabilities:  []cloudformationtypes.Capability{cloudformationtypes.CapabilityCapabilityIam, cloudformationtypes.CapabilityCapabilityNamedIam},
		ChangeSetType: cloudformationtypes.ChangeSetTypeUpdate,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create changeset: %v", err)
	}

	waiter := cloudformation.NewChangeSetCreateCompleteWaiter(eventCtx.CloudformationClient)

	err = waiter.Wait(transportCtx, &cloudformation.DescribeChangeSetInput{
		StackName:     aws.String(projectDoc.DedicatedInfrastructure.StackName),
		ChangeSetName: aws.String(changeSetName),
	}, eventCtx.DeploymentConfiguration.ChangeSetTimeout)
	if err != nil {
		return "", fmt.Errorf("failed to wait for changeset completion: %v", err)
	}

	return changeSetName, nil
}

// intrinsicJsonHandler is a goformation handler that does not modify intrinsic functions (like Fn::GetAtt)
// it essentially converts the intrinsic into a json object ({"Fn::GetAtt": "Role.Arn"}).
// The returned interface is base64 encoded as this is required by the goformation process.
func intrinsicJsonHandler(intrinsic string, value interface{}, template interface{}) interface{} {
	intrinsicFunc := map[string]interface{}{
		intrinsic: value,
	}
	intrinsicFuncRaw, err := json.Marshal(intrinsicFunc)
	if err != nil {
		return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(
			"{\"%s\":\"%s\"}", intrinsic, "-- failed to marshal intrinsic --",
		)))
	}
	return base64.StdEncoding.EncodeToString(intrinsicFuncRaw)
}

// describeChangeSet generates a informational string based on the changes done in the changeset.
func describeChangeSet(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, changeSetName string) (string, error) {
	changeSet, err := eventCtx.CloudformationClient.DescribeChangeSet(transportCtx, &cloudformation.DescribeChangeSetInput{
		StackName:     aws.String(projectDoc.DedicatedInfrastructure.StackName),
		ChangeSetName: aws.String(changeSetName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe changeset: %v", err)
	}

	changeDescription := "Infrastructure Changes:"
	for _, change := range changeSet.Changes {
		changeDescription += fmt.Sprintf(
			" [ %s: %s ] ",
			change.ResourceChange.Action,
			*change.ResourceChange.ResourceType,
		)
	}

	return changeDescription, nil
}

// executeChangeSet executes the provided changeset and waits until the stack is updated.
func executeChangeSet(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, changeSetName string) error {
	_, err := eventCtx.CloudformationClient.ExecuteChangeSet(transportCtx, &cloudformation.ExecuteChangeSetInput{
		StackName:     aws.String(projectDoc.DedicatedInfrastructure.StackName),
		ChangeSetName: aws.String(changeSetName),
	})
	if err != nil {
		if _, err := eventCtx.CloudformationClient.DeleteChangeSet(transportCtx, &cloudformation.DeleteChangeSetInput{
			StackName:     aws.String(projectDoc.DedicatedInfrastructure.StackName),
			ChangeSetName: aws.String(changeSetName),
		}); err != nil {
			return fmt.Errorf("failed to delete failed changeset: %v", err)
		}
		return fmt.Errorf("failed to execute changeset: %v", err)
	}

	waiter := cloudformation.NewStackUpdateCompleteWaiter(eventCtx.CloudformationClient)
	err = waiter.Wait(transportCtx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(projectDoc.DedicatedInfrastructure.StackName),
	}, eventCtx.DeploymentConfiguration.DeplyomentTimeout)
	if err != nil {
		return fmt.Errorf("failed to wait for update completion: %v", err)
	}

	return nil
}

// attachServerSystem adds the project server system to the stack.
func attachServerSystem(stackTemplate *goformation.Template, eventCtx eventcontext.Context, projectDoc *project.Project, serverBucketName, serverBucketKey string) {
	// ServerLogGroup is deployed at initialization (combined with all other log groups)
	const SERVER_LOG_GROUP string = "ServerLogGroup"

	const SERVER_FUNCTION_ROLE = "ServerFunctionRole"
	stackTemplate.Resources[SERVER_FUNCTION_ROLE] = &iam.Role{
		RoleName:    aws.String(fmt.Sprintf("battleshiper-project-server-job-exec-role-%s", projectDoc.ProjectName)),
		Description: aws.String("role associated with the battleshiper server function"),
		AssumeRolePolicyDocument: map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Service": "lambda.amazonaws.com",
					},
					"Action": "sts:AssumeRole",
				},
			},
		},
		Policies: []iam.Role_Policy{
			{
				PolicyName: fmt.Sprintf("battleshiper-pipeline-server-log-%s-exec-access", projectDoc.ProjectName),
				PolicyDocument: map[string]interface{}{
					"Version": "2012-10-17",
					"Statement": []map[string]interface{}{
						{
							"Effect": "Allow",
							"Action": []string{
								"logs:CreateLogStream",
								"logs:PutLogEvents",
							},
							"Resource": goformation.GetAtt((SERVER_LOG_GROUP), "Arn"),
						},
					},
				},
			},
		},
	}

	const SERVER_FUNCTION string = "ServerFunction"
	stackTemplate.Resources[SERVER_FUNCTION] = &lambda.Function{
		FunctionName:  aws.String(fmt.Sprintf("%s%s", eventCtx.ProjectConfiguration.ServerNamePrefix, projectDoc.ProjectName)),
		Description:   aws.String(fmt.Sprintf("Server backend for battleshiper project %s", projectDoc.ProjectName)),
		Architectures: []string{"x86_64"},
		Runtime:       aws.String(eventCtx.ProjectConfiguration.ServerRuntime),
		MemorySize:    aws.Int(eventCtx.ProjectConfiguration.ServerMemory),
		Timeout:       aws.Int(eventCtx.ProjectConfiguration.ServerTimeout),
		Role:          goformation.GetAtt(SERVER_FUNCTION_ROLE, "Arn"),
		Code: &lambda.Function_Code{
			S3Bucket: aws.String(serverBucketName),
			S3Key:    aws.String(serverBucketKey),
		},
		Handler: aws.String("index.handler"),
		LoggingConfig: &lambda.Function_LoggingConfig{
			LogGroup:  aws.String(goformation.Ref(SERVER_LOG_GROUP)),
			LogFormat: aws.String("Text"),
		},
	}
}
