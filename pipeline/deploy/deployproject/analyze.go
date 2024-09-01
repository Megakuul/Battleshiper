package deployproject

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	SERVER_PATH    = "server"
	CLIENT_PATH    = "client"
	PRERENDER_PATH = "prerendered"
)

type BuildInformation struct {
	FunctionPath    string
	StaticAssetPath string
	StaticPagePaths []string
}

func analyzeBuildAssets(transportCtx context.Context, storageClient *s3.Client, bucketPath string, maxFunctionStorage, maxAssetStorage, maPageStorage int64) (*BuildInformation, error) {
	bucketPathSegments := strings.SplitN(bucketPath, "/", 2)
	if len(bucketPathSegments) != 2 {
		return nil, fmt.Errorf("failed to decode asset bucket path")
	}
	bucketName := bucketPathSegments[0]
	bucketPrefix := bucketPathSegments[1]

	result, err := storageClient.HeadObject(transportCtx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(),
	})
}
