package deleteproject

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/smithy-go"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/delete/eventcontext"
)

// deleteStack deletes the dedicated project stack (if existent).
func deleteStack(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	if projectDoc.DedicatedInfrastructure.StackName == "" {
		return nil
	}

	_, err := eventCtx.CloudformationClient.DescribeStacks(transportCtx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(projectDoc.DedicatedInfrastructure.StackName),
	})
	if err != nil {
		// This is again the result of the aws crap api that does not provide a simple way to see if a stack exists or not.
		// Issue described here: https://github.com/aws/aws-sdk-go-v2/issues/2296. Behavior described here: https://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/API_DescribeStacks.html
		var sErr smithy.APIError
		if errors.As(err, &sErr) && sErr.ErrorCode() == "ValidationError" {
			return nil
		}
		return fmt.Errorf("failed to describe stack: %v", err)
	}

	_, err = eventCtx.CloudformationClient.DeleteStack(transportCtx, &cloudformation.DeleteStackInput{
		StackName: aws.String(projectDoc.DedicatedInfrastructure.StackName),
	})
	if err != nil {
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
