package deployproject

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
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

func cleanStaticBucket(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
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

		if len(deleteObjects) < 1 {
			continue
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

func copyStaticAssets(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, assets []ObjectDescription) error {
	bucketPathSegments := strings.SplitN(projectDoc.SharedInfrastructure.StaticBucketPath, "/", 2)
	if len(bucketPathSegments) != 2 {
		return fmt.Errorf("failed to decode static bucket path")
	}
	bucketName := bucketPathSegments[0]
	bucketPrefix := fmt.Sprintf("%s/", bucketPathSegments[1])

	for _, obj := range assets {
		_, err := eventCtx.S3Client.CopyObject(transportCtx, &s3.CopyObjectInput{
			Bucket:     aws.String(bucketName),
			CopySource: aws.String(fmt.Sprintf("%s/%s", obj.SourceBucket, obj.SourceKey)),
			Key:        aws.String(fmt.Sprintf("%s%s", bucketPrefix, obj.RelativeKey)),
		})
		if err != nil {
			return fmt.Errorf("failed to move build assets to static bucket: %v", err)
		}
	}

	return nil
}

func copyStaticPages(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, pages []ObjectDescription) error {
	bucketPathSegments := strings.SplitN(projectDoc.SharedInfrastructure.StaticBucketPath, "/", 2)
	if len(bucketPathSegments) != 2 {
		return fmt.Errorf("failed to decode static bucket path")
	}
	bucketName := bucketPathSegments[0]
	bucketPrefix := fmt.Sprintf("%s/", bucketPathSegments[1])

	for _, obj := range pages {
		_, err := eventCtx.S3Client.CopyObject(transportCtx, &s3.CopyObjectInput{
			Bucket:     aws.String(bucketName),
			CopySource: aws.String(fmt.Sprintf("%s/%s", obj.SourceBucket, obj.SourceKey)),
			Key:        aws.String(fmt.Sprintf("%s%s", bucketPrefix, obj.RelativeKey)),
		})
		if err != nil {
			return fmt.Errorf("failed to move build pages to static bucket: %v", err)
		}
	}

	return nil
}

func updateStaticPageKeys(transportCtx context.Context, eventCtx eventcontext.Context, newStaticPages map[string]string, oldStaticPages map[string]string) error {
	addStaticPageKeys := []cloudfrontkeyvaluetypes.PutKeyRequestListItem{}
	for key, path := range newStaticPages {
		addStaticPageKeys = append(addStaticPageKeys, cloudfrontkeyvaluetypes.PutKeyRequestListItem{
			Key:   aws.String(key),
			Value: aws.String(path),
		})
	}

	deleteStaticPageKeys := []cloudfrontkeyvaluetypes.DeleteKeyRequestListItem{}
	for key := range oldStaticPages {
		if _, exists := newStaticPages[key]; !exists {
			deleteStaticPageKeys = append(deleteStaticPageKeys, cloudfrontkeyvaluetypes.DeleteKeyRequestListItem{
				Key: aws.String(key),
			})
		}
	}

	if len(addStaticPageKeys) < 1 && len(deleteStaticPageKeys) < 1 {
		return nil
	}

	storeMetadata, err := eventCtx.CloudfrontCacheClient.DescribeKeyValueStore(transportCtx, &cloudfrontkeyvaluestore.DescribeKeyValueStoreInput{
		KvsARN: aws.String(eventCtx.ProjectConfiguration.CloudfrontCacheArn),
	})
	if err != nil {
		return fmt.Errorf("failed to describe cdn store: %v", err)
	}
	_, err = eventCtx.CloudfrontCacheClient.UpdateKeys(transportCtx, &cloudfrontkeyvaluestore.UpdateKeysInput{
		KvsARN:  aws.String(eventCtx.ProjectConfiguration.CloudfrontCacheArn),
		Puts:    addStaticPageKeys,
		Deletes: deleteStaticPageKeys,
		IfMatch: storeMetadata.ETag,
	})
	if err != nil {
		return fmt.Errorf("failed to update cdn store keys: %v", err)
	}

	return nil
}
