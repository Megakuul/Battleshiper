package cleanupweb

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	codedeploytypes "github.com/aws/aws-sdk-go-v2/service/codedeploy/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/megakuul/battleshiper/web/hooks/cleanup/eventcontext"
)

const (
	DEPLOYMENT_ID_TAG_KEY = "DeploymentID"
)

type CodeDeployEvent struct {
	DeploymentId                  string `json:"DeploymentId"`
	LifecycleEventHookExecutionId string `json:"LifecycleEventHookExecutionId"`
}

func HandleCleanupWeb(eventCtx eventcontext.Context) func(context.Context, CodeDeployEvent) error {
	return func(ctx context.Context, event CodeDeployEvent) error {
		err := runHandleCleanupWeb(event, ctx, eventCtx)
		if err != nil {
			if _, err := eventCtx.CodeDeployClient.PutLifecycleEventHookExecutionStatus(ctx, &codedeploy.PutLifecycleEventHookExecutionStatusInput{
				DeploymentId:                  aws.String(event.DeploymentId),
				LifecycleEventHookExecutionId: aws.String(event.LifecycleEventHookExecutionId),
				Status:                        codedeploytypes.LifecycleEventStatusFailed,
			}); err != nil {
				return fmt.Errorf("ERROR CLEANUPWEB: %v", err)
			}
			log.Printf("ERROR CLEANUPWEB: %v\n", err)
			return nil
		}
		if _, err := eventCtx.CodeDeployClient.PutLifecycleEventHookExecutionStatus(ctx, &codedeploy.PutLifecycleEventHookExecutionStatusInput{
			DeploymentId:                  aws.String(event.DeploymentId),
			LifecycleEventHookExecutionId: aws.String(event.LifecycleEventHookExecutionId),
			Status:                        codedeploytypes.LifecycleEventStatusSucceeded,
		}); err != nil {
			return fmt.Errorf("ERROR CLEANUPWEB: %v", err)
		}
		return nil
	}
}

func runHandleCleanupWeb(event CodeDeployEvent, transportCtx context.Context, eventCtx eventcontext.Context) error {
	paginator := s3.NewListObjectsV2Paginator(eventCtx.S3Client, &s3.ListObjectsV2Input{
		Bucket:  aws.String(eventCtx.BucketConfiguration.StaticBucketName),
		MaxKeys: aws.Int32(1000), // DeleteObjects call deletes max 1000 objects, one page should only use one DeleteObjects call.
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(transportCtx)
		if err != nil {
			return err
		}

		deleteObjects := []s3types.ObjectIdentifier{}

		for _, object := range page.Contents {
			objectTagging, err := eventCtx.S3Client.GetObjectTagging(transportCtx, &s3.GetObjectTaggingInput{
				Bucket: aws.String(eventCtx.BucketConfiguration.StaticBucketName),
				Key:    object.Key,
			})
			if err != nil {
				return err
			}
			if !checkObjectTag(objectTagging.TagSet, DEPLOYMENT_ID_TAG_KEY, event.DeploymentId) {
				deleteObjects = append(deleteObjects, s3types.ObjectIdentifier{
					Key: object.Key,
				})
			}
		}

		_, err = eventCtx.S3Client.DeleteObjects(transportCtx, &s3.DeleteObjectsInput{
			Bucket: aws.String(eventCtx.BucketConfiguration.StaticBucketName),
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

// checkObjectTag checks if the expected key / value combination exists (necessary because the retarded aws interface doen't return a map).
func checkObjectTag(tags []s3types.Tag, expectedKey string, expectedValue string) bool {
	for _, tag := range tags {
		if *tag.Key == expectedKey && *tag.Value == expectedValue {
			return true
		}
	}
	return false
}
