package deleteuser

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

type findUserInput struct {
	UserId         string `json:"user_id"`
	SubscriptionId string `json:"subscription_id"`
}

type userOutput struct {
	Id             string                 `json:"id"`
	Privileged     bool                   `json:"privileged"`
	Provider       string                 `json:"provider"`
	Roles          map[rbac.ROLE]struct{} `json:"roles"`
	SubscriptionId string                 `json:"subscription_id"`
	ProjectIds     []string               `json:"project_ids"`
}

type findUserOutput struct {
	Message string       `json:"message"`
	Users   []userOutput `json:"users"`
}

// HandleFindUser performs a lookup for the specified project and returns it as json object.
func HandleFindUser(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleFindUser(request, transportCtx, routeCtx)
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

func runHandleFindUser(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*findUserOutput, int, error) {
	var findUserInput findUserInput
	err := json.Unmarshal([]byte(request.Body), &findUserInput)
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

	if !rbac.CheckPermission(userDoc.Roles, rbac.READ_USER) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	cursor, err := userCollection.Find(transportCtx,
		bson.M{"$or": bson.A{
			bson.M{"id": findUserInput.UserId},
			bson.M{"subscription_id": findUserInput.SubscriptionId},
		}},
	)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch data from database")
	}

	foundUserDocs := []user.User{}
	err = cursor.All(transportCtx, &foundUserDocs)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch and decode users")
	}

	foundUserOutput := []userOutput{}
	for _, user := range foundUserDocs {
		foundUserOutput = append(foundUserOutput, userOutput{
			Id:             user.Id,
			Privileged:     user.Privileged,
			Provider:       user.Provider,
			Roles:          user.Roles,
			SubscriptionId: user.SubscriptionId,
			ProjectIds:     user.ProjectIds,
		})
	}

	return &findUserOutput{
		Message: "users fetched",
		Users:   foundUserOutput,
	}, http.StatusOK, nil
}
