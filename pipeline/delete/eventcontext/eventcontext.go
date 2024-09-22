package eventcontext

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
)

type DeletionConfiguration struct {
	Timeout time.Duration
}

type BucketConfiguration struct {
	StaticBucketName string
}

type CloudfrontConfiguration struct {
	CacheArn string
}

// Context provides data to event handlers.
type Context struct {
	DynamoClient            *dynamodb.Client
	ProjectTable            string
	TicketOptions           *pipeline.TicketOptions
	S3Client                *s3.Client
	CloudformationClient    *cloudformation.Client
	CloudfrontCacheClient   *cloudfrontkeyvaluestore.Client
	DeletionConfiguration   *DeletionConfiguration
	BucketConfiguration     *BucketConfiguration
	CloudfrontConfiguration *CloudfrontConfiguration
}
