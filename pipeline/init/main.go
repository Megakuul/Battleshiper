package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
	"github.com/megakuul/battleshiper/pipeline/deploy/initproject"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	REGION              = os.Getenv("AWS_REGION")
	JWT_CREDENTIAL_ARN  = os.Getenv("JWT_CREDENTIAL_ARN")
	DATABASE_ENDPOINT   = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME       = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN = os.Getenv("DATABASE_SECRET_ARN")
	EVENTBUS_NAME       = os.Getenv("EVENTBUS_NAME")
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

	lambda.Start(initproject.HandleInitProject(eventcontext.Context{
		Database: databaseHandle,
	}))

	return nil
}
