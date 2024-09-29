package eventcontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type BucketConfiguration struct {
	StaticBucketName string
}

type CloudfrontConfiguration struct {
	DistributionId string
}

// Context provides data to event handlers.
type Context struct {
	CodeDeployClient        *codedeploy.Client
	S3Client                *s3.Client
	BucketConfiguration     *BucketConfiguration
	CloudfrontClient        *cloudfront.Client
	CloudfrontConfiguration *CloudfrontConfiguration
}
