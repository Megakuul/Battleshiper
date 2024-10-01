package deleteproject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/delete/eventcontext"
)

var logger = log.New(os.Stderr, "DELETE DELETEPROJECT: ", 0)

func HandleDeleteProject(eventCtx eventcontext.Context) func(context.Context, events.CloudWatchEvent) error {
	return func(ctx context.Context, event events.CloudWatchEvent) error {
		err := runHandleDeleteProject(event, ctx, eventCtx)
		if err != nil {
			logger.Printf("%v\n", err)
			return err
		}
		return nil
	}
}

func runHandleDeleteProject(request events.CloudWatchEvent, transportCtx context.Context, eventCtx eventcontext.Context) error {
	deleteRequest := &event.DeleteRequest{}
	if err := json.Unmarshal(request.Detail, &deleteRequest); err != nil {
		return fmt.Errorf("failed to deserialize deploy request")
	}

	deleteClaims, err := pipeline.ParseTicket(eventCtx.TicketOptions, deleteRequest.DeleteTicket)
	if err != nil {
		return fmt.Errorf("failed to parse ticket: %v", err)
	}

	if deleteClaims.Action != request.DetailType {
		return fmt.Errorf("action mismatch: provided ticket was not issued for the specified action")
	}

	projectDoc, err := database.GetSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(eventCtx.ProjectTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":project_name": &dynamodbtypes.AttributeValueMemberS{Value: deleteClaims.Project},
		},
		ConditionExpr: aws.String("project_name = :project_name"),
	})
	if err != nil {
		// if the project is not existent, the deletion is considered successful.
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil
		}
		return fmt.Errorf("failed to load project from database: %v", err)
	}
	if projectDoc.OwnerId != deleteClaims.UserID {
		return fmt.Errorf("user '%s' is not authorized to delete this project", deleteClaims.UserID)
	}
	if !projectDoc.Deleted {
		return fmt.Errorf("project is not marked for deletion")
	}

	if err := deleteProject(transportCtx, eventCtx, projectDoc); err != nil {
		if _, err := database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
			Table: aws.String(eventCtx.ProjectTable),
			PrimaryKey: map[string]dynamodbtypes.AttributeValue{
				"project_name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.ProjectName},
			},
			AttributeNames: map[string]string{
				"#status": "status",
			},
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":status": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("DELETION FAILED: %v", err)},
			},
			UpdateExpr: aws.String("SET #status = :status"),
		}); err != nil {
			return fmt.Errorf("failed to update project: %v", err)
		}
		return err
	}

	return nil
}

func deleteProject(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	if projectDoc.PipelineLock {
		return fmt.Errorf("project pipeline is locked")
	}

	if err := deleteStaticAssets(transportCtx, eventCtx, projectDoc); err != nil {
		return err
	}

	if err := deleteAliases(transportCtx, eventCtx, projectDoc.Aliases); err != nil {
		return err
	}

	if err := deleteStaticPageKeys(transportCtx, eventCtx, projectDoc.SharedInfrastructure.PrerenderPageKeys); err != nil {
		return err
	}

	if err := deleteStack(transportCtx, eventCtx, projectDoc); err != nil {
		return err
	}

	if err := database.DeleteSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.DeleteSingleInput{
		Table: aws.String(eventCtx.ProjectTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"project_name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.ProjectName},
		},
	}); err != nil {
		return fmt.Errorf("failed to delete project from database: %v", err)
	}

	return nil
}
