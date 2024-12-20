package eventcontext

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
)

type DeploymentConfiguration struct {
	ChangeSetTimeout  time.Duration
	DeplyomentTimeout time.Duration
}

type ProjectConfiguration struct {
	ServerNamePrefix         string
	ServerRuntime            string
	ServerMemory             int
	ServerTimeout            int
	CloudfrontDistributionId string
	CloudfrontCacheArn       string
}

// Context provides data to event handlers.
type Context struct {
	DynamoClient            *dynamodb.Client
	UserTable               string
	ProjectTable            string
	SubscriptionTable       string
	TicketOptions           *pipeline.TicketOptions
	CloudformationClient    *cloudformation.Client
	S3Client                *s3.Client
	CloudwatchClient        *cloudwatchlogs.Client
	CloudfrontClient        *cloudfront.Client
	CloudfrontCacheClient   *cloudfrontkeyvaluestore.Client
	DeploymentConfiguration *DeploymentConfiguration
	ProjectConfiguration    *ProjectConfiguration
}
