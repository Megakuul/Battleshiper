package updateuser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/user"
)

type updateInput struct {
	SubscriptionId string `json:"subscription_id"`
}

type updateUserInput struct {
	UserId  string      `json:"user_id"`
	Updates updateInput `json:"updates"`
}

type updateUserOutput struct {
	Message string `json:"message"`
}

// HandleUpdateUser updates specified fields on a user identified by id.
func HandleUpdateUser(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleUpdateUser(request, transportCtx, routeCtx)
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

func runHandleUpdateUser(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*updateUserOutput, int, error) {
	var updateUserInput updateUserInput
	err := json.Unmarshal([]byte(request.Body), &updateUserInput)
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

	if !rbac.CheckPermission(userDoc.Roles, rbac.WRITE_USER) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	updateSpec := bson.M{}
	if updateUserInput.Updates.SubscriptionId != "" {
		updateSpec["subscription_id"] = updateUserInput.Updates.SubscriptionId
	}

	// MIG: Possible with update item and primary key
	result, err := userCollection.UpdateOne(transportCtx, bson.M{"id": updateUserInput.UserId}, bson.M{
		"$set": updateSpec,
	})
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch data from database")
	}
	if result.MatchedCount < 1 {
		return nil, http.StatusNotFound, fmt.Errorf("user not found")
	}

	return &updateUserOutput{
		Message: "users updated",
	}, http.StatusOK, nil
}
