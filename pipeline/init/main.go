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
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
	"github.com/megakuul/battleshiper/pipeline/deploy/initproject"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	REGION                 = os.Getenv("AWS_REGION")
	DATABASE_ENDPOINT      = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME          = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN    = os.Getenv("DATABASE_SECRET_ARN")
	EVENTBUS_NAME          = os.Getenv("EVENTBUS_NAME")
	TICKET_CREDENTIAL_ARN  = os.Getenv("TICKET_CREDENTIAL_ARN")
	CLOUDFORMATION_TIMEOUT = os.Getenv("CLOUDFORMATION_TIMEOUT")
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
	})

	cloudformationTimeout, err := time.ParseDuration(CLOUDFORMATION_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse CLOUDFORMATION_TIMEOUT environment variable")
	}

	ticketOptions, err := pipeline.CreateTicketOptions(awsConfig, context.TODO(), TICKET_CREDENTIAL_ARN, "", 0)
	if err != nil {
		return err
	}

	lambda.Start(initproject.HandleInitProject(eventcontext.Context{
		Database:              databaseHandle,
		TicketOptions:         ticketOptions,
		CloudformationClient:  cloudformationClient,
		CloudformationTimeout: cloudformationTimeout,
	}))

	return nil
}
