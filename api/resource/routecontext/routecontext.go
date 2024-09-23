package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/google/go-github/v63/github"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
)

// Context provides data to route handlers.
type Context struct {
	DynamoClient          *dynamodb.Client
	UserTable             string
	ProjectTable          string
	SubscriptionTable     string
	CloudwatchClient      *cloudwatchlogs.Client
	GithubAppClient       *github.Client
	JwtOptions            *auth.JwtOptions
	EventClient           *eventbridge.Client
	InitEventOptions      *pipeline.EventOptions
	BuildEventOptions     *pipeline.EventOptions
	DeployTicketOptions   *pipeline.TicketOptions
	DeleteEventOptions    *pipeline.EventOptions
	CloudfrontCacheClient *cloudfrontkeyvaluestore.Client
	CloudfrontCacheArn    string
}
