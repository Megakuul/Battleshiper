package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/web/hooks/cleanup/cleanupweb"
	"github.com/megakuul/battleshiper/web/hooks/cleanup/eventcontext"
)

var (
	REGION             = os.Getenv("AWS_REGION")
	BOOTSTRAP_TIMEOUT  = os.Getenv("BOOTSTRAP_TIMEOUT")
	STATIC_BUCKET_NAME = os.Getenv("STATIC_BUCKET_NAME")
)

func main() {
	if err := run(); err != nil {
		log.Printf("ERROR INITIALIZATION: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	bootstrapTimeout, err := time.ParseDuration(BOOTSTRAP_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse BOOTSTRAP_TIMEOUT environment variable")
	}
	bootstrapContext, cancel := context.WithTimeout(context.Background(), bootstrapTimeout)
	defer cancel()

	awsConfig, err := config.LoadDefaultConfig(bootstrapContext, config.WithRegion(REGION))
	if err != nil {
		return fmt.Errorf("failed to load aws config: %v", err)
	}

	codeDeployClient := codedeploy.NewFromConfig(awsConfig)

	s3Client := s3.NewFromConfig(awsConfig)

	lambda.Start(cleanupweb.HandleCleanupWeb(eventcontext.Context{
		CodeDeployClient: codeDeployClient,
		S3Client:         s3Client,
		BucketConfiguration: &eventcontext.BucketConfiguration{
			StaticBucketName: STATIC_BUCKET_NAME,
		},
	}))

	return nil
}
