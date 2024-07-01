package register

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/api/user/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/user"
)

// HandleRegister registers a user in the database (if not existent) based on the cognito user attributes.
func HandleRegister(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleInfo(request, transportCtx, routeCtx)
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

func runHandleRegister(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (int, error) {
	userAttributes, err := auth.FetchUserAttributes(request, transportCtx, routeCtx.CognitoClient)
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("failed to acquire user information: %v", err)
	}

	subAttr := userAttributes["sub"]
	if subAttr == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("openid connect user attribute 'sub' was not provided by the auth provider")
	}

	userCollection := routeCtx.Database.Collection("users")

	var userDoc user.User
	err = userCollection.FindOne(transportCtx, bson.M{"sub": subAttr}).Decode(&userDoc)
	if err == mongo.ErrNoDocuments {
		newDoc := user.User{
			Sub: subAttr,
			Subscriptions: user.Subscriptions{
				DailyPipelineExecutions: 0,
				DefaultDeployments:      0,
			},
		}
		_, err := userCollection.InsertOne(transportCtx, newDoc)
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("user record was not found and insertion of default record failed")
		}
	} else if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to read user record from database")
	}

	return &infoResponse{
		Name:       userAttributes["name"],
		Nickname:   userAttributes["nickname"],
		Email:      userAttributes["email"],
		PictureURL: userAttributes["picture"],
		Subscriptions: subscriptions{
			DailyPipelineExecutions: userDoc.Subscriptions.DailyPipelineExecutions,
			DefaultDeployments:      userDoc.Subscriptions.DefaultDeployments,
		},
	}, http.StatusOK, nil
}
