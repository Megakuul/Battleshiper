package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"go.mongodb.org/mongo-driver/mongo"
)

type LogConfiguration struct {
	ApiLogGroup      string
	PipelineLogGroup string
}

// Context provides data to route handlers.
type Context struct {
	JwtOptions       *auth.JwtOptions
	Database         *mongo.Database
	CloudwatchClient *cloudwatchlogs.Client
	LogConfiguration *LogConfiguration
}
