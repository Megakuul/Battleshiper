package deleteproject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	eventtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
)

var logger = log.New(os.Stderr, "RESOURCE DELETEPROJECT: ", 0)

type deleteProjectInput struct {
	ProjectName string `json:"project_name"`
}

type deleteProjectOutput struct {
	Message string `json:"message"`
}

// HandleDeleteProject marks the specified project as deleted.
func HandleDeleteProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleDeleteProject(request, transportCtx, routeCtx)
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

func runHandleDeleteProject(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*deleteProjectOutput, int, error) {

	var deleteProjectInput deleteProjectInput
	err := json.Unmarshal([]byte(request.Body), &deleteProjectInput)
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

	projectDoc, err := database.UpdateSingle[project.Project](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(routeCtx.ProjectTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"project_name": &dynamodbtypes.AttributeValueMemberS{Value: deleteProjectInput.ProjectName},
		},
		AttributeNames: map[string]string{
			"#owner_id": "owner_id",
			"#deleted":  "deleted",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":owner_id": &dynamodbtypes.AttributeValueMemberS{Value: userToken.Id},
			":deleted":  &dynamodbtypes.AttributeValueMemberBOOL{Value: true},
		},
		ConditionExpr: aws.String("#owner_id = :owner_id"),
		UpdateExpr:    aws.String("SET #deleted = :deleted"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("project not found")
		}
		logger.Printf("failed to mark project as deleted on database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to mark project as deleted on database")
	}

	deleteTicket, err := pipeline.CreateTicket(routeCtx.DeleteEventOptions.TicketOpts, userToken.Id, projectDoc.ProjectName)
	if err != nil {
		logger.Printf("failed to create pipeline ticket: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create pipeline ticket")
	}
	deleteRequest := &event.DeleteRequest{
		DeleteTicket: deleteTicket,
	}
	deleteRequestRaw, err := json.Marshal(deleteRequest)
	if err != nil {
		logger.Printf("failed to serialize deletion request: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to serialize deletion request")
	}
	eventEntry := eventtypes.PutEventsRequestEntry{
		Source:       aws.String(routeCtx.DeleteEventOptions.Source),
		DetailType:   aws.String(routeCtx.DeleteEventOptions.Action),
		Detail:       aws.String(string(deleteRequestRaw)),
		EventBusName: aws.String(routeCtx.DeleteEventOptions.EventBus),
	}
	res, err := routeCtx.EventClient.PutEvents(transportCtx, &eventbridge.PutEventsInput{
		Entries: []eventtypes.PutEventsRequestEntry{eventEntry},
	})
	if err != nil {
		logger.Printf("failed to emit deletion event: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to emit deletion event")
	} else if res.FailedEntryCount > 0 && len(res.Entries) > 0 {
		logger.Printf("failed to ingest deletion event: %v\n", *res.Entries[0].ErrorMessage)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to ingest deletion event")
	}

	return &deleteProjectOutput{
		Message: "successfully marked project as deleted",
	}, http.StatusOK, nil
}
