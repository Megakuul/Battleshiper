package upsertsubscription

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
)

type pipelineSpecsInput struct {
	DailyBuilds      int64 `json:"daily_builds"`
	DailyDeployments int64 `json:"daily_deployments"`
}

type projectSpecsInput struct {
	ProjectCount     int64 `json:"project_count"`
	PrerenderRoutes  int64 `json:"prerender_routes"`
	ServerStorage    int64 `json:"server_storage"`
	ClientStorage    int64 `json:"client_storage"`
	PrerenderStorage int64 `json:"prerender_storage"`
}

type cdnSpecsInput struct {
	InstanceCount int64 `json:"instance_count"`
}

type upsertSubscriptionInput struct {
	Id            string             `json:"id"`
	Name          string             `json:"name"`
	PipelineSpecs pipelineSpecsInput `json:"pipeline_specs"`
	ProjectSpecs  projectSpecsInput  `json:"project_specs"`
	CDNSpecs      cdnSpecsInput      `json:"cdn_specs"`
}

type upsertSubscriptionOutput struct {
	Message string `json:"message"`
}

// HandleUpsertSubscription upserts a subscription identified by the subscription id.
func HandleUpsertSubscription(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleUpsertSubscription(request, transportCtx, routeCtx)
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

func runHandleUpsertSubscription(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*upsertSubscriptionOutput, int, error) {
	var upsertSubscriptionInput upsertSubscriptionInput
	err := json.Unmarshal([]byte(request.Body), &upsertSubscriptionInput)
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

	if !rbac.CheckPermission(userDoc.Roles, rbac.WRITE_SUBSCRIPTION) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	subscriptionCollection := routeCtx.Database.Collection(subscription.SUBSCRIPTION_COLLECTION)

	_, err = subscriptionCollection.UpdateOne(transportCtx, bson.M{"id": upsertSubscriptionInput.Id},
		bson.M{
			"$set": subscription.Subscription{
				Id:   upsertSubscriptionInput.Id,
				Name: upsertSubscriptionInput.Name,
				PipelineSpecs: subscription.PipelineSpecs{
					DailyBuilds:      upsertSubscriptionInput.PipelineSpecs.DailyBuilds,
					DailyDeployments: upsertSubscriptionInput.PipelineSpecs.DailyDeployments,
				},
				ProjectSpecs: subscription.ProjectSpecs{
					ProjectCount:     upsertSubscriptionInput.ProjectSpecs.ProjectCount,
					ServerStorage:    upsertSubscriptionInput.ProjectSpecs.ServerStorage,
					ClientStorage:    upsertSubscriptionInput.ProjectSpecs.ClientStorage,
					PrerenderStorage: upsertSubscriptionInput.ProjectSpecs.PrerenderStorage,
					PrerenderRoutes:  upsertSubscriptionInput.ProjectSpecs.PrerenderRoutes,
				},
				CDNSpecs: subscription.CDNSpecs{
					InstanceCount: upsertSubscriptionInput.CDNSpecs.InstanceCount,
				},
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch data from database")
	}

	return &upsertSubscriptionOutput{
		Message: "subscription upserted",
	}, http.StatusOK, nil
}
