package registeruser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/megakuul/battleshiper/api/user/routecontext"
)

// HandleRouteRequest routes request either to s3 or to the corresponding lambda http endpoint.
func HandleRouteRequest(request events.ALBTargetGroupRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.ALBTargetGroupResponse, error) {
	project, exists := request.Headers["Battleshiper-Project"]
	if !exists || project == "" {
		return events.ALBTargetGroupResponse{
			StatusCode: 404,
			Headers:    map[string]string{"Content-Type": "text/plain"},
			Body:       "Project not found",
		}, nil
	}

	if strings.HasSuffix(request.Path, ".html") && request.HTTPMethod == "GET" {
		response, code, err := proxyPrerendered(request, transportCtx, routeCtx)
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

	response, code, err := proxyServer(request, transportCtx, routeCtx)
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

func proxyPrerendered(request events.ALBTargetGroupRequest, transportCtx context.Context, routeCtx routecontext.Context) (*events.ALBTargetGroupResponse, int, error) {
	objectOutput, err := routeCtx.S3Client.GetObject(transportCtx, &s3.GetObjectInput{
		Bucket: aws.String(routeCtx.S3Bucket),
		Key:    aws.String(request.Path),
	})
	if err != nil {
		var nsk *s3types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, http.StatusNotFound, fmt.Errorf("prerendered asset not found")
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

func proxyServer(request events.ALBTargetGroupRequest, transportCtx context.Context, routeCtx routecontext.Context) (*events.ALBTargetGroupResponse, int, error) {
	query := url.Values{}
	for key, val := range request.QueryStringParameters {
		query.Set(key, val)
	}

	rawPath := fmt.Sprintf("https://%s.%s/%s", request.Path) // TODO find endpoint
	if len(query) > 0 {
		rawPath = fmt.Sprintf("%s?%s", rawPath, query.Encode())
	}

	httpRequest, err := http.NewRequestWithContext(
		transportCtx,
		request.HTTPMethod,
		rawPath,
		strings.NewReader(request.Body),
	)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	httpResponse, err := routeCtx.HttpClient.Do(httpRequest)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	headers := map[string][]string{}
	for key, val := range httpResponse.Header {
		headers[key] = val
	}

	return &events.ALBTargetGroupResponse{
		MultiValueHeaders: headers,
		Body:              string(body),
	}, http.StatusOK, nil
}
