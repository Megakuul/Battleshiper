package deployproject

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	goform "github.com/awslabs/goformation/v7"
	goformation "github.com/awslabs/goformation/v7/cloudformation"

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
	case cloudformationtypes.StackStatusRollbackComplete:
	case cloudformationtypes.StackStatusUpdateRollbackComplete:
		return nil
	default:
		return fmt.Errorf("detected non-updateable stack state (%s): %s", stack.StackStatus, *stack.StackStatusReason)
	}

	return nil
}

// createChangeSet loads the current stack, builds a changeset with the new system and pushes the change set to cloudformation.
func createChangeSet(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, execIdentifier string) (string, error) {
	stackTemplate, err := eventCtx.CloudformationClient.GetTemplate(transportCtx, &cloudformation.GetTemplateInput{
		StackName: aws.String(projectDoc.DedicatedInfrastructure.StackName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch stack template: %v", err)
	}
	stackBody, err := goform.ParseJSON([]byte(*stackTemplate.TemplateBody))
	if err != nil {
		return "", fmt.Errorf("failed to parse stack template: %v", err)
	}

	attachServerSystem(stackBody, eventCtx, projectDoc)

	stackBodyRaw, err := stackBody.JSON()
	if err != nil {
		return "", fmt.Errorf("failed to serialize stack template: %v", err)
	}

	_, err = eventCtx.CloudformationClient.ValidateTemplate(transportCtx, &cloudformation.ValidateTemplateInput{
		TemplateBody: aws.String(string(stackBodyRaw)),
	})
	if err != nil {
		return "", fmt.Errorf("failed to validate stack template: %v", err)
	}

	changeSetName := fmt.Sprintf("deployment-%s", execIdentifier)
	_, err = eventCtx.CloudformationClient.CreateChangeSet(transportCtx, &cloudformation.CreateChangeSetInput{
		StackName:     aws.String(projectDoc.DedicatedInfrastructure.StackName),
		ChangeSetName: aws.String(changeSetName),
		TemplateBody:  aws.String(string(stackBodyRaw)),
		Capabilities:  []cloudformationtypes.Capability{cloudformationtypes.CapabilityCapabilityIam},
		ChangeSetType: cloudformationtypes.ChangeSetTypeUpdate,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create changeset: %v", err)
	}

	return changeSetName, nil
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

	changeDescription := ""
	for _, change := range changeSet.Changes {
		changeDescription += fmt.Sprintf(
			"%s: %s",
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
	stackOutput, err := waiter.WaitForOutput(transportCtx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(projectDoc.DedicatedInfrastructure.StackName),
	}, eventCtx.DeploymentTimeout)

	return nil
}

// attachServerSystem adds the project server system to the stack.
func attachServerSystem(stackTemplate *goformation.Template, eventCtx eventcontext.Context, projectDoc *project.Project) {

}
