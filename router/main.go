package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	function "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/megakuul/battleshiper/api/user/routecontext"
	"github.com/megakuul/battleshiper/api/user/routerequest"
)

var (
	REGION             = os.Getenv("AWS_REGION")
	STATIC_BUCKET_NAME = os.Getenv("STATIC_BUCKET_NAME")
	SERVER_NAME_PREFIX = os.Getenv("SERVER_NAME_PREFIX")
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

	s3Client := s3.NewFromConfig(awsConfig)

	functionClient := function.NewFromConfig(awsConfig)

	lambda.Start(routerequest.HandleRouteRequest(routecontext.Context{
		S3Client:         s3Client,
		StaticBucketName: STATIC_BUCKET_NAME,
		FunctionClient:   functionClient,
		ServerNamePrefix: SERVER_NAME_PREFIX,
	}))

	return nil
}
