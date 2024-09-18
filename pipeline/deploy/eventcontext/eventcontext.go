package eventcontext

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
)

type DeploymentConfiguration struct {
	Timeout time.Duration
}

type ProjectConfiguration struct {
	ServerNamePrefix   string
	ServerRuntime      string
	ServerMemory       int
	ServerTimeout      int
	CloudfrontCacheArn string
}

// Context provides data to event handlers.
type Context struct {
	TicketOptions           *pipeline.TicketOptions
	CloudformationClient    *cloudformation.Client
	S3Client                *s3.Client
	CloudwatchClient        *cloudwatchlogs.Client
	CloudfrontCacheClient   *cloudfrontkeyvaluestore.Client
	DeploymentConfiguration *DeploymentConfiguration
	ProjectConfiguration    *ProjectConfiguration
}
