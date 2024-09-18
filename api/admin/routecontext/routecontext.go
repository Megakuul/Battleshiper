package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/megakuul/battleshiper/lib/helper/auth"
)

type LogConfiguration struct {
	ApiLogGroup      string
	PipelineLogGroup string
}

// Context provides data to route handlers.
type Context struct {
	JwtOptions *auth.JwtOptions

	CloudwatchClient *cloudwatchlogs.Client
	LogConfiguration *LogConfiguration
}
