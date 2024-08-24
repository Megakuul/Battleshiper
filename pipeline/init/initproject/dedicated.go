package initproject

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	goformation "github.com/awslabs/goformation/v7/cloudformation"
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
	}, eventCtx.CloudformationTimeout)
	if err != nil {
		return fmt.Errorf("failed to apply cloudformation stack: %v", err)
	}

	return nil
	// - create stack
	// - add event rule sending the request to the batch
	// - setup aws batch with permissions to insert to the asset-bucket and to send to battleshiper.deploy
}
