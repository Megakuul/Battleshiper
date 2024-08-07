package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"go.mongodb.org/mongo-driver/mongo"
)

// Context provides data to route handlers.
type Context struct {
	CloudWatchClient *cloudwatchlogs.Client
	JwtOptions       *auth.JwtOptions
	Database         *mongo.Database
}
