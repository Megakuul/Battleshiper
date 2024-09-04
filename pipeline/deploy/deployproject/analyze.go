package deployproject

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

const (
	SERVER_PATH    = "server/index.js"
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
	ClientObjects      []ObjectDescription
	PrerenderedObjects []ObjectDescription
	ServerObject       ObjectDescription
}

// analyzeBuildAssets analyzes the content of the build assets, expecting to find sveltekit build output from adapter-battleshiper.
func analyzeBuildAssets(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, subscriptionDoc *subscription.Subscription, execIdentifier string) (*BuildInformation, error) {
	bucketPathSegments := strings.SplitN(projectDoc.SharedInfrastructure.BuildAssetBucketPath, "/", 2)
	if len(bucketPathSegments) != 2 {
		return nil, fmt.Errorf("failed to decode build asset bucket path")
	}
	bucketName := bucketPathSegments[0]
	bucketPrefix := fmt.Sprintf("%s/", bucketPathSegments[1])

	clientObjects, err := analyzeClientObjects(
		transportCtx, eventCtx.S3Client, bucketName, bucketPrefix, execIdentifier, subscriptionDoc.ProjectSpecs.ClientStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze client assets: %v", err)
	}

	prerenderObjects, err := analyzePrerenderObjects(
		transportCtx, eventCtx.S3Client, bucketName, bucketPrefix, execIdentifier, subscriptionDoc.ProjectSpecs.PrerenderStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze prerendered assets: %v", err)
	}

	serverObject, err := analyzeServerObject(
		transportCtx, eventCtx.S3Client, bucketName, bucketPrefix, execIdentifier, subscriptionDoc.ProjectSpecs.ServerStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze server asset: %v", err)
	}

	return &BuildInformation{
		ClientObjects:      clientObjects,
		PrerenderedObjects: prerenderObjects,
		ServerObject:       *serverObject,
	}, nil
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
				ObjectName:  path.Base(*obj.Key),
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
				ObjectName:  path.Base(*obj.Key),
				SourcePath:  fmt.Sprintf("%s/%s", bucketName, obj.Key),
				RelativeKey: strings.TrimPrefix(*obj.Key, prerenderPrefix),
			})
		}
	}

	return prerenderObjects, nil
}

func analyzeServerObject(transportCtx context.Context, s3Client *s3.Client, bucketName, bucketPrefix, execIdentifier string, maxBytes int64) (*ObjectDescription, error) {
	serverKey := fmt.Sprintf("%s%s/%s", bucketPrefix, execIdentifier, SERVER_PATH)

	serverObject, err := s3Client.HeadObject(transportCtx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(serverKey),
	})
	if err != nil {
		var nfe s3types.NotFound
		if ok := errors.As(err, nfe); ok {
			return nil, fmt.Errorf("expected server object at '%s'", serverKey)
		} else {
			return nil, fmt.Errorf("failed to fetch object: %v", err)
		}
	}

	if *serverObject.ContentLength > maxBytes {
		return nil, fmt.Errorf("exceeded maximum asset size of %s bytes", maxBytes)
	}

	return &ObjectDescription{
		ObjectName:  path.Base(serverKey),
		SourcePath:  fmt.Sprintf("%s/%s", bucketName, serverKey),
		RelativeKey: SERVER_PATH,
	}, nil
}
