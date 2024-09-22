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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"

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
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/router"
)

var (
	REGION                  = os.Getenv("AWS_REGION")
	BOOTSTRAP_TIMEOUT       = os.Getenv("BOOTSTRAP_TIMEOUT")
	USERTABLE               = os.Getenv("USERTABLE")
	PROJECTTABLE            = os.Getenv("PROJECTTABLE")
	SUBSCRIPTIONTABLE       = os.Getenv("SUBSCRIPTIONTABLE")
	JWT_CREDENTIAL_ARN      = os.Getenv("JWT_CREDENTIAL_ARN")
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
	DELETE_EVENTBUS_NAME    = os.Getenv("DELETE_EVENTBUS_NAME")
	DELETE_EVENT_SOURCE     = os.Getenv("DELETE_EVENT_SOURCE")
	DELETE_EVENT_ACTION     = os.Getenv("DELETE_EVENT_ACTION")
	DELETE_EVENT_TICKET_TTL = os.Getenv("DELETE_EVENT_TICKET_TTL")
	CLOUDFRONT_CACHE_ARN    = os.Getenv("CLOUDFRONT_CACHE_ARN")
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

	cloudwatchClient := cloudwatchlogs.NewFromConfig(awsConfig)

	eventClient := eventbridge.NewFromConfig(awsConfig)

	cloudfrontClient := cloudfrontkeyvaluestore.NewFromConfig(awsConfig)

	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	jwtOptions, err := auth.CreateJwtOptions(awsConfig, bootstrapContext, JWT_CREDENTIAL_ARN, 0)
	if err != nil {
		return err
	}

	initTicketTTL, err := strconv.Atoi(INIT_EVENT_TICKET_TTL)
	if err != nil {
		return fmt.Errorf("failed to parse INIT_EVENT_TICKET_TTL environment variable")
	}
	initTicketOptions, err := pipeline.CreateTicketOptions(
		awsConfig, bootstrapContext, TICKET_CREDENTIAL_ARN, INIT_EVENT_SOURCE, INIT_EVENT_ACTION, time.Duration(initTicketTTL)*time.Second)
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
		awsConfig, bootstrapContext, TICKET_CREDENTIAL_ARN, DEPLOY_EVENT_SOURCE, DEPLOY_EVENT_ACTION, time.Duration(deployTicketTTL)*time.Second)
	if err != nil {
		return err
	}

	deleteTicketTTL, err := strconv.Atoi(DELETE_EVENT_TICKET_TTL)
	if err != nil {
		return fmt.Errorf("failed to parse DELETE_EVENT_TICKET_TTL environment variable")
	}
	deleteTicketOptions, err := pipeline.CreateTicketOptions(
		awsConfig, bootstrapContext, TICKET_CREDENTIAL_ARN, DELETE_EVENT_SOURCE, DELETE_EVENT_ACTION, time.Duration(deleteTicketTTL)*time.Second)
	if err != nil {
		return err
	}
	deleteEventOptions := pipeline.CreateEventOptions(DELETE_EVENTBUS_NAME, DELETE_EVENT_SOURCE, DELETE_EVENT_ACTION, deleteTicketOptions)

	httpRouter := router.NewRouter(routecontext.Context{
		DynamoClient:          dynamoClient,
		UserTable:             USERTABLE,
		ProjectTable:          PROJECTTABLE,
		SubscriptionTable:     SUBSCRIPTIONTABLE,
		CloudwatchClient:      cloudwatchClient,
		JwtOptions:            jwtOptions,
		EventClient:           eventClient,
		InitEventOptions:      initEventOptions,
		BuildEventOptions:     buildEventOptions,
		DeployTicketOptions:   deployTicketOptions,
		DeleteEventOptions:    deleteEventOptions,
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
