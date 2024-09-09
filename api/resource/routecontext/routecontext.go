package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"go.mongodb.org/mongo-driver/mongo"
)

// Context provides data to route handlers.
type Context struct {
	CloudwatchClient      *cloudwatchlogs.Client
	JwtOptions            *auth.JwtOptions
	Database              *mongo.Database
	EventClient           *eventbridge.Client
	InitEventOptions      *pipeline.EventOptions
	BuildEventOptions     *pipeline.EventOptions
	DeployTicketOptions   *pipeline.TicketOptions
	CloudfrontCacheClient *cloudfrontkeyvaluestore.Client
	CloudfrontCacheArn    string
}
