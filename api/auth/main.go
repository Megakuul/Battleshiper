package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/megakuul/battleshiper/api/auth/authorize"
	"github.com/megakuul/battleshiper/api/auth/callback"
	"github.com/megakuul/battleshiper/api/auth/logout"
	"github.com/megakuul/battleshiper/api/auth/refresh"
	"github.com/megakuul/battleshiper/api/auth/router"
)

var (
	REGION                = os.Getenv("AWS_REGION")
	COGNITO_DOMAIN        = os.Getenv("COGNITO_DOMAIN")
	CLIENT_ID             = os.Getenv("CLIENT_ID")
	CLIENT_SECRET         = os.Getenv("CLIENT_SECRET")
	REDIRECT_URI          = os.Getenv("REDIRECT_URI")
	FRONTEND_REDIRECT_URI = os.Getenv("FRONTEND_REDIRECT_URI")
)

func main() {
	awsConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(REGION))
	if err != nil {
		log.Printf("ERROR INITIALIZATION: Failed to load sdk configuration: %v\n", err)
	}
	awsCognitoClient := cognitoidentityprovider.NewFromConfig(awsConfig)

	httpRouter := router.NewRouter(router.RouteContext{
		CognitoClient:       awsCognitoClient,
		CognitoDomain:       COGNITO_DOMAIN,
		ClientID:            CLIENT_ID,
		ClientSecret:        CLIENT_SECRET,
		RedirectURI:         REDIRECT_URI,
		FrontendRedirectURI: FRONTEND_REDIRECT_URI,
	})

	httpRouter.AddRoute("GET", "/api/auth/authorize", authorize.HandleAuthorization)
	httpRouter.AddRoute("GET", "/api/auth/callback", callback.HandleCallback)
	httpRouter.AddRoute("POST", "/api/auth/refresh", refresh.HandleRefresh)
	httpRouter.AddRoute("POST", "/api/auth/logout", logout.HandleLogout)

	lambda.Start(httpRouter.Route)
}
