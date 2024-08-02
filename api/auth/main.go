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
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/user"
	"github.com/megakuul/battleshiper/lib/router"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/oauth2/github"

	"github.com/megakuul/battleshiper/api/auth/authorize"
	"github.com/megakuul/battleshiper/api/auth/callback"
	"github.com/megakuul/battleshiper/api/auth/logout"
	"github.com/megakuul/battleshiper/api/auth/refresh"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
)

var (
	REGION                       = os.Getenv("AWS_REGION")
	JWT_CREDENTIAL_ARN           = os.Getenv("JWT_CREDENTIAL_ARN")
	USER_TOKEN_TTL               = os.Getenv("USER_TOKEN_TTL")
	GITHUB_CLIENT_CREDENTIAL_ARN = os.Getenv("GITHUB_CLIENT_CREDENTIAL_ARN")
	REDIRECT_URI                 = os.Getenv("REDIRECT_URI")
	FRONTEND_REDIRECT_URI        = os.Getenv("FRONTEND_REDIRECT_URI")
	DATABASE_ENDPOINT            = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME                = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN          = os.Getenv("DATABASE_SECRET_ARN")
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
		{
			FieldNames:   []string{"id"},
			SortingOrder: 1,
			Unique:       true,
		},
	})

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
		Database:            databaseHandle,
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
