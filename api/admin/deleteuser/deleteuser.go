package deleteuser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/user"
)

type deleteUserInput struct {
	UserId string `json:"user_id"`
}

type deleteUserOutput struct {
	Message string `json:"message"`
}

// HandleDeleteUser marks all projects of a user as deleted and removes the user from the database.
func HandleDeleteUser(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleDeleteUser(request, transportCtx, routeCtx)
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

func runHandleDeleteUser(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*deleteUserOutput, int, error) {
	var deleteUserInput deleteUserInput
	err := json.Unmarshal([]byte(request.Body), &deleteUserInput)
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
		return nil, http.StatusBadRequest, fmt.Errorf("failed to load user record from database")
	}

	if !rbac.CheckPermission(userDoc.Roles, rbac.WRITE_USER) || !rbac.CheckPermission(userDoc.Roles, rbac.WRITE_PROJECT) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	deletionUserDoc, err := database.GetSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: routeCtx.UserTable,
		Index: "",
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: deleteUserInput.UserId},
		},
		ConditionExpr: "id = :id",
	})
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to load user record from database")
	}

	if deletionUserDoc.Privileged {
		return nil, http.StatusBadRequest, fmt.Errorf("user has elevated permissions and cannot be deleted. remove privileges first")
	}

	deletionUserProjectDocs, err := database.GetMany[project.Project](transportCtx, routeCtx.DynamoClient, &database.GetManyInput{
		Table: routeCtx.ProjectTable,
		Index: project.GSI_OWNER_ID,
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":owner_id": &dynamodbtypes.AttributeValueMemberS{Value: deletionUserDoc.Id},
		},
		ConditionExpr: "owner_id = :owner_id",
		Limit:         1,
	})
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed load projects on database")
	}

	if len(deletionUserProjectDocs) > 0 {
		return nil, http.StatusBadRequest, fmt.Errorf("user has at least one active project and cannot be deleted. delete the projects first")
	}

	err = database.DeleteSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.DeleteSingleInput{
		Table: routeCtx.UserTable,
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: deletionUserDoc.Id},
		},
	})
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to delete user from database")
	}

	return &deleteUserOutput{
		Message: "successfully removed user and marked associated projects as deleted",
	}, http.StatusOK, nil
}
