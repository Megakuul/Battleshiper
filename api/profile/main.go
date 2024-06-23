package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	userPoolID  = "your_user_pool_id"
	region      = "your_aws_region"
	appClientID = "your_app_client_id"
)

func handler(request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {

	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintln(request.Cookies),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
