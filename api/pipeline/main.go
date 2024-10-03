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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"

	"github.com/megakuul/battleshiper/api/pipeline/event"
	"github.com/megakuul/battleshiper/api/pipeline/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/router"
)

var (
	REGION                       = os.Getenv("AWS_REGION")
	BOOTSTRAP_TIMEOUT            = os.Getenv("BOOTSTRAP_TIMEOUT")
	USERTABLE                    = os.Getenv("USERTABLE")
	PROJECTTABLE                 = os.Getenv("PROJECTTABLE")
	SUBSCRIPTIONTABLE            = os.Getenv("SUBSCRIPTIONTABLE")
	GITHUB_CLIENT_CREDENTIAL_ARN = os.Getenv("GITHUB_CLIENT_CREDENTIAL_ARN")
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

	eventbridgeClient := eventbridge.NewFromConfig(awsConfig)

	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	webhookClient, err := auth.CreateGithubWebhookClient(awsConfig, bootstrapContext, GITHUB_CLIENT_CREDENTIAL_ARN)
	if err != nil {
		return err
	}

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

	githubAppOptions, err := auth.CreateGithubAppOptions(awsConfig, bootstrapContext, GITHUB_CLIENT_CREDENTIAL_ARN)
	if err != nil {
		return err
	}

	httpRouter := router.NewRouter(routecontext.Context{
		DynamoClient:        dynamoClient,
		UserTable:           USERTABLE,
		ProjectTable:        PROJECTTABLE,
		SubscriptionTable:   SUBSCRIPTIONTABLE,
		WebhookClient:       webhookClient,
		GithubAppOptions:    githubAppOptions,
		CloudwatchClient:    cloudwatchClient,
		EventClient:         eventbridgeClient,
		BuildEventOptions:   buildEventOptions,
		DeployTicketOptions: deployTicketOptions,
	})

	httpRouter.AddRoute("POST", "/api/pipeline/event", event.HandleEvent)

	lambda.Start(httpRouter.Route)

	return nil
}
