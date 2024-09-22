package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
)

type LogConfiguration struct {
	ApiLogGroup      string
	PipelineLogGroup string
}

// Context provides data to route handlers.
type Context struct {
	DynamoClient       *dynamodb.Client
	UserTable          string
	ProjectTable       string
	SubscriptionTable  string
	JwtOptions         *auth.JwtOptions
	EventClient        *eventbridge.Client
	DeleteEventOptions *pipeline.EventOptions
	CloudwatchClient   *cloudwatchlogs.Client
	LogConfiguration   *LogConfiguration
}
