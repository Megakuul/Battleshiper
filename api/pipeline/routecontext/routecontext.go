package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	webhook "github.com/go-playground/webhooks/v6/github"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
)

// Context provides data to route handlers.
type Context struct {
	DynamoClient        *dynamodb.Client
	UserTable           string
	ProjectTable        string
	SubscriptionTable   string
	WebhookClient       *webhook.Webhook
	GithubAppOptions    *auth.GithubAppOptions
	CloudwatchClient    *cloudwatchlogs.Client
	EventClient         *eventbridge.Client
	BuildEventOptions   *pipeline.EventOptions
	DeployTicketOptions *pipeline.TicketOptions
}
