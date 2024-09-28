package eventcontext

import (
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type BucketConfiguration struct {
	StaticBucketName string
}

// Context provides data to event handlers.
type Context struct {
	CodeDeployClient    *codedeploy.Client
	S3Client            *s3.Client
	BucketConfiguration *BucketConfiguration
}
