package upsertsubscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
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
	AliasCount       int64 `bson:"alias_count"`
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

	userDoc, err := database.GetSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: routeCtx.UserTable,
		Index: "",
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: userToken.Id},
		},
		ConditionExpr: "id = :id",
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("user not found")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
	}

	if !rbac.CheckPermission(userDoc.Roles, rbac.WRITE_SUBSCRIPTION) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	// MIG: Possible with update item and primary key
	err = database.PutSingle(transportCtx, routeCtx.DynamoClient, &database.PutSingleInput[subscription.Subscription]{
		Table: routeCtx.SubscriptionTable,
		Item: subscription.Subscription{
			Id:   upsertSubscriptionInput.Id,
			Name: upsertSubscriptionInput.Name,
			PipelineSpecs: subscription.PipelineSpecs{
				DailyBuilds:      upsertSubscriptionInput.PipelineSpecs.DailyBuilds,
				DailyDeployments: upsertSubscriptionInput.PipelineSpecs.DailyDeployments,
			},
			ProjectSpecs: subscription.ProjectSpecs{
				ProjectCount:     upsertSubscriptionInput.ProjectSpecs.ProjectCount,
				AliasCount:       upsertSubscriptionInput.ProjectSpecs.AliasCount,
				ServerStorage:    upsertSubscriptionInput.ProjectSpecs.ServerStorage,
				ClientStorage:    upsertSubscriptionInput.ProjectSpecs.ClientStorage,
				PrerenderStorage: upsertSubscriptionInput.ProjectSpecs.PrerenderStorage,
				PrerenderRoutes:  upsertSubscriptionInput.ProjectSpecs.PrerenderRoutes,
			},
			CDNSpecs: subscription.CDNSpecs{
				InstanceCount: upsertSubscriptionInput.CDNSpecs.InstanceCount,
			},
		},
	})
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to update subscription: %v", err)
	}

	return &upsertSubscriptionOutput{
		Message: "subscription upserted",
	}, http.StatusOK, nil
}
