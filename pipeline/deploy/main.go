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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/pipeline/deploy/deployproject"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

var (
	REGION                = os.Getenv("AWS_REGION")
	BOOTSTRAP_TIMEOUT     = os.Getenv("BOOTSTRAP_TIMEOUT")
	USERTABLE             = os.Getenv("USERTABLE")
	PROJECTTABLE          = os.Getenv("PROJECTTABLE")
	SUBSCRIPTIONTABLE     = os.Getenv("SUBSCRIPTIONTABLE")
	TICKET_CREDENTIAL_ARN = os.Getenv("TICKET_CREDENTIAL_ARN")
	CHANGESET_TIMEOUT     = os.Getenv("CHANGESET_TIMEOUT")
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

	cloudformationClient := cloudformation.NewFromConfig(awsConfig)

	cloudwatchClient := cloudwatchlogs.NewFromConfig(awsConfig)

	s3Client := s3.NewFromConfig(awsConfig)

	cloudfrontClient := cloudfrontkeyvaluestore.NewFromConfig(awsConfig)

	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	ticketOptions, err := pipeline.CreateTicketOptions(awsConfig, bootstrapContext, TICKET_CREDENTIAL_ARN, "", "", 0)
	if err != nil {
		return err
	}

	changesetTimeout, err := time.ParseDuration(CHANGESET_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse CHANGESET_TIMEOUT environment variable")
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
		DynamoClient:          dynamoClient,
		UserTable:             USERTABLE,
		ProjectTable:          PROJECTTABLE,
		SubscriptionTable:     SUBSCRIPTIONTABLE,
		TicketOptions:         ticketOptions,
		CloudformationClient:  cloudformationClient,
		S3Client:              s3Client,
		CloudwatchClient:      cloudwatchClient,
		CloudfrontCacheClient: cloudfrontClient,
		DeploymentConfiguration: &eventcontext.DeploymentConfiguration{
			ChangeSetTimeout:  changesetTimeout,
			DeplyomentTimeout: deploymentTimeout,
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
