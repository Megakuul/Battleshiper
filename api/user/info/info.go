package info

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/api/user/routecontext"

	"github.com/megakuul/battleshiper/lib/model/user"
)

type subscriptions struct {
	DailyPipelineExecutions int `json:"daily_pipeline_executions"`
	DefaultDeployments      int `json:"default_deployments"`
}

type infoResponse struct {
	Name       string `json:"name"`
	Nickname   string `json:"nickname"`
	Email      string `json:"email"`
	PictureURL string `json:"picture_url"`

	Subscriptions subscriptions `json:"subscriptions"`
}

// HandleInfo fetches user information from the database cluster.
func HandleInfo(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
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

func fetchUserAttributes(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (map[string]string, error) {
	// Parse cookie by creating a http.Request and reading the cookie from there.
	accessTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("access_token")
	if err != nil {
		return nil, fmt.Errorf("user did not provide a valid access_token")
	}

	userRequest := &cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(accessTokenCookie.Value),
	}

	userResponse, err := routeCtx.CognitoClient.GetUser(transportCtx, userRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire user information: %v", err)
	}

	attributes := map[string]string{}
	for _, attr := range userResponse.UserAttributes {
		attributes[*attr.Name] = *attr.Value
	}

	return attributes, nil
}

func runHandleInfo(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*infoResponse, int, error) {

	userAttributes, err := fetchUserAttributes(request, transportCtx, routeCtx)
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("failed to acquire user information: %v", err)
	}
	subAttr := attributes["sub"]

	userCollection := routeCtx.Database.Collection("users")

	if subAttr == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("openid connect user attribute 'sub' was not provided by the auth provider")
	}

	var userDoc user.User
	err = userCollection.FindOne(transportCtx, bson.M{"sub": subAttr}).Decode(&userDoc)
	if err == mongo.ErrNoDocuments {
		newDoc := user.User{
			Sub: subAttr,
			Subscriptions: &user.Subscriptions{
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
		Name:       attributes["name"],
		Nickname:   attributes["nickname"],
		Email:      attributes["email"],
		PictureURL: attributes["picture"],
		Subscriptions: &subscriptions{
			DailyPipelineExecutions: userDoc.Subscriptions.DailyPipelineExecutions,
			DefaultDeployments:      userDoc.Subscriptions.DefaultDeployments,
		},
	}, http.StatusOK, nil
}
