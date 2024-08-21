package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/go-playground/webhooks/v6/github"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"go.mongodb.org/mongo-driver/mongo"
)

// Context provides data to route handlers.
type Context struct {
	Database      *mongo.Database
	WebhookClient *github.Webhook
	TicketOptions *pipeline.TicketOptions
	EventClient   *eventbridge.Client
	EventBus      string
}