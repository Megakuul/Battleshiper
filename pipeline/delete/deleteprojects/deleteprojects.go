package deleteprojects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/delete/eventcontext"
)

func HandleDeleteProjects(eventCtx eventcontext.Context) func(context.Context, events.CloudWatchEvent) error {
	return func(ctx context.Context, event events.CloudWatchEvent) error {
		err := runHandleDeleteProjects(event, ctx, eventCtx)
		if err != nil {
			log.Printf("ERROR DELETEPROJECTS: %v\n", err)
			return err
		}
		return nil
	}
}

func runHandleDeleteProjects(request events.CloudWatchEvent, transportCtx context.Context, eventCtx eventcontext.Context) error {
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
		Table: eventCtx.ProjectTable,
		Index: "",
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":name":     &dynamodbtypes.AttributeValueMemberS{Value: deleteClaims.Project},
			":owner_id": &dynamodbtypes.AttributeValueMemberS{Value: deleteClaims.UserID},
			":deleted":  &dynamodbtypes.AttributeValueMemberBOOL{Value: true},
		},
		ConditionExpr: "name = :name AND owner_id = :owner_id AND deleted = :deleted",
	})
	if err != nil {
		// if the project is not existent, the deletion is considered successful.
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil
		}
		return fmt.Errorf("failed to load project from database")
	}

	if err := deleteProject(transportCtx, eventCtx, projectDoc); err != nil {
		_, err = database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
			Table: eventCtx.ProjectTable,
			PrimaryKey: map[string]dynamodbtypes.AttributeValue{
				"name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.Name},
			},
			AttributeNames: map[string]string{
				"#status": "status",
			},
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":status": &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("DELETION FAILED: %v", err)},
			},
			UpdateExpr: "SET #status = :status",
		})
		if err != nil {
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

	if err := deleteStaticPageKeys(transportCtx, eventCtx, projectDoc.SharedInfrastructure.PrerenderPageKeys); err != nil {
		return err
	}

	if err := deleteStack(transportCtx, eventCtx, projectDoc); err != nil {
		return err
	}

	if err := database.DeleteSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.DeleteSingleInput{
		Table: eventCtx.ProjectTable,
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.Name},
		},
	}); err != nil {
		return fmt.Errorf("failed to delete project from database: %v", err)
	}

	return nil
}
