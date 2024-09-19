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

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/user"
)

const MAX_LOG_EVENTS = 200

type fetchLogInput struct {
	LogType   string `json:"log_type"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Count     int32  `json:"count"`
}

type eventOutput struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

type fetchLogOutput struct {
	Message string        `json:"message"`
	Events  []eventOutput `json:"events"`
}

// HandleFetchLog performs a lookup for the cloudwatch logs of the internal functions and returns them.
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

	// MIG: Possible with query item and primary key
	userDoc := &user.User{}
	err = userCollection.FindOne(transportCtx, bson.M{"id": userToken.Id}).Decode(&userDoc)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to load user record from database")
	}

	if !rbac.CheckPermission(userDoc.Roles, rbac.READ_LOGS) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	var logGroup string
	switch fetchLogInput.LogType {
	case "api":
		logGroup = routeCtx.LogConfiguration.ApiLogGroup
	case "pipeline":
		logGroup = routeCtx.LogConfiguration.PipelineLogGroup
	}
	if logGroup == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid logtype; expected 'api' or 'pipeline'")
	}

	logLimit := fetchLogInput.Count
	if logLimit > MAX_LOG_EVENTS {
		logLimit = MAX_LOG_EVENTS
	}

	filterLogOutput, err := routeCtx.CloudwatchClient.FilterLogEvents(transportCtx, &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(logGroup),
		StartTime:    aws.Int64(fetchLogInput.StartTime),
		EndTime:      aws.Int64(fetchLogInput.EndTime),
		Limit:        aws.Int32(logLimit),
	})
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch log data from cloudwatch")
	}

	logEvents := []eventOutput{}
	for _, event := range filterLogOutput.Events {
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
