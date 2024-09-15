package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
	"github.com/megakuul/battleshiper/pipeline/deploy/deployproject"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	REGION                = os.Getenv("AWS_REGION")
	DATABASE_ENDPOINT     = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME         = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN   = os.Getenv("DATABASE_SECRET_ARN")
	TICKET_CREDENTIAL_ARN = os.Getenv("TICKET_CREDENTIAL_ARN")
	DEPLOYMENT_TIMEOUT    = os.Getenv("DEPLOYMENT_TIMEOUT")
	CLOUDFRONT_CACHE_ARN  = os.Getenv("CLOUDFRONT_CACHE_ARN")
	SERVER_NAME_PREFIX    = os.Getenv("SERVER_NAME_PREFIX")
	SERVER_RUNTIME        = os.Getenv("SERVER_RUNTIME")
	SERVER_MEMORY         = os.Getenv("SERVER_MEMORY")
	SERVER_TIMEOUT        = os.Getenv("SERVER_TIMEOUT")
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

	cloudformationClient := cloudformation.NewFromConfig(awsConfig)

	cloudwatchClient := cloudwatchlogs.NewFromConfig(awsConfig)

	s3Client := s3.NewFromConfig(awsConfig)

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
		{FieldNames: []string{"name"}, SortingOrder: 1, Unique: true},
		{FieldNames: []string{"owner_id"}, SortingOrder: 1, Unique: false},
		{FieldNames: []string{"deleted"}, SortingOrder: 1, Unique: false},
	})

	database.SetupIndexes(databaseHandle.Collection(user.USER_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"id"}, SortingOrder: 1, Unique: true},
	})

	database.SetupIndexes(databaseHandle.Collection(subscription.SUBSCRIPTION_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"id"}, SortingOrder: 1, Unique: true},
	})

	ticketOptions, err := pipeline.CreateTicketOptions(awsConfig, context.TODO(), TICKET_CREDENTIAL_ARN, "", "", 0)
	if err != nil {
		return err
	}

	deploymentTimeout, err := time.ParseDuration(DEPLOYMENT_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse DEPLOYMENT_TIMEOUT environment variable")
	}

	serverMemory, err := strconv.Atoi(SERVER_MEMORY)
	if err != nil {
		return fmt.Errorf("failed to parse SERVER_MEMORY environment variable")
	}

	serverTimeout, err := strconv.Atoi(SERVER_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse SERVER_TIMEOUT environment variable")
	}

	lambda.Start(deployproject.HandleDeployProject(eventcontext.Context{
		Database:              databaseHandle,
		TicketOptions:         ticketOptions,
		CloudformationClient:  cloudformationClient,
		S3Client:              s3Client,
		CloudwatchClient:      cloudwatchClient,
		CloudfrontCacheClient: cloudfrontClient,
		DeploymentConfiguration: &eventcontext.DeploymentConfiguration{
			Timeout: deploymentTimeout,
		},
		ProjectConfiguration: &eventcontext.ProjectConfiguration{
			ServerNamePrefix:   SERVER_NAME_PREFIX,
			ServerRuntime:      SERVER_RUNTIME,
			ServerMemory:       serverMemory,
			ServerTimeout:      serverTimeout,
			CloudfrontCacheArn: CLOUDFRONT_CACHE_ARN,
		},
	}))

	return nil
}
