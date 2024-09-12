package routecontext

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Context provides data to route handlers.
type Context struct {
	S3Bucket   string
	S3Client   *s3.Client
	HttpSuffix string
	HttpClient *http.Client
}
