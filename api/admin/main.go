package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/router"

	"github.com/megakuul/battleshiper/api/admin/deleteproject"
	"github.com/megakuul/battleshiper/api/admin/deleteuser"
	"github.com/megakuul/battleshiper/api/admin/fetchlog"
	"github.com/megakuul/battleshiper/api/admin/findproject"
	"github.com/megakuul/battleshiper/api/admin/finduser"
	"github.com/megakuul/battleshiper/api/admin/listsubscription"
	"github.com/megakuul/battleshiper/api/admin/routecontext"
	"github.com/megakuul/battleshiper/api/admin/updaterole"
	"github.com/megakuul/battleshiper/api/admin/updateuser"
	"github.com/megakuul/battleshiper/api/admin/upsertsubscription"
)

var (
	REGION             = os.Getenv("AWS_REGION")
	USERTABLE          = os.Getenv("USERTABLE")
	PROJECTTABLE       = os.Getenv("PROJECTTABLE")
	SUBSCRIPTIONTABLE  = os.Getenv("SUBSCRIPTIONTABLE")
	JWT_CREDENTIAL_ARN = os.Getenv("JWT_CREDENTIAL_ARN")
	API_LOG_GROUP      = os.Getenv("API_LOG_GROUP")
	PIPELINE_LOG_GROUP = os.Getenv("PIPELINE_LOG_GROUP")
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

	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	jwtOptions, err := auth.CreateJwtOptions(awsConfig, context.TODO(), JWT_CREDENTIAL_ARN, 0)
	if err != nil {
		return err
	}

	httpRouter := router.NewRouter(routecontext.Context{
		DynamoClient:      dynamoClient,
		UserTable:         USERTABLE,
		ProjectTable:      PROJECTTABLE,
		SubscriptionTable: SUBSCRIPTIONTABLE,
		CloudwatchClient:  cloudwatchClient,
		JwtOptions:        jwtOptions,
		LogConfiguration: &routecontext.LogConfiguration{
			ApiLogGroup:      API_LOG_GROUP,
			PipelineLogGroup: PIPELINE_LOG_GROUP,
		},
	})

	httpRouter.AddRoute("GET", "/api/admin/finduser", finduser.HandleFindUser)
	httpRouter.AddRoute("GET", "/api/admin/findproject", findproject.HandleFindProject)
	httpRouter.AddRoute("PATCH", "/api/admin/updateuser", updateuser.HandleUpdateUser)
	httpRouter.AddRoute("PATCH", "/api/admin/updaterole", updaterole.HandleUpdateRole)
	httpRouter.AddRoute("GET", "/api/admin/fetchlog", fetchlog.HandleFetchLog)
	httpRouter.AddRoute("GET", "/api/admin/listsubscription", listsubscription.HandleListSubscription)
	httpRouter.AddRoute("PUT", "/api/admin/upsertsubscription", upsertsubscription.HandleUpsertSubscription)
	httpRouter.AddRoute("DELETE", "/api/admin/deleteuser", deleteuser.HandleDeleteUser)
	httpRouter.AddRoute("DELETE", "/api/admin/deleteproject", deleteproject.HandleDeleteProject)

	lambda.Start(httpRouter.Route)

	return nil
}
