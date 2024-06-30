package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/megakuul/battleshiper/lib/router"

	"github.com/megakuul/battleshiper/api/user/info"
	"github.com/megakuul/battleshiper/api/user/routecontext"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	REGION         = os.Getenv("AWS_REGION")
	COGNITO_DOMAIN = os.Getenv("COGNITO_DOMAIN")
	CLIENT_ID      = os.Getenv("CLIENT_ID")
	CLIENT_SECRET  = os.Getenv("CLIENT_SECRET")
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
	awsCognitoClient := cognitoidentityprovider.NewFromConfig(awsConfig)

	databaseClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
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

	httpRouter := router.NewRouter[routecontext.Context](routecontext.Context{
		DatabaseClient: databaseClient,
		CognitoClient:  awsCognitoClient,
		CognitoDomain:  COGNITO_DOMAIN,
		ClientID:       CLIENT_ID,
		ClientSecret:   CLIENT_SECRET,
	})

	httpRouter.AddRoute("GET", "/api/user/info", info.HandleInfo)

	lambda.Start(httpRouter.Route)

	return nil
}
