package routerequest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/megakuul/battleshiper/api/user/routecontext"
)

var logger = log.New(os.Stderr, "ROUTEREQUEST: ", 0)

// HandleRouteRequest routes request either to s3 or to the corresponding server function.
func HandleRouteRequest(routeCtx routecontext.Context) func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return func(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return runHandleRouteRequest(request, ctx, routeCtx)
	}
}

func runHandleRouteRequest(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	project := request.Headers["Battleshiper-Project"]

	if strings.HasSuffix(request.RawPath, ".html") && request.RequestContext.HTTP.Method == "GET" {
		response, code, err := proxyStatic(request, transportCtx, routeCtx, project)
		if err != nil {
			return events.APIGatewayV2HTTPResponse{
				StatusCode: code,
				Headers:    map[string]string{"Content-Type": "text/plain"},
				Body:       err.Error(),
			}, nil
		}
		response.StatusCode = code
		return *response, nil
	}

	response, code, err := proxyServer(request, transportCtx, routeCtx, project)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: code,
			Headers:    map[string]string{"Content-Type": "text/plain"},
			Body:       err.Error(),
		}, nil
	}
	response.StatusCode = code
	return *response, nil
}

// proxyStatic reads the requested path from the static s3 bucket and returns it as Content-Type text/html.
func proxyStatic(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context, projectName string) (*events.APIGatewayV2HTTPResponse, int, error) {
	objectOutput, err := routeCtx.S3Client.GetObject(transportCtx, &s3.GetObjectInput{
		Bucket: aws.String(routeCtx.StaticBucketName),
		Key:    aws.String(fmt.Sprintf("%s%s", projectName, request.RawPath)),
	})
	if err != nil {
		var nsk *s3types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, http.StatusNotFound, fmt.Errorf("static asset not found")
		} else {
			logger.Printf("failed to load static asset: %v\n", err)
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to load static asset")
		}
	}

	body, err := io.ReadAll(objectOutput.Body)
	if err != nil {
		logger.Printf("failed to read static asset data: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to read static asset data")
	}
	return &events.APIGatewayV2HTTPResponse{
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
		Body: string(body),
	}, http.StatusOK, nil
}

// proxyServer invokes the origin server function (LambdaPrefix-ProjectName) and returns the server response.
func proxyServer(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context, projectName string) (*events.APIGatewayV2HTTPResponse, int, error) {
	requestRaw, err := json.Marshal(request)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to serialize api request")
	}

	result, err := routeCtx.FunctionClient.Invoke(transportCtx, &lambda.InvokeInput{
		FunctionName:   aws.String(fmt.Sprintf("%s%s", routeCtx.ServerNamePrefix, projectName)),
		Payload:        requestRaw,
		InvocationType: lambdatypes.InvocationTypeRequestResponse,
	})
	if err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("failed to invoke origin server")
	}
	if result.FunctionError != nil && *result.FunctionError != "" {
		logger.Printf("origin server failed to handle request: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("origin server failed to handle request")
	}

	response := &events.APIGatewayV2HTTPResponse{}
	err = json.Unmarshal(result.Payload, response)
	if err != nil {
		logger.Printf("failed to deserialize api response: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to deserialize api response")
	}

	return response, response.StatusCode, nil
}
