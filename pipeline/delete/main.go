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
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/delete/deleteprojects"
	"github.com/megakuul/battleshiper/pipeline/delete/eventcontext"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	REGION               = os.Getenv("AWS_REGION")
	DATABASE_ENDPOINT    = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME        = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN  = os.Getenv("DATABASE_SECRET_ARN")
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

	databaseOptions, err := database.CreateDatabaseOptions(awsConfig, context.TODO(), DATABASE_SECRET_ARN, DATABASE_ENDPOINT, DATABASE_NAME)
	if err != nil {
		return err
	}
	databaseClient, err := mongo.Connect(context.TODO(), databaseOptions)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		if err = databaseClient.Disconnect(ctx); err != nil {
			log.Printf("ERROR CLEANUP: %v\n", err)
		}
		cancel()
	}()
	databaseHandle := databaseClient.Database(DATABASE_NAME)

	database.SetupIndexes(databaseHandle.Collection(project.PROJECT_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"deleted"}, SortingOrder: 1, Unique: false},
	})

	deletionTimeout, err := time.ParseDuration(DELETION_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse DELETION_TIMEOUT environment variable")
	}

	lambda.Start(deleteprojects.HandleDeleteProjects(eventcontext.Context{
		Database:              databaseHandle,
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
