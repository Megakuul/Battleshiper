package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Context provides data to route handlers.
type Context struct {
	S3Client         *s3.Client
	StaticBucketName string
	FunctionClient   *lambda.Client
	ServerNamePrefix string
	ErrorPage        string
}
