package bootstrapweb

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	codedeploytypes "github.com/aws/aws-sdk-go-v2/service/codedeploy/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/web/hooks/bootstrap/eventcontext"
)

const (
	DEPLOYMENT_ID_TAG_KEY     = "DeploymentID"
	PRERENDERED_RELATIVE_PATH = "prerendered"
	CLIENT_RELATIVE_PATH      = "client"
)

type CodeDeployEvent struct {
	DeploymentId                  string `json:"DeploymentId"`
	LifecycleEventHookExecutionId string `json:"LifecycleEventHookExecutionId"`
}

func HandleBootstrapWeb(eventCtx eventcontext.Context) func(context.Context, CodeDeployEvent) error {
	return func(ctx context.Context, event CodeDeployEvent) error {
		err := runHandleBootstrapWeb(event, ctx, eventCtx)
		if err != nil {
			_, err := eventCtx.CodeDeployClient.PutLifecycleEventHookExecutionStatus(ctx, &codedeploy.PutLifecycleEventHookExecutionStatusInput{
				DeploymentId:                  aws.String(event.DeploymentId),
				LifecycleEventHookExecutionId: aws.String(event.LifecycleEventHookExecutionId),
				Status:                        codedeploytypes.LifecycleEventStatusFailed,
			})
			if err != nil {
				return fmt.Errorf("ERROR BOOTSTRAPWEB: %v", err)
			}
			return nil
		}
		_, err = eventCtx.CodeDeployClient.PutLifecycleEventHookExecutionStatus(ctx, &codedeploy.PutLifecycleEventHookExecutionStatusInput{
			DeploymentId:                  aws.String(event.DeploymentId),
			LifecycleEventHookExecutionId: aws.String(event.LifecycleEventHookExecutionId),
			Status:                        codedeploytypes.LifecycleEventStatusSucceeded,
		})
		if err != nil {
			return fmt.Errorf("ERROR BOOTSTRAPWEB: %v", err)
		}
		return nil
	}
}

func runHandleBootstrapWeb(event CodeDeployEvent, transportCtx context.Context, eventCtx eventcontext.Context) error {
	prerenderPath, err := filepath.Abs(PRERENDERED_RELATIVE_PATH)
	if err != nil {
		return err
	}
	if err := uploadDirectory(transportCtx, eventCtx, prerenderPath, event.DeploymentId); err != nil {
		return fmt.Errorf("failed to upload prerendered assets: %v", err)
	}

	clientPath, err := filepath.Abs(CLIENT_RELATIVE_PATH)
	if err != nil {
		return err
	}
	if err := uploadDirectory(transportCtx, eventCtx, clientPath, event.DeploymentId); err != nil {
		return fmt.Errorf("failed to upload client assets: %v", err)
	}

	return nil
}

func uploadDirectory(transportCtx context.Context, eventCtx eventcontext.Context, rootPath string, deploymentId string) error {
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			s3Key := strings.TrimPrefix(path, rootPath)
			_, err = eventCtx.S3Client.PutObject(transportCtx, &s3.PutObjectInput{
				Bucket:  aws.String(eventCtx.BucketConfiguration.StaticBucketName),
				Key:     aws.String(s3Key),
				Body:    file,
				Tagging: aws.String(fmt.Sprintf("%s=%s", DEPLOYMENT_ID_TAG_KEY, deploymentId)),
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
