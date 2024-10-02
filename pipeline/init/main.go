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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/pipeline/init/eventcontext"
	"github.com/megakuul/battleshiper/pipeline/init/initproject"
)

var (
	REGION                      = os.Getenv("AWS_REGION")
	BOOTSTRAP_TIMEOUT           = os.Getenv("BOOTSTRAP_TIMEOUT")
	USERTABLE                   = os.Getenv("USERTABLE")
	PROJECTTABLE                = os.Getenv("PROJECTTABLE")
	SUBSCRIPTIONTABLE           = os.Getenv("SUBSCRIPTIONTABLE")
	TICKET_CREDENTIAL_ARN       = os.Getenv("TICKET_CREDENTIAL_ARN")
	DEPLOYMENT_SERVICE_ROLE_ARN = os.Getenv("DEPLOYMENT_SERVICE_ROLE_ARN")
	DEPLOYMENT_TIMEOUT          = os.Getenv("DEPLOYMENT_TIMEOUT")
	STATIC_BUCKET_NAME          = os.Getenv("STATIC_BUCKET_NAME")
	BUILD_ASSET_BUCKET_NAME     = os.Getenv("BUILD_ASSET_BUCKET_NAME")
	EVENT_LOG_GROUP_PREFIX      = os.Getenv("EVENT_LOG_GROUP_PREFIX")
	BUILD_LOG_GROUP_PREFIX      = os.Getenv("BUILD_LOG_GROUP_PREFIX")
	DEPLOY_LOG_GROUP_PREFIX     = os.Getenv("DEPLOY_LOG_GROUP_PREFIX")
	SERVER_LOG_GROUP_PREFIX     = os.Getenv("SERVER_LOG_GROUP_PREFIX")
	LOG_GROUP_RETENTION_DAYS    = os.Getenv("LOG_GROUP_RETENTION_DAYS")
	BUILD_EVENTBUS_NAME         = os.Getenv("BUILD_EVENTBUS_NAME")
	BUILD_EVENT_SOURCE          = os.Getenv("BUILD_EVENT_SOURCE")
	BUILD_EVENT_ACTION          = os.Getenv("BUILD_EVENT_ACTION")
	BUILD_QUEUE_ARN             = os.Getenv("BUILD_QUEUE_ARN")
	BUILD_JOB_TIMEOUT           = os.Getenv("BUILD_JOB_TIMEOUT")
	BUILD_JOB_VCPUS             = os.Getenv("BUILD_JOB_VCPUS")
	BUILD_JOB_MEMORY            = os.Getenv("BUILD_JOB_MEMORY")
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

	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	deploymentTimeout, err := time.ParseDuration(DEPLOYMENT_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse DEPLOYMENT_TIMEOUT environment variable")
	}

	logGroupRetentionDays, err := strconv.Atoi(LOG_GROUP_RETENTION_DAYS)
	if err != nil {
		return fmt.Errorf("failed to parse LOG_GROUP_RETENTION_DAYS environment variable")
	}

	buildJobTimeout, err := time.ParseDuration(BUILD_JOB_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse BUILD_JOB_TIMEOUT environment variable")
	}

	ticketOptions, err := pipeline.CreateTicketOptions(awsConfig, bootstrapContext, TICKET_CREDENTIAL_ARN, "", "", 0)
	if err != nil {
		return err
	}

	lambda.Start(initproject.HandleInitProject(eventcontext.Context{
		DynamoClient:         dynamoClient,
		UserTable:            USERTABLE,
		ProjectTable:         PROJECTTABLE,
		SubscriptionTable:    SUBSCRIPTIONTABLE,
		TicketOptions:        ticketOptions,
		CloudformationClient: cloudformationClient,
		DeploymentConfiguration: &eventcontext.DeploymentConfiguration{
			ServiceRoleArn: DEPLOYMENT_SERVICE_ROLE_ARN,
			Timeout:        deploymentTimeout,
		},
		BucketConfiguration: &eventcontext.BucketConfiguration{
			StaticBucketName:     STATIC_BUCKET_NAME,
			BuildAssetBucketName: BUILD_ASSET_BUCKET_NAME,
		},
		ProjectConfiguration: &eventcontext.ProjectConfiguration{
			EventLogPrefix:    EVENT_LOG_GROUP_PREFIX,
			BuildLogPrefix:    BUILD_LOG_GROUP_PREFIX,
			DeployLogPrefix:   DEPLOY_LOG_GROUP_PREFIX,
			ServerLogPrefix:   SERVER_LOG_GROUP_PREFIX,
			LogRetentionDays:  logGroupRetentionDays,
			BuildEventbusName: BUILD_EVENTBUS_NAME,
			BuildEventSource:  BUILD_EVENT_SOURCE,
			BuildEventAction:  BUILD_EVENT_ACTION,
			BuildJobQueueArn:  BUILD_QUEUE_ARN,
			BuildJobTimeout:   buildJobTimeout,
			BuildJobVCPUS:     BUILD_JOB_VCPUS,
			BuildJobMemory:    BUILD_JOB_MEMORY,
		},
	}))

	return nil
}
