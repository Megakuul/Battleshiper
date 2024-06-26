package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/index"
	"github.com/megakuul/battleshiper/lib/model/user"
	"github.com/megakuul/battleshiper/lib/router"

	"github.com/megakuul/battleshiper/api/user/info"
	"github.com/megakuul/battleshiper/api/user/routecontext"
)

var (
	REGION              = os.Getenv("AWS_REGION")
	COGNITO_DOMAIN      = os.Getenv("COGNITO_DOMAIN")
	CLIENT_ID           = os.Getenv("CLIENT_ID")
	CLIENT_SECRET       = os.Getenv("CLIENT_SECRET")
	DATABASE_ENDPOINT   = os.Getenv("DATABASE_ENDPOINT")
	DATABASE_NAME       = os.Getenv("DATABASE_NAME")
	DATABASE_SECRET_ARN = os.Getenv("DATABASE_SECRET_ARN")
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
	awsCognitoClient := cognitoidentityprovider.NewFromConfig(awsConfig)

	databaseOptions, err := database.CreateDatabaseOptions(awsConfig, DATABASE_SECRET_ARN, DATABASE_ENDPOINT, DATABASE_NAME)
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

	index.SetupIndexes(databaseHandle.Collection("user"), context.TODO(), bson.D{user.UserIndex{Sub: 1}}, true)

	httpRouter := router.NewRouter(routecontext.Context{
		Database:      databaseHandle,
		CognitoClient: awsCognitoClient,
		CognitoDomain: COGNITO_DOMAIN,
		ClientID:      CLIENT_ID,
		ClientSecret:  CLIENT_SECRET,
	})

	httpRouter.AddRoute("GET", "/api/user/info", info.HandleInfo)

	lambda.Start(httpRouter.Route)

	return nil
}
