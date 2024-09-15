package initproject

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
	"go.mongodb.org/mongo-driver/bson"
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

	projectCollection := eventCtx.Database.Collection(project.PROJECT_COLLECTION)

	projectDoc := &project.Project{}
	err = projectCollection.FindOne(transportCtx, bson.D{
		{Key: "name", Value: initClaims.Project},
		{Key: "owner_id", Value: initClaims.UserID},
	}).Decode(&projectDoc)
	if err != nil {
		return fmt.Errorf("failed to fetch project from database")
	}

	err = initProject(transportCtx, eventCtx, projectDoc)
	if err != nil {
		result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
			"$set": bson.M{
				"status":        fmt.Errorf("INITIALIZATION FAILED: %v", err),
				"pipeline_lock": false,
			},
		})
		if err != nil || result.MatchedCount < 1 {
			return fmt.Errorf("failed to update project on database")
		}
		return nil
	}

	result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
		"$set": bson.M{
			"initialized":   true,
			"status":        "",
			"pipeline_lock": false,
		},
	})
	if err != nil || result.MatchedCount < 1 {
		return fmt.Errorf("failed to update project on database")
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
