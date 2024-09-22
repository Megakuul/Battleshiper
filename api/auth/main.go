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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/router"
	"golang.org/x/oauth2/github"

	"github.com/megakuul/battleshiper/api/auth/authorize"
	"github.com/megakuul/battleshiper/api/auth/callback"
	"github.com/megakuul/battleshiper/api/auth/logout"
	"github.com/megakuul/battleshiper/api/auth/refresh"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
)

var (
	REGION                       = os.Getenv("AWS_REGION")
	USERTABLE                    = os.Getenv("USERTABLE")
	PROJECTTABLE                 = os.Getenv("PROJECTTABLE")
	JWT_CREDENTIAL_ARN           = os.Getenv("JWT_CREDENTIAL_ARN")
	USER_TOKEN_TTL               = os.Getenv("USER_TOKEN_TTL")
	GITHUB_CLIENT_CREDENTIAL_ARN = os.Getenv("GITHUB_CLIENT_CREDENTIAL_ARN")
	REDIRECT_URI                 = os.Getenv("REDIRECT_URI")
	FRONTEND_REDIRECT_URI        = os.Getenv("FRONTEND_REDIRECT_URI")
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
		return err
	}

	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	userTokenTTL, err := strconv.Atoi(USER_TOKEN_TTL)
	if err != nil {
		return fmt.Errorf("failed to parse USER_TOKEN_TTL environment variable")
	}
	jwtOptions, err := auth.CreateJwtOptions(awsConfig, context.TODO(), JWT_CREDENTIAL_ARN, time.Duration(userTokenTTL)*time.Second)
	if err != nil {
		return err
	}

	authOptions, err := auth.CreateOAuthOptions(awsConfig, context.TODO(), GITHUB_CLIENT_CREDENTIAL_ARN, github.Endpoint, REDIRECT_URI, []string{"read:user"})
	if err != nil {
		return err
	}

	httpRouter := router.NewRouter(routecontext.Context{
		DynamoClient:        dynamoClient,
		UserTable:           USERTABLE,
		ProjectTable:        PROJECTTABLE,
		JwtOptions:          jwtOptions,
		OAuthConfig:         authOptions,
		FrontendRedirectURI: FRONTEND_REDIRECT_URI,
	})

	httpRouter.AddRoute("GET", "/api/auth/authorize", authorize.HandleAuthorization)
	httpRouter.AddRoute("GET", "/api/auth/callback", callback.HandleCallback)
	httpRouter.AddRoute("POST", "/api/auth/refresh", refresh.HandleRefresh)
	httpRouter.AddRoute("POST", "/api/auth/logout", logout.HandleLogout)

	lambda.Start(httpRouter.Route)

	return nil
}
