package deleteproject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	eventtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/user"
)

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

	userDoc, err := database.GetSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: routeCtx.UserTable,
		Index: "",
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: userToken.Id},
		},
		ConditionExpr: "id = :id",
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("user not found")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
	}

	if !rbac.CheckPermission(userDoc.Roles, rbac.WRITE_PROJECT) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	projectDoc, err := database.UpdateSingle[project.Project](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
		Table: routeCtx.ProjectTable,
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"name": &dynamodbtypes.AttributeValueMemberS{Value: deleteProjectInput.ProjectName},
		},
		AttributeNames: map[string]string{
			"#deleted": "deleted",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":deleted": &dynamodbtypes.AttributeValueMemberBOOL{Value: true},
		},
		UpdateExpr: "SET #deleted = :deleted",
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("project not found")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to mark project as deleted on database")
	}

	deleteTicket, err := pipeline.CreateTicket(routeCtx.DeleteEventOptions.TicketOpts, userToken.Id, projectDoc.Name)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create pipeline ticket")
	}
	deleteRequest := &event.DeleteRequest{
		DeleteTicket: deleteTicket,
	}
	deleteRequestRaw, err := json.Marshal(deleteRequest)
	if err != nil {
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
	if err != nil || res.FailedEntryCount > 0 {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to emit deletion event")
	}

	return &deleteProjectOutput{
		Message: "successfully marked project as deleted",
	}, http.StatusOK, nil
}
