package deleteproject

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	cloudfrontkeyvaluetypes "github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/delete/eventcontext"
)

func deleteStaticAssets(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	if projectDoc.SharedInfrastructure.StaticBucketPath == "" {
		return nil
	}
	bucketPathSegments := strings.SplitN(projectDoc.SharedInfrastructure.StaticBucketPath, "/", 2)
	if len(bucketPathSegments) != 2 {
		return fmt.Errorf("failed to decode static bucket path")
	}
	bucketName := bucketPathSegments[0]
	// Check ensuring that, for whatever reason, bucketPrefix is NEVER "", which could lead to dangerous behavior.
	if bucketPathSegments[1] == "" {
		return fmt.Errorf("malformed bucket prefix detected")
	}
	bucketPrefix := fmt.Sprintf("%s/", bucketPathSegments[1])

	paginator := s3.NewListObjectsV2Paginator(eventCtx.S3Client, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		Prefix:  aws.String(bucketPrefix),
		MaxKeys: aws.Int32(1000), // DeleteObjects call deletes max 1000 objects, one page should only use one DeleteObjects call.
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(transportCtx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %v", err)
		}

		deleteObjects := []s3types.ObjectIdentifier{}
		for _, object := range page.Contents {
			deleteObjects = append(deleteObjects, s3types.ObjectIdentifier{
				Key: object.Key,
			})
		}

		_, err = eventCtx.S3Client.DeleteObjects(transportCtx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucketName),
			Delete: &s3types.Delete{
				Objects: deleteObjects,
				Quiet:   aws.Bool(true),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to delete objects in static bucket: %v", err)
		}
	}

	return nil
}

func deleteStaticPageKeys(transportCtx context.Context, eventCtx eventcontext.Context, staticPages map[string]string) error {
	if len(staticPages) < 1 {
		return nil
	}
	deleteStaticPageKeys := []cloudfrontkeyvaluetypes.DeleteKeyRequestListItem{}
	for key := range staticPages {
		deleteStaticPageKeys = append(deleteStaticPageKeys, cloudfrontkeyvaluetypes.DeleteKeyRequestListItem{
			Key: aws.String(key),
		})
	}

	storeMetadata, err := eventCtx.CloudfrontCacheClient.DescribeKeyValueStore(transportCtx, &cloudfrontkeyvaluestore.DescribeKeyValueStoreInput{
		KvsARN: aws.String(eventCtx.CloudfrontConfiguration.CacheArn),
	})
	if err != nil {
		return fmt.Errorf("failed to describe cdn store: %v", err)
	}
	_, err = eventCtx.CloudfrontCacheClient.UpdateKeys(transportCtx, &cloudfrontkeyvaluestore.UpdateKeysInput{
		KvsARN:  aws.String(eventCtx.CloudfrontConfiguration.CacheArn),
		Deletes: deleteStaticPageKeys,
		IfMatch: storeMetadata.ETag,
	})
	if err != nil {
		return fmt.Errorf("failed to update cdn store keys: %v", err)
	}

	return nil
}
