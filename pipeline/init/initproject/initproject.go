package initproject

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

func HandleInitProject(
	request events.CloudWatchEvent,
	transportCtx context.Context,
	eventCtx eventcontext.Context) func(context.Context, events.CloudWatchEvent) error {

	return func(ctx context.Context, event events.CloudWatchEvent) error {
		err := runHandleInitProject(request, transportCtx, eventCtx)
		if err != nil {
			log.Printf("ERROR INITPROJECT: %v\n", err)
			return err
		}
		return nil
	}
}

func runHandleInitProject(request events.CloudWatchEvent, transportCtx context.Context, eventCtx eventcontext.Context) error {

}
