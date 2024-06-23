package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	clientID      = os.Getenv("CLIENT_ID")
	clientSecret  = os.Getenv("CLIENT_SECRET")
	redirectURI   = os.Getenv("REDIRECT_URI")
	cognitoDomain = os.Getenv("COGNITO_DOMAIN")
	region        = os.Getenv("AWS_REGION")
)

func handler(request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	claims := request.RequestContext.Authorizer.JWT.Claims
	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintln(claims),
		StatusCode: 200,
	}, nil
}

func AcquireTokens(authCode string) (string, string, error) {
	// Do HTTP request for cognito
}

func main() {
	lambda.Start(handler)
}
