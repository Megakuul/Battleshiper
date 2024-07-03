package register

import (
	"context"
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
	code, err := runHandleRegister(request, transportCtx, routeCtx)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: code,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: err.Error(),
		}, nil
	}
	return events.APIGatewayV2HTTPResponse{
		StatusCode: code,
	}, nil
}

func runHandleRegister(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (int, error) {
	userAttributes, err := auth.FetchUserAttributes(request, transportCtx, routeCtx.CognitoClient)
	if err != nil {
		return http.StatusUnauthorized, fmt.Errorf("failed to acquire user information: %v", err)
	}

	subAttr := userAttributes["sub"]
	if subAttr == "" {
		return http.StatusBadRequest, fmt.Errorf("openid connect user attribute 'sub' was not provided by the auth provider")
	}

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	var userDoc user.User
	err = userCollection.FindOne(transportCtx, bson.M{"sub": subAttr}).Decode(&userDoc)
	if err == mongo.ErrNoDocuments {
		newDoc := user.User{
			Sub: subAttr,
			Subscriptions: user.Subscriptions{
				DailyPipelineExecutions: 0,
				DefaultDeployments:      0,
			},
			ProjectIds: []string{},
		}
		_, err := userCollection.InsertOne(transportCtx, newDoc)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("failed to insert default user record to database")
		}
	} else if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to read user records from database")
	}

	// Operation is idempotent; returns OK whether the document already existed or was freshly inserted.
	return http.StatusNoContent, nil
}
