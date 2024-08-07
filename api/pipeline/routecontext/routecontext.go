package routecontext

import (
	"github.com/go-playground/webhooks/v6/github"
	"go.mongodb.org/mongo-driver/mongo"
)

// Context provides data to route handlers.
type Context struct {
	Database      *mongo.Database
	WebhookClient *github.Webhook
}
