package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/softlayer/softlayer-go/session"
)

var (
	CLIENT_ID      = os.Getenv("CLIENT_ID")
	CLIENT_SECRET  = os.Getenv("CLIENT_SECRET")
	REDIRECT_URI   = os.Getenv("REDIRECT_URI")
	COGNITO_DOMAIN = os.Getenv("COGNITO_DOMAIN")
	REGION         = os.Getenv("AWS_REGION")
)

// Using global variables is as of my knowledge the practice used to share an object between
// multiple invocations: https://www.alexedwards.net/blog/serverless-api-with-go-and-aws-lambda.
var AwsConfig aws.Config
var AwsCognitoClient *cognitoidentityprovider.Client

func handler(request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	claims := request.RequestContext.Authorizer.JWT.Claims
	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintln(claims),
		StatusCode: 200,
	}, nil
}

func getUserInfo(accessToken, region string) (*cognitoidentityprovider.GetUserOutput, error) {
	svc := cognitoidentityprovider.New(session.New(), aws.NewConfig().WithRegion(region))
	req := &cognitoidentityprovider.GetUserInput{
		AccessToken: accessToken,
	}
	res, err := svc.GetUser(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func main() {
	AwsConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(REGION))
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}
	AwsCognitoClient = cognitoidentityprovider.NewFromConfig(AwsConfig)

	lambda.Start(handler)
}
