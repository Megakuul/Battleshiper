package deployproject

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

const (
	SERVER_PATH    = "server"
	CLIENT_PATH    = "client"
	PRERENDER_PATH = "prerendered"
)

// ObjectDescription provides information about the location of an s3 object.
type ObjectDescription struct {
	ObjectName string
	// Key relative to the objects logical root
	// e.g. build_asset_bucket/x/123/client/_app/static/myimage.png = _app/static/myimage.png.
	RelativeKey string
	// Full source bucket path of the object (including key).
	SourcePath string
}

// BuildInformation provides information about the content of the build output.
type BuildInformation struct {
	ServerObject       ObjectDescription
	ClientObjects      []ObjectDescription
	PrerenderedObjects []ObjectDescription
	PrerenderedPages   []string
}

// analyzeBuildAssets analyzes the content of the build assets, expecting to find sveltekit build output from adapter-battleshiper.
func analyzeBuildAssets(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, execIdentifier string, maxClientBytes, maxPrerenderBytes, maxServerBytes int64) (*BuildInformation, error) {
	bucketPathSegments := strings.SplitN(projectDoc.SharedInfrastructure.BuildAssetBucketPath, "/", 2)
	if len(bucketPathSegments) != 2 {
		return nil, fmt.Errorf("failed to decode build asset bucket path")
	}
	bucketName := bucketPathSegments[0]
	bucketPrefix := fmt.Sprintf("%s/", bucketPathSegments[1])

	clientObjects, err := analyzeClientObjects(transportCtx, eventCtx.S3Client, bucketName, bucketPrefix, execIdentifier, maxClientBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze client assets: %v", err)
	}

	prerenderObjects, err := analyzePrerenderObjects(transportCtx, eventCtx.S3Client, bucketName, bucketPrefix, execIdentifier, maxClientBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze prerendered assets: %v", err)
	}

	// TODO add serverObject analysis + extract objectName
}

func analyzeClientObjects(transportCtx context.Context, s3Client *s3.Client, bucketName, bucketPrefix, execIdentifier string, maxBytes int64) ([]ObjectDescription, error) {
	clientPrefix := fmt.Sprintf("%s/%s", execIdentifier, CLIENT_PATH)
	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(fmt.Sprintf("%s%s", bucketPrefix, clientPrefix)),
	})

	var clientSize int64 = 0
	clientObjects := []ObjectDescription{}

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(transportCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %v", err)
		}

		for _, obj := range page.Contents {
			clientSize += *obj.Size
			if clientSize > maxBytes {
				return nil, fmt.Errorf("exceeded maximum asset size of %s bytes", maxBytes)
			}

			clientObjects = append(clientObjects, ObjectDescription{
				SourcePath:  fmt.Sprintf("%s/%s", bucketName, obj.Key),
				RelativeKey: strings.TrimPrefix(*obj.Key, clientPrefix),
			})
		}
	}

	return clientObjects, nil
}

func analyzePrerenderObjects(transportCtx context.Context, s3Client *s3.Client, bucketName, bucketPrefix, execIdentifier string, maxBytes int64) ([]ObjectDescription, error) {
	prerenderPrefix := fmt.Sprintf("%s/%s", execIdentifier, PRERENDER_PATH)
	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(fmt.Sprintf("%s%s", bucketPrefix, prerenderPrefix)),
	})

	var prerenderSize int64 = 0
	prerenderObjects := []ObjectDescription{}

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(transportCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %v", err)
		}

		for _, obj := range page.Contents {
			prerenderSize += *obj.Size
			if prerenderSize > maxBytes {
				return nil, fmt.Errorf("exceeded maximum asset size of %s bytes", maxBytes)
			}

			if !strings.HasSuffix(*obj.Key, ".html") {
				return nil, fmt.Errorf("prerendered objects ('%s') are expected to have the '.html' extension", *obj.Key)
			}

			prerenderObjects = append(prerenderObjects, ObjectDescription{
				SourcePath:  fmt.Sprintf("%s/%s", bucketName, obj.Key),
				RelativeKey: strings.TrimPrefix(*obj.Key, prerenderPrefix),
			})
		}
	}

	return prerenderObjects, nil
}