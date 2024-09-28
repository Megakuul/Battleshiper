package initproject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/init/eventcontext"
)

func HandleInitProject(eventCtx eventcontext.Context) func(context.Context, events.CloudWatchEvent) error {
	return func(ctx context.Context, event events.CloudWatchEvent) error {
		err := runHandleInitProject(event, ctx, eventCtx)
		if err != nil {
			log.Printf("ERROR INITPROJECT: %v\n", err)
			return err
		}
		return nil
	}
}

func runHandleInitProject(request events.CloudWatchEvent, transportCtx context.Context, eventCtx eventcontext.Context) error {
	initRequest := &event.InitRequest{}
	if err := json.Unmarshal(request.Detail, &initRequest); err != nil {
		return fmt.Errorf("failed to deserialize init request")
	}

	initClaims, err := pipeline.ParseTicket(eventCtx.TicketOptions, initRequest.InitTicket)
	if err != nil {
		return fmt.Errorf("failed to parse ticket: %v", err)
	}

	if initClaims.Source != request.Source {
		return fmt.Errorf("source mismatch: provided ticket was not issued for this event source")
	}
	if initClaims.Action != request.DetailType {
		return fmt.Errorf("action mismatch: provided ticket was not issued for the specified action")
	}

	projectDoc, err := database.GetSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(eventCtx.ProjectTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":name":     &dynamodbtypes.AttributeValueMemberS{Value: initClaims.Project},
			":owner_id": &dynamodbtypes.AttributeValueMemberS{Value: initClaims.UserID},
		},
		ConditionExpr: aws.String("name = :name AND owner_id = :owner_id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return fmt.Errorf("project not found")
		}
		return fmt.Errorf("failed to load project from database")
	}

	err = initProject(transportCtx, eventCtx, projectDoc)
	if err != nil {
		_, err = database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
			Table: aws.String(eventCtx.ProjectTable),
			PrimaryKey: map[string]dynamodbtypes.AttributeValue{
				"name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.Name},
			},
			AttributeNames: map[string]string{
				"#status":        "status",
				"#pipeline_lock": "pipeline_lock",
			},
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":status":        &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("INITIALIZATION FAILED: %v", err)},
				":pipeline_lock": &dynamodbtypes.AttributeValueMemberBOOL{Value: false},
			},
			UpdateExpr: aws.String("SET #pipeline_lock = :pipeline_lock, #status = :status"),
		})
		if err != nil {
			return fmt.Errorf("failed to update project: %v", err)
		}
		return nil
	}

	_, err = database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(eventCtx.ProjectTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.Name},
		},
		AttributeNames: map[string]string{
			"#initialized":   "initialized",
			"#status":        "status",
			"#pipeline_lock": "pipeline_lock",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":initialized":   &dynamodbtypes.AttributeValueMemberBOOL{Value: true},
			":status":        &dynamodbtypes.AttributeValueMemberS{Value: ""},
			":pipeline_lock": &dynamodbtypes.AttributeValueMemberBOOL{Value: false},
		},
		UpdateExpr: aws.String("SET #initialized = :initialized, #pipeline_lock = :pipeline_lock, #status = :status"),
	})
	if err != nil {
		return fmt.Errorf("failed to update project: %v", err)
	}
	return nil
}

func initProject(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {

	err := initSharedInfrastructure(transportCtx, eventCtx, projectDoc)
	if err != nil {
		return fmt.Errorf("failed to init shared infrastructure: %v", err)
	}

	err = createStack(transportCtx, eventCtx, projectDoc)
	if err != nil {
		return fmt.Errorf("failed to create dedicated infrastructure: %v", err)
	}

	return nil
}
