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
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/api/pipeline/event"
	"github.com/megakuul/battleshiper/api/pipeline/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
	"github.com/megakuul/battleshiper/lib/router"
)

var (
	REGION                       = os.Getenv("AWS_REGION")
	GITHUB_CLIENT_CREDENTIAL_ARN = os.Getenv("GITHUB_CLIENT_CREDENTIAL_ARN")
	DATABASE_ENDPOINT            = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME                = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN          = os.Getenv("DATABASE_SECRET_ARN")
	TICKET_CREDENTIAL_ARN        = os.Getenv("TICKET_CREDENTIAL_ARN")
	BUILD_EVENTBUS_NAME          = os.Getenv("BUILD_EVENTBUS_NAME")
	BUILD_EVENT_SOURCE           = os.Getenv("BUILD_EVENT_SOURCE")
	BUILD_EVENT_ACTION           = os.Getenv("BUILD_EVENT_ACTION")
	DEPLOY_EVENT_SOURCE          = os.Getenv("DEPLOY_EVENT_SOURCE")
	DEPLOY_EVENT_ACTION          = os.Getenv("DEPLOY_EVENT_ACTION")
	DEPLOY_EVENT_TICKET_TTL      = os.Getenv("DEPLOY_EVENT_TICKET_TTL")
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

	cloudwatchClient := cloudwatchlogs.NewFromConfig(awsConfig)

	eventbridgeClient := eventbridge.NewFromConfig(awsConfig)

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

	database.SetupIndexes(databaseHandle.Collection(project.PROJECT_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"repository.id", "owner_id", "repository.branch", "deleted"}, SortingOrder: 1, Unique: false},
	})

	database.SetupIndexes(databaseHandle.Collection(subscription.SUBSCRIPTION_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"id"}, SortingOrder: 1, Unique: true},
	})

	webhookClient, err := auth.CreateGithubWebhookClient(awsConfig, context.TODO(), GITHUB_CLIENT_CREDENTIAL_ARN)
	if err != nil {
		return err
	}

	buildEventOptions := pipeline.CreateEventOptions(BUILD_EVENTBUS_NAME, BUILD_EVENT_SOURCE, BUILD_EVENT_ACTION, nil)

	deployTicketTTL, err := strconv.Atoi(DEPLOY_EVENT_TICKET_TTL)
	if err != nil {
		return fmt.Errorf("failed to parse DEPLOY_EVENT_TICKET_TTL environment variable")
	}
	deployTicketOptions, err := pipeline.CreateTicketOptions(
		awsConfig, context.TODO(), TICKET_CREDENTIAL_ARN, DEPLOY_EVENT_SOURCE, DEPLOY_EVENT_ACTION, time.Duration(deployTicketTTL)*time.Second)
	if err != nil {
		return err
	}

	httpRouter := router.NewRouter(routecontext.Context{
		WebhookClient:       webhookClient,
		CloudwatchClient:    cloudwatchClient,
		Database:            databaseHandle,
		EventClient:         eventbridgeClient,
		BuildEventOptions:   buildEventOptions,
		DeployTicketOptions: deployTicketOptions,
	})

	httpRouter.AddRoute("POST", "/api/pipeline/event", event.HandleEvent)

	lambda.Start(httpRouter.Route)

	return nil
}
