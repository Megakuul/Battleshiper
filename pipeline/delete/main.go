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
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/pipeline/delete/deleteproject"
	"github.com/megakuul/battleshiper/pipeline/delete/eventcontext"
)

var (
	REGION                = os.Getenv("AWS_REGION")
	BOOTSTRAP_TIMEOUT     = os.Getenv("BOOTSTRAP_TIMEOUT")
	PROJECTTABLE          = os.Getenv("PROJECTTABLE")
	TICKET_CREDENTIAL_ARN = os.Getenv("TICKET_CREDENTIAL_ARN")
	DELETION_TIMEOUT      = os.Getenv("DELETION_TIMEOUT")
	STATIC_BUCKET_NAME    = os.Getenv("STATIC_BUCKET_NAME")
	CLOUDFRONT_CACHE_ARN  = os.Getenv("CLOUDFRONT_CACHE_ARN")
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

	s3Client := s3.NewFromConfig(awsConfig)

	cloudformationClient := cloudformation.NewFromConfig(awsConfig)

	cloudfrontClient := cloudfrontkeyvaluestore.NewFromConfig(awsConfig)

	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	ticketOptions, err := pipeline.CreateTicketOptions(awsConfig, bootstrapContext, TICKET_CREDENTIAL_ARN, "", "", 0)
	if err != nil {
		return err
	}

	deletionTimeout, err := time.ParseDuration(DELETION_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse DELETION_TIMEOUT environment variable")
	}

	lambda.Start(deleteproject.HandleDeleteProject(eventcontext.Context{
		DynamoClient:          dynamoClient,
		ProjectTable:          PROJECTTABLE,
		TicketOptions:         ticketOptions,
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
