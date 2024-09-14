package eventcontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.mongodb.org/mongo-driver/mongo"
)

type BucketConfiguration struct {
	StaticBucketName string
}

type CloudfrontConfiguration struct {
	CacheArn string
}

// Context provides data to event handlers.
type Context struct {
	Database                *mongo.Database
	S3Client                *s3.Client
	CloudformationClient    *cloudformation.Client
	CloudfrontCacheClient   *cloudfrontkeyvaluestore.Client
	BucketConfiguration     *BucketConfiguration
	CloudfrontConfiguration *CloudfrontConfiguration
}
