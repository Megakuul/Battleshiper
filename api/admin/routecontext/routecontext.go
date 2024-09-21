package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/megakuul/battleshiper/lib/helper/auth"
)

type LogConfiguration struct {
	ApiLogGroup      string
	PipelineLogGroup string
}

// Context provides data to route handlers.
type Context struct {
	DynamoClient      *dynamodb.Client
	UserTable         string
	ProjectTable      string
	SubscriptionTable string
	CloudwatchClient  *cloudwatchlogs.Client
	JwtOptions        *auth.JwtOptions
	LogConfiguration  *LogConfiguration
}
