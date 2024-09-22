package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/router"

	"github.com/megakuul/battleshiper/api/user/fetchinfo"
	"github.com/megakuul/battleshiper/api/user/registeruser"
	"github.com/megakuul/battleshiper/api/user/routecontext"
)

var (
	REGION                = os.Getenv("AWS_REGION")
	BOOTSTRAP_TIMEOUT     = os.Getenv("BOOTSTRAP_TIMEOUT")
	USERTABLE             = os.Getenv("USERTABLE")
	SUBSCRIPTIONTABLE     = os.Getenv("SUBSCRIPTIONTABLE")
	JWT_CREDENTIAL_ARN    = os.Getenv("JWT_CREDENTIAL_ARN")
	ADMIN_GITHUB_USERNAME = os.Getenv("ADMIN_GITHUB_USERNAME")
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

	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	jwtOptions, err := auth.CreateJwtOptions(awsConfig, bootstrapContext, JWT_CREDENTIAL_ARN, 0)
	if err != nil {
		return err
	}

	httpRouter := router.NewRouter(routecontext.Context{
		DynamoClient:      dynamoClient,
		UserTable:         USERTABLE,
		SubscriptionTable: SUBSCRIPTIONTABLE,
		JwtOptions:        jwtOptions,
		UserConfiguration: &routecontext.UserConfiguration{
			AdminUsername: ADMIN_GITHUB_USERNAME,
		},
	})

	httpRouter.AddRoute("GET", "/api/user/fetchinfo", fetchinfo.HandleFetchInfo)
	httpRouter.AddRoute("POST", "/api/user/registeruser", registeruser.HandleRegisterUser)

	lambda.Start(httpRouter.Route)

	return nil
}
