package routerequest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/megakuul/battleshiper/api/user/routecontext"
)

type AdapterRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type AdapterResponse struct {
	StatusCode        int               `json:"status_code"`
	StatusDescription string            `json:"status_description"`
	Headers           map[string]string `json:"headers"`
	Body              string            `json:"body"`
}

// HandleRouteRequest routes request either to s3 or to the corresponding server function.
func HandleRouteRequest(routeCtx routecontext.Context) func(context.Context, events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	return func(ctx context.Context, request events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
		return runHandleRouteRequest(request, ctx, routeCtx)
	}
}

func runHandleRouteRequest(request events.ALBTargetGroupRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.ALBTargetGroupResponse, error) {
	project := request.Headers["Battleshiper-Project"]

	if strings.HasSuffix(request.Path, ".html") && request.HTTPMethod == "GET" {
		response, code, err := proxyStatic(request, transportCtx, routeCtx)
		if err != nil {
			return events.ALBTargetGroupResponse{
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
		return events.ALBTargetGroupResponse{
			StatusCode: code,
			Headers:    map[string]string{"Content-Type": "text/plain"},
			Body:       err.Error(),
		}, nil
	}
	response.StatusCode = code
	return *response, nil
}

// proxyStatic reads the requested path from the static s3 bucket and returns it as Content-Type text/html.
func proxyStatic(request events.ALBTargetGroupRequest, transportCtx context.Context, routeCtx routecontext.Context) (*events.ALBTargetGroupResponse, int, error) {
	objectOutput, err := routeCtx.S3Client.GetObject(transportCtx, &s3.GetObjectInput{
		Bucket: aws.String(routeCtx.S3Bucket),
		Key:    aws.String(request.Path),
	})
	if err != nil {
		var nsk *s3types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, http.StatusNotFound, fmt.Errorf("static asset not found")
		} else {
			return nil, http.StatusInternalServerError, err
		}
	}

	body, err := io.ReadAll(objectOutput.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return &events.ALBTargetGroupResponse{
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
		Body: string(body),
	}, http.StatusOK, nil
}

// proxyServer invokes the origin server function (LambdaPrefix-ProjectName) and returns the server response.
func proxyServer(request events.ALBTargetGroupRequest, transportCtx context.Context, routeCtx routecontext.Context, projectName string) (*events.ALBTargetGroupResponse, int, error) {
	adapterRequest := &AdapterRequest{
		Method:  request.HTTPMethod,
		Path:    request.Path,
		Headers: request.Headers,
		Body:    request.Body,
	}

	adapterRequestRaw, err := json.Marshal(adapterRequest)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to serialize adapter request")
	}

	result, err := routeCtx.FunctionClient.Invoke(transportCtx, &lambda.InvokeInput{
		FunctionName:   aws.String(fmt.Sprintf("%s-%s", routeCtx.FunctionPrefix, projectName)),
		Payload:        adapterRequestRaw,
		InvocationType: lambdatypes.InvocationTypeRequestResponse,
	})
	if err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("failed to invoke origin server")
	}
	if result.FunctionError != nil && *result.FunctionError != "" {
		return nil, http.StatusInternalServerError, fmt.Errorf(*result.FunctionError)
	}

	adapterResponse := &AdapterResponse{}
	err = json.Unmarshal(result.Payload, adapterResponse)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to deserialize adapter response")
	}

	return &events.ALBTargetGroupResponse{
		StatusDescription: adapterResponse.StatusDescription,
		Headers:           adapterResponse.Headers,
		Body:              adapterResponse.Body,
	}, adapterResponse.StatusCode, nil
}
