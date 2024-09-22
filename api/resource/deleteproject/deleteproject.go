package deleteproject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/api/resource/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/project"
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

	_, err = database.UpdateSingle[project.Project](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
		Table:      routeCtx.ProjectTable,
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{},
		AttributeNames: map[string]string{
			"#name":     "name",
			"#owner_id": "owner_id",
			"#deleted":  "deleted",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":name":     &dynamodbtypes.AttributeValueMemberS{Value: deleteProjectInput.ProjectName},
			":owner_id": &dynamodbtypes.AttributeValueMemberS{Value: userToken.Id},
			":deleted":  &dynamodbtypes.AttributeValueMemberBOOL{Value: true},
		},
		ConditionExpr: "#owner_id = :owner_id",
		UpdateExpr:    "SET #deleted = :deleted",
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("project not found")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to mark project as deleted on database")
	}

	return &deleteProjectOutput{
		Message: "successfully marked project as deleted",
	}, http.StatusOK, nil
}
