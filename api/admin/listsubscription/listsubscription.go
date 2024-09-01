package listsubscription

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
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
)

type subscriptionOutput struct {
	Id                       string `json:"id"`
	Name                     string `json:"name"`
	DailyPipelineBuilds      int    `json:"daily_pipeline_builds"`
	DailyPipelineDeployments int    `json:"daily_pipeline_deployments"`
	StaticCacheRoutes        int    `json:"static_cache_routes"`
	DedicatedCDNInstances    int    `json:"dedicated_cdn_instances"`
	Projects                 int    `json:"projects"`
}

type listSubscriptionOutput struct {
	Message       string               `json:"message"`
	Subscriptions []subscriptionOutput `json:"subscriptions"`
}

// HandleListSubscription performs a lookup for the specified subscriptions and returns them as json object.
func HandleListSubscription(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleListSubscription(request, transportCtx, routeCtx)
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

func runHandleListSubscription(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*listSubscriptionOutput, int, error) {
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

	if !rbac.CheckPermission(userDoc.Roles, rbac.READ_SUBSCRIPTION) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	subscriptionCollection := routeCtx.Database.Collection(subscription.SUBSCRIPTION_COLLECTION)

	cursor, err := subscriptionCollection.Find(transportCtx, bson.D{})
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch data from database")
	}

	foundSubscriptionDocs := []subscription.Subscription{}
	err = cursor.All(transportCtx, &foundSubscriptionDocs)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch and decode subscriptions")
	}

	foundSubscriptionOutput := []subscriptionOutput{}
	for _, sub := range foundSubscriptionOutput {
		foundSubscriptionOutput = append(foundSubscriptionOutput, subscriptionOutput{
			Id:                       sub.Id,
			Name:                     sub.Name,
			DailyPipelineBuilds:      sub.DailyPipelineBuilds,
			DailyPipelineDeployments: sub.DailyPipelineDeployments,
			StaticCacheRoutes:        sub.StaticCacheRoutes,
			DedicatedCDNInstances:    sub.DedicatedCDNInstances,
			Projects:                 sub.Projects,
		})
	}

	return &listSubscriptionOutput{
		Message:       "subscriptions fetched",
		Subscriptions: foundSubscriptionOutput,
	}, http.StatusOK, nil
}
