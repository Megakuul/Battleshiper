package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/api/resource/createproject"
	"github.com/megakuul/battleshiper/api/resource/deleteproject"
	"github.com/megakuul/battleshiper/api/resource/fetchlog"
	"github.com/megakuul/battleshiper/api/resource/listproject"
	"github.com/megakuul/battleshiper/api/resource/listrepository"
	"github.com/megakuul/battleshiper/api/resource/routecontext"
	"github.com/megakuul/battleshiper/api/resource/updateproject"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
	"github.com/megakuul/battleshiper/lib/router"
)

var (
	REGION                = os.Getenv("AWS_REGION")
	JWT_CREDENTIAL_ARN    = os.Getenv("JWT_CREDENTIAL_ARN")
	DATABASE_ENDPOINT     = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME         = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN   = os.Getenv("DATABASE_SECRET_ARN")
	EVENTBUS_NAME         = os.Getenv("EVENTBUS_NAME")
	TICKET_CREDENTIAL_ARN = os.Getenv("TICKET_CREDENTIAL_ARN")
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

	eventClient := eventbridge.NewFromConfig(awsConfig)

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
	})

	database.SetupIndexes(databaseHandle.Collection(subscription.SUBSCRIPTION_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"id"}, SortingOrder: 1, Unique: true},
	})

	database.SetupIndexes(databaseHandle.Collection(project.PROJECT_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"name"}, SortingOrder: 1, Unique: true},
		{FieldNames: []string{"owner_id"}, SortingOrder: 1, Unique: false},
	})

	jwtOptions, err := auth.CreateJwtOptions(awsConfig, context.TODO(), JWT_CREDENTIAL_ARN, 0)
	if err != nil {
		return err
	}

	ticketOptions, err := pipeline.CreateTicketOptions(awsConfig, context.TODO(), TICKET_CREDENTIAL_ARN)
	if err != nil {
		return err
	}

	httpRouter := router.NewRouter(routecontext.Context{
		CloudWatchClient: cloudwatchClient,
		JwtOptions:       jwtOptions,
		Database:         databaseHandle,
		TicketOptions:    ticketOptions,
		EventClient:      eventClient,
		EventBus:         EVENTBUS_NAME,
	})

	httpRouter.AddRoute("GET", "/api/resource/listrepository", listrepository.HandleListRepositories)
	httpRouter.AddRoute("GET", "/api/resource/listproject", listproject.HandleListProject)
	httpRouter.AddRoute("GET", "/api/resource/fetchlog", fetchlog.HandleFetchLog)
	httpRouter.AddRoute("POST", "/api/resource/createproject", createproject.HandleCreateProject)
	httpRouter.AddRoute("PATCH", "/api/resource/updateproject", updateproject.HandleUpdateProject)
	httpRouter.AddRoute("DELETE", "/api/resource/deleteproject", deleteproject.HandleDeleteProject)

	lambda.Start(httpRouter.Route)

	return nil
}
