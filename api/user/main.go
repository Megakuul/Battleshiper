package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/megakuul/battleshiper/api/user/info"
	"github.com/megakuul/battleshiper/api/user/router"
)

var (
	REGION         = os.Getenv("AWS_REGION")
	COGNITO_DOMAIN = os.Getenv("COGNITO_DOMAIN")
	CLIENT_ID      = os.Getenv("CLIENT_ID")
	CLIENT_SECRET  = os.Getenv("CLIENT_SECRET")
)

func main() {
	awsConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(REGION))
	if err != nil {
		log.Printf("ERROR INITIALIZATION: Failed to load sdk configuration: %v\n", err)
	}
	awsCognitoClient := cognitoidentityprovider.NewFromConfig(awsConfig)

	httpRouter := router.NewRouter(router.RouteContext{
		CognitoClient: awsCognitoClient,
		CognitoDomain: COGNITO_DOMAIN,
		ClientID:      CLIENT_ID,
		ClientSecret:  CLIENT_SECRET,
	})

	httpRouter.AddRoute("GET", "/api/user/info", info.HandleInfo)

	lambda.Start(httpRouter.Route)
}
