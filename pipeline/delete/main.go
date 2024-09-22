package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/pipeline/delete/deleteprojects"
	"github.com/megakuul/battleshiper/pipeline/delete/eventcontext"
)

var (
	REGION               = os.Getenv("AWS_REGION")
	PROJECTTABLE         = os.Getenv("PROJECTTABLE")
	DELETION_TIMEOUT     = os.Getenv("DELETION_TIMEOUT")
	STATIC_BUCKET_NAME   = os.Getenv("STATIC_BUCKET_NAME")
	CLOUDFRONT_CACHE_ARN = os.Getenv("CLOUDFRONT_CACHE_ARN")
)

func main() {
	if err := run(); err != nil {
		log.Printf("ERROR INITIALIZATION: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	awsConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(REGION))
	if err != nil {
		return fmt.Errorf("failed to load aws config: %v", err)
	}

	s3Client := s3.NewFromConfig(awsConfig)

	cloudformationClient := cloudformation.NewFromConfig(awsConfig)

	cloudfrontClient := cloudfrontkeyvaluestore.NewFromConfig(awsConfig)

	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	deletionTimeout, err := time.ParseDuration(DELETION_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse DELETION_TIMEOUT environment variable")
	}

	lambda.Start(deleteprojects.HandleDeleteProjects(eventcontext.Context{
		DynamoClient:          dynamoClient,
		ProjectTable:          PROJECTTABLE,
		S3Client:              s3Client,
		CloudformationClient:  cloudformationClient,
		CloudfrontCacheClient: cloudfrontClient,
		DeletionConfiguration: &eventcontext.DeletionConfiguration{
			Timeout: deletionTimeout,
		},
		BucketConfiguration: &eventcontext.BucketConfiguration{
			StaticBucketName: STATIC_BUCKET_NAME,
		},
		CloudfrontConfiguration: &eventcontext.CloudfrontConfiguration{
			CacheArn: CLOUDFRONT_CACHE_ARN,
		},
	}))

	return nil
}
