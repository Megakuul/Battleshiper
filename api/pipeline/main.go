package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/api/pipeline/event"
	"github.com/megakuul/battleshiper/api/pipeline/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/user"
	"github.com/megakuul/battleshiper/lib/router"
)

var (
	REGION                       = os.Getenv("AWS_REGION")
	GITHUB_CLIENT_CREDENTIAL_ARN = os.Getenv("GITHUB_CLIENT_CREDENTIAL_ARN")
	DATABASE_ENDPOINT            = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME                = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN          = os.Getenv("DATABASE_SECRET_ARN")
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

	database.SetupIndexes(databaseHandle.Collection(user.USER_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"id"}, SortingOrder: 1, Unique: true},
		{FieldNames: []string{"github_data.installation_id"}, SortingOrder: 1, Unique: true},
	})

	webhookClient, err := auth.CreateGithubWebhookClient(awsConfig, context.TODO(), GITHUB_CLIENT_CREDENTIAL_ARN)
	if err != nil {
		return err
	}

	httpRouter := router.NewRouter(routecontext.Context{
		WebhookClient: webhookClient,
		Database:      databaseHandle,
	})

	httpRouter.AddRoute("POST", "/api/pipeline/event", event.HandleEvent)

	lambda.Start(httpRouter.Route)

	return nil
}
