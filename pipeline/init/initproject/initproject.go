package initproject

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
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

	// TODO:
	// - read project
	// - add api routes (only determine and insert to database)
	// - add bucket suffixes (only determine and insert to database)
	// - create stack
	// - add event rule sending the request to the batch
	// - setup aws batch with permissions to insert to the asset-bucket and to send to battleshiper.deploy
}
