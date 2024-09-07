package eventcontext

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"go.mongodb.org/mongo-driver/mongo"
)

// Context provides data to event handlers.
type Context struct {
	Database              *mongo.Database
	TicketOptions         *pipeline.TicketOptions
	CloudformationClient  *cloudformation.Client
	DeploymentTimeout     time.Duration
	S3Client              *s3.Client
	CloudwatchClient      *cloudwatchlogs.Client
	CloudfrontCacheClient *cloudfrontkeyvaluestore.Client
	CloudfrontCacheArn    string
}
