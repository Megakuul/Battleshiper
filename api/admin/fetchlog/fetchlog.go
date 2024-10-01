package fetchlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/user"
)

const (
	MAX_LOG_EVENTS             = 200
	LOG_RETRIEVE_RETRY_COUNT   = 3
	LOG_RETRIEVE_RETRY_TIMEOUT = time.Millisecond * 200
)

var logger = log.New(os.Stderr, "ADMIN FETCHLOG: ", 0)

type fetchLogInput struct {
	LogType      string `json:"log_type"`
	StartTime    int64  `json:"start_time"`
	EndTime      int64  `json:"end_time"`
	Count        int32  `json:"count"`
	FilterLambda bool   `json:"filter_lambda"`
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

	userDoc, err := database.GetSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(routeCtx.UserTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: userToken.Id},
		},
		ConditionExpr: aws.String("id = :id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("user not found")
		}
		logger.Printf("failed to load user record from database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
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
	case "router":
		logGroup = routeCtx.LogConfiguration.RouterLogGroup
	}
	if logGroup == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("invalid logtype; expected 'api', 'pipeline' or 'router'")
	}

	logLimit := fetchLogInput.Count
	if logLimit > MAX_LOG_EVENTS {
		logLimit = MAX_LOG_EVENTS
	}

	lambdaFilter := ""
	if fetchLogInput.FilterLambda {
		// filters out the lambda generated START, END, REPORT and INIT_START messages
		lambdaFilter = "| filter @message not like /^(?:START|END|REPORT|INIT_START)/"
	}

	queryRequestOutput, err := routeCtx.CloudwatchClient.StartQuery(transportCtx, &cloudwatchlogs.StartQueryInput{
		LogGroupName: aws.String(logGroup),
		StartTime:    aws.Int64(fetchLogInput.StartTime),
		EndTime:      aws.Int64(fetchLogInput.EndTime),
		Limit:        aws.Int32(logLimit),
		QueryString: aws.String(fmt.Sprintf(
			"fields @timestamp, @message %s | sort @timestamp desc | limit %d",
			lambdaFilter,
			logLimit,
		)),
	})
	if err != nil {
		logger.Printf("failed to start cloudwatch query: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to start cloudwatch query")
	}

	for retries := 0; retries < LOG_RETRIEVE_RETRY_COUNT; retries++ {
		queryResultOutput, err := routeCtx.CloudwatchClient.GetQueryResults(transportCtx, &cloudwatchlogs.GetQueryResultsInput{
			QueryId: queryRequestOutput.QueryId,
		})
		if err != nil {
			logger.Printf("failed to retrieve cloudwatch query result: %v\n", err)
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to retrieve cloudwatch query result")
		}
		if queryResultOutput.Status == cloudwatchtypes.QueryStatusComplete {
			logEvents, err := extractLogEvents(queryResultOutput.Results)
			if err != nil {
				logger.Printf("failed to deserialize cloudwatch query result: %v\n", err)
				return nil, http.StatusInternalServerError, fmt.Errorf("failed to deserialize cloudwatch query result")
			}
			return &fetchLogOutput{
				Message: "logs fetched",
				Events:  logEvents,
			}, http.StatusOK, nil
		}
		time.Sleep(LOG_RETRIEVE_RETRY_TIMEOUT)
	}
	return nil, http.StatusBadRequest, fmt.Errorf("cloudwatch query timed out: try reducing the log timeframe")
}

// extractLogEvents converts the aws crap result field interface into an eventOutput slice
// it also converts the ISO 8601 timestamp of the logs (who came up with that idea... wtf???) into a unix timestamp (ms).
// what the fuck am I even doing here... this is called enterprise software, I kipp from se stuhl.
func extractLogEvents(results [][]cloudwatchtypes.ResultField) ([]eventOutput, error) {
	logEvents := []eventOutput{}
	for _, event := range results {
		logEvent := eventOutput{}
		for _, field := range event {
			switch *field.Field {
			case "@timestamp":
				// timestamp uses ISO 8601 format (RFC3339)
				fieldTimestamp, err := time.Parse(time.RFC3339, *field.Value)
				if err != nil {
					return nil, err
				}
				logEvent.Timestamp = fieldTimestamp.UnixMilli()
			case "@message":
				logEvent.Message = *field.Value
			}
		}
		logEvents = append(logEvents, logEvent)
	}
	return logEvents, nil
}
