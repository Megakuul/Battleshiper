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
	TICKET_CREDENTIAL_ARN  = os.Getenv("TICKET_CREDENTIAL_ARN")
	DEPLOYMENT_TIMEOUT     = os.Getenv("DEPLOYMENT_TIMEOUT")
	BUILD_EVENTBUS_NAME    = os.Getenv("BUILD_EVENTBUS_NAME")
	BUILD_EVENT_SOURCE     = os.Getenv("BUILD_EVENT_SOURCE")
	BUILD_EVENT_ACTION     = os.Getenv("BUILD_EVENT_ACTION")
	BUILD_QUEUE_ARN        = os.Getenv("BUILD_QUEUE_ARN")
	BUILD_QUEUE_POLICY_ARN = os.Getenv("BUILD_QUEUE_POLICY_ARN")
	BUILD_JOB_TIMEOUT      = os.Getenv("BUILD_JOB_TIMEOUT")
	BUILD_JOB_VCPUS        = os.Getenv("BUILD_JOB_VCPUS")
	BUILD_JOB_MEMORY       = os.Getenv("BUILD_JOB_MEMORY")
	DEPLOY_EVENTBUS_NAME   = os.Getenv("DEPLOY_EVENTBUS_NAME")
	DEPLOY_EVENT_SOURCE    = os.Getenv("DEPLOY_EVENT_SOURCE")
	DEPLOY_EVENT_ACTION    = os.Getenv("DEPLOY_EVENT_ACTION")
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

	deploymentTimeout, err := time.ParseDuration(DEPLOYMENT_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse DEPLOYMENT_TIMEOUT environment variable")
	}

	buildJobTimeout, err := time.ParseDuration(BUILD_JOB_TIMEOUT)
	if err != nil {
		return fmt.Errorf("failed to parse BUILD_JOB_TIMEOUT environment variable")
	}

	buildJobVcpus, err := strconv.Atoi(BUILD_JOB_VCPUS)
	if err != nil {
		return fmt.Errorf("failed to parse BUILD_JOB_VCPUS environment variable")
	}

	buildJobMemory, err := strconv.Atoi(BUILD_JOB_MEMORY)
	if err != nil {
		return fmt.Errorf("failed to parse BUILD_JOB_MEMORY environment variable")
	}

	ticketOptions, err := pipeline.CreateTicketOptions(awsConfig, context.TODO(), TICKET_CREDENTIAL_ARN, "", 0)
	if err != nil {
		return err
	}

	lambda.Start(initproject.HandleInitProject(eventcontext.Context{
		Database:             databaseHandle,
		TicketOptions:        ticketOptions,
		CloudformationClient: cloudformationClient,
		DeploymentTimeout:    deploymentTimeout,
		DeployConfiguration: &eventcontext.DeployConfiguration{
			DeployEventbusName: DEPLOY_EVENTBUS_NAME,
			DeployEventSource:  DEPLOY_EVENT_SOURCE,
			DeployEventAction:  DEPLOY_EVENT_ACTION,
		},
		BuildConfiguration: &eventcontext.BuildConfiguration{
			BuildEventbusName:      BUILD_EVENTBUS_NAME,
			BuildEventSource:       BUILD_EVENT_SOURCE,
			BuildEventAction:       BUILD_EVENT_ACTION,
			BuildJobQueueArn:       BUILD_QUEUE_ARN,
			BuildJobQueuePolicyArn: BUILD_QUEUE_POLICY_ARN,
			BuildJobTimeout:        buildJobTimeout,
			BuildJobVCPUS:          buildJobVcpus,
			BuildJobMemory:         buildJobMemory,
		},
	}))

	return nil
}
