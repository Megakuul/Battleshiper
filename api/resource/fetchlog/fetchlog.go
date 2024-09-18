package fetchlog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/project"
)

const MAX_LOG_EVENTS = 50

type fetchLogInput struct {
	ProjectName string `json:"project_name"`
	LogType     string `json:"log_type"`
	StartTime   int64  `json:"start_time"`
	EndTime     int64  `json:"end_time"`
	Count       int32  `json:"count"`
}

type eventOutput struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

type fetchLogOutput struct {
	Message string        `json:"message"`
	Events  []eventOutput `json:"events"`
}

// HandleFetchLog performs a lookup for the cloudwatch logs of the associated project function.
func HandleFetchLog(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleFetchLog(request, transportCtx, routeCtx)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: code,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: err.Error(),
		}, nil
	}
	rawResponse, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "failed to serialize response",
		}, nil
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: code,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(rawResponse),
	}, nil
}

func runHandleFetchLog(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*fetchLogOutput, int, error) {
	fetchLogInput := &fetchLogInput{}
	err := json.Unmarshal([]byte(request.Body), &fetchLogInput)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to deserialize request: invalid body")
	}

	userTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("user_token")
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("no user_token provided")
	}

	userToken, err := auth.ParseJWT(routeCtx.JwtOptions, userTokenCookie.Value)
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("user_token is invalid: %v", err)
	}

	specifiedProject := &project.Project{}
	// MIG: Possible with query item and primary key + condition on owner_id and deleted
	err = projectCollection.FindOne(transportCtx,
		bson.D{
			{Key: "owner_id", Value: userToken.Id},
			{Key: "name", Value: fetchLogInput.ProjectName},
			{Key: "deleted", Value: false},
		},
	).Decode(&specifiedProject)
	if err == mongo.ErrNoDocuments {
		return nil, http.StatusNotFound, fmt.Errorf("project not found")
	} else if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch data from database")
	}

	var logGroup string
	switch fetchLogInput.LogType {
	case "server":
		logGroup = specifiedProject.DedicatedInfrastructure.ServerLogGroup
	case "event":
		logGroup = specifiedProject.DedicatedInfrastructure.EventLogGroup
	case "build":
		logGroup = specifiedProject.DedicatedInfrastructure.BuildLogGroup
	case "deploy":
		logGroup = specifiedProject.DedicatedInfrastructure.DeployLogGroup
	}
	if logGroup == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid logtype; expected 'server', 'event', 'build' or 'deploy'")
	}

	logLimit := fetchLogInput.Count
	if logLimit > MAX_LOG_EVENTS {
		logLimit = MAX_LOG_EVENTS
	}

	logFilterOutput, err := routeCtx.CloudwatchClient.FilterLogEvents(transportCtx, &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(logGroup),
		StartTime:    aws.Int64(fetchLogInput.StartTime),
		EndTime:      aws.Int64(fetchLogInput.EndTime),
		Limit:        aws.Int32(logLimit),
	})
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch logs from cloudwatch")
	}

	logEvents := []eventOutput{}
	for _, event := range logFilterOutput.Events {
		logEvents = append(logEvents, eventOutput{
			Timestamp: *event.Timestamp,
			Message:   *event.Message,
		})
	}

	return &fetchLogOutput{
		Message: "logs fetched",
		Events:  logEvents,
	}, http.StatusOK, nil
}
