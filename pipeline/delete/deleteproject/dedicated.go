package deleteproject

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/delete/eventcontext"
)

// deleteStack deletes the dedicated project stack (if existent).
func deleteStack(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	if projectDoc.DedicatedInfrastructure.StackName == "" {
		return nil
	}

	_, err := eventCtx.CloudformationClient.DeleteStack(transportCtx, &cloudformation.DeleteStackInput{
		StackName: aws.String(projectDoc.DedicatedInfrastructure.StackName),
	})
	if err != nil {
		if _, ok := err.(*cloudformationtypes.StackNotFoundException); ok {
			return nil
		}
		return fmt.Errorf("failed to delete stack: %v", err)
	}

	waiter := cloudformation.NewStackDeleteCompleteWaiter(eventCtx.CloudformationClient)

	err = waiter.Wait(transportCtx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(projectDoc.DedicatedInfrastructure.StackName),
	}, eventCtx.DeletionConfiguration.Timeout)
	if err != nil {
		return fmt.Errorf("stack deletion failed: %v", err)
	}

	return nil
}
