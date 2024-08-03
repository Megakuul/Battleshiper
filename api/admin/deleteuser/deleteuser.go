package finduser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
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

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	userDoc := &user.User{}
	err = userCollection.FindOne(transportCtx, bson.M{"id": userToken.Id}).Decode(&userDoc)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to load user record from database")
	}

	if !rbac.CheckPermission(userDoc.Roles, rbac.WRITE_USER) || !rbac.CheckPermission(userDoc.Roles, rbac.WRITE_PROJECT) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	deletionUserDoc := &user.User{}
	err = userCollection.FindOne(transportCtx, bson.M{"id": deleteUserInput.UserId}).Decode(&deletionUserDoc)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to load user record from database")
	}

	if deletionUserDoc.Privileged {
		return nil, http.StatusBadRequest, fmt.Errorf("user has elevated permissions and cannot be deleted. remove privileges first")
	}

	projectCollection := routeCtx.Database.Collection(project.PROJECT_COLLECTION)

	for _, id := range deletionUserDoc.ProjectIds {
		_, err = projectCollection.UpdateOne(transportCtx, bson.M{"id": id}, bson.M{
			"$set": bson.M{
				"deleted": true,
			},
		})
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("failed to mark project as deleted on database")
		}
	}

	_, err = userCollection.DeleteOne(transportCtx, bson.M{"id": deleteUserInput.UserId})
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to delete user from database")
	}

	return &deleteUserOutput{
		Message: "successfully removed user and marked associated projects as deleted",
	}, http.StatusOK, nil
}
