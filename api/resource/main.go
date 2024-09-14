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
	"github.com/aws/aws-sdk-go-v2/service/cloudfrontkeyvaluestore"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/api/resource/buildproject"
	"github.com/megakuul/battleshiper/api/resource/createproject"
	"github.com/megakuul/battleshiper/api/resource/deleteproject"
	"github.com/megakuul/battleshiper/api/resource/fetchlog"
	"github.com/megakuul/battleshiper/api/resource/listproject"
	"github.com/megakuul/battleshiper/api/resource/listrepository"
	"github.com/megakuul/battleshiper/api/resource/routecontext"
	"github.com/megakuul/battleshiper/api/resource/updatealias"
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
	REGION                  = os.Getenv("AWS_REGION")
	JWT_CREDENTIAL_ARN      = os.Getenv("JWT_CREDENTIAL_ARN")
	DATABASE_ENDPOINT       = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME           = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN     = os.Getenv("DATABASE_SECRET_ARN")
	TICKET_CREDENTIAL_ARN   = os.Getenv("TICKET_CREDENTIAL_ARN")
	INIT_EVENTBUS_NAME      = os.Getenv("INIT_EVENTBUS_NAME")
	INIT_EVENT_SOURCE       = os.Getenv("INIT_EVENT_SOURCE")
	INIT_EVENT_ACTION       = os.Getenv("INIT_TICKET_ACTION")
	INIT_EVENT_TICKET_TTL   = os.Getenv("INIT_TICKET_TTL")
	BUILD_EVENTBUS_NAME     = os.Getenv("BUILD_EVENTBUS_NAME")
	BUILD_EVENT_SOURCE      = os.Getenv("BUILD_EVENT_SOURCE")
	BUILD_EVENT_ACTION      = os.Getenv("BUILD_EVENT_ACTION")
	DEPLOY_EVENT_SOURCE     = os.Getenv("DEPLOY_EVENT_SOURCE")
	DEPLOY_EVENT_ACTION     = os.Getenv("DEPLOY_EVENT_ACTION")
	DEPLOY_EVENT_TICKET_TTL = os.Getenv("DEPLOY_EVENT_TICKET_TTL")
	CLOUDFRONT_CACHE_ARN    = os.Getenv("CLOUDFRONT_CACHE_ARN")
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

	database.SetupIndexes(databaseHandle.Collection(user.USER_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"id"}, SortingOrder: 1, Unique: true},
	})

	database.SetupIndexes(databaseHandle.Collection(subscription.SUBSCRIPTION_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"id"}, SortingOrder: 1, Unique: true},
	})

	database.SetupIndexes(databaseHandle.Collection(project.PROJECT_COLLECTION), context.TODO(), []database.Index{
		{FieldNames: []string{"name"}, SortingOrder: 1, Unique: true},
		{FieldNames: []string{"owner_id"}, SortingOrder: 1, Unique: false},
		{FieldNames: []string{"deleted"}, SortingOrder: 1, Unique: false},
	})

	jwtOptions, err := auth.CreateJwtOptions(awsConfig, context.TODO(), JWT_CREDENTIAL_ARN, 0)
	if err != nil {
		return err
	}

	initTicketTTL, err := strconv.Atoi(INIT_EVENT_TICKET_TTL)
	if err != nil {
		return fmt.Errorf("failed to parse INIT_EVENT_TICKET_TTL environment variable")
	}
	initTicketOptions, err := pipeline.CreateTicketOptions(
		awsConfig, context.TODO(), TICKET_CREDENTIAL_ARN, INIT_EVENT_SOURCE, INIT_EVENT_ACTION, time.Duration(initTicketTTL)*time.Second)
	if err != nil {
		return err
	}
	initEventOptions := pipeline.CreateEventOptions(INIT_EVENTBUS_NAME, INIT_EVENT_SOURCE, INIT_EVENT_ACTION, initTicketOptions)

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
		CloudwatchClient:      cloudwatchClient,
		JwtOptions:            jwtOptions,
		Database:              databaseHandle,
		EventClient:           eventClient,
		InitEventOptions:      initEventOptions,
		BuildEventOptions:     buildEventOptions,
		DeployTicketOptions:   deployTicketOptions,
		CloudfrontCacheClient: cloudfrontClient,
		CloudfrontCacheArn:    CLOUDFRONT_CACHE_ARN,
	})

	httpRouter.AddRoute("GET", "/api/resource/listrepository", listrepository.HandleListRepositories)
	httpRouter.AddRoute("GET", "/api/resource/listproject", listproject.HandleListProject)
	httpRouter.AddRoute("GET", "/api/resource/fetchlog", fetchlog.HandleFetchLog)
	httpRouter.AddRoute("POST", "/api/resource/createproject", createproject.HandleCreateProject)
	httpRouter.AddRoute("POST", "/api/resource/buildproject", buildproject.HandleBuildProject)
	httpRouter.AddRoute("POST", "/api/resource/updatealias", updatealias.HandleUpdateAlias)
	httpRouter.AddRoute("PATCH", "/api/resource/updateproject", updateproject.HandleUpdateProject)
	httpRouter.AddRoute("DELETE", "/api/resource/deleteproject", deleteproject.HandleDeleteProject)

	lambda.Start(httpRouter.Route)

	return nil
}
