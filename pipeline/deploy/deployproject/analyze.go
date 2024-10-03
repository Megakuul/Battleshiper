package deployproject

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

const (
	SERVER_PATH    = "server/handler.zip"
	CLIENT_PATH    = "client/"
	PRERENDER_PATH = "prerendered/"
)

// ObjectDescription provides information about the location of an s3 object.
type ObjectDescription struct {
	// Key relative to the objects logical root
	// e.g. build_asset_bucket/x/123/client/_app/static/myimage.png = _app/static/myimage.png.
	RelativeKey string
	// Source bucket name of the object.
	SourceBucket string
	// Source bucket key of the object.
	SourceKey string
}

// BuildInformation provides information about the content of the build output.
type BuildInformation struct {
	ClientObjects      []ObjectDescription
	PrerenderedObjects []ObjectDescription
	ServerObject       ObjectDescription
	PageKeys           map[string]string
}

// analyzeBuildAssets analyzes the content of the build assets, expecting to find sveltekit build output from adapter-battleshiper.
func analyzeBuildAssets(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, subscriptionDoc *subscription.Subscription, execIdentifier string) (*BuildInformation, error) {
	bucketPathSegments := strings.SplitN(projectDoc.SharedInfrastructure.BuildAssetBucketPath, "/", 2)
	if len(bucketPathSegments) != 2 {
		return nil, fmt.Errorf("failed to decode build asset bucket path")
	}
	bucketName := bucketPathSegments[0]
	bucketPrefix := bucketPathSegments[1]

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
		PageKeys:           extractPageKeys(prerenderObjects, projectDoc.ProjectName),
	}, nil
}

func analyzeClientObjects(transportCtx context.Context, s3Client *s3.Client, bucketName, bucketPrefix, execIdentifier string, maxBytes int64) ([]ObjectDescription, error) {
	clientPrefix := fmt.Sprintf("%s/%s/%s", bucketPrefix, execIdentifier, CLIENT_PATH)
	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(clientPrefix),
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
				return nil, fmt.Errorf("exceeded maximum asset size of %d bytes", maxBytes)
			}

			clientObjects = append(clientObjects, ObjectDescription{
				SourceBucket: bucketName,
				SourceKey:    *obj.Key,
				RelativeKey:  strings.TrimPrefix(*obj.Key, clientPrefix),
			})
		}
	}

	return clientObjects, nil
}

func analyzePrerenderObjects(transportCtx context.Context, s3Client *s3.Client, bucketName, bucketPrefix, execIdentifier string, maxBytes int64) ([]ObjectDescription, error) {
	prerenderPrefix := fmt.Sprintf("%s/%s/%s", bucketPrefix, execIdentifier, PRERENDER_PATH)
	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prerenderPrefix),
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
				return nil, fmt.Errorf("exceeded maximum asset size of %d bytes", maxBytes)
			}

			if !strings.HasSuffix(*obj.Key, ".html") {
				return nil, fmt.Errorf("prerendered objects ('%s') are expected to have the '.html' extension", *obj.Key)
			}

			prerenderObjects = append(prerenderObjects, ObjectDescription{
				SourceBucket: bucketName,
				SourceKey:    *obj.Key,
				RelativeKey:  strings.TrimPrefix(*obj.Key, prerenderPrefix),
			})
		}
	}

	return prerenderObjects, nil
}

func analyzeServerObject(transportCtx context.Context, s3Client *s3.Client, bucketName, bucketPrefix, execIdentifier string, maxBytes int64) (*ObjectDescription, error) {
	serverKey := fmt.Sprintf("%s/%s/%s", bucketPrefix, execIdentifier, SERVER_PATH)

	serverObject, err := s3Client.HeadObject(transportCtx, &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(serverKey),
	})
	if err != nil {
		nfe := &s3types.NotFound{}
		if ok := errors.As(err, &nfe); ok {
			return nil, fmt.Errorf("expected server object at '%s'", serverKey)
		} else {
			return nil, fmt.Errorf("failed to fetch object: %v", err)
		}
	}

	if *serverObject.ContentLength > maxBytes {
		return nil, fmt.Errorf("exceeded maximum asset size of %d bytes", maxBytes)
	}

	return &ObjectDescription{
		SourceBucket: bucketName,
		SourceKey:    serverKey,
		RelativeKey:  SERVER_PATH,
	}, nil
}

func extractPageKeys(prerenderObjects []ObjectDescription, projectName string) map[string]string {
	pageKeys := map[string]string{}
	for _, object := range prerenderObjects {
		pageKey := fmt.Sprintf("/%s/%s", projectName, object.RelativeKey)
		// index key is the path that is received by cloudfront (essentially the path the user enters in the browser).
		// if the index is /index.html it is fully trimmed, otherwise just the .html extension is trimmed.
		if strings.HasSuffix(pageKey, "/index.html") {
			pageKeys[strings.TrimSuffix(pageKey, "/index.html")] = fmt.Sprintf("/%s", object.RelativeKey)
		} else {
			pageKeys[strings.TrimSuffix(pageKey, ".html")] = fmt.Sprintf("/%s", object.RelativeKey)
		}

	}
	return pageKeys
}
