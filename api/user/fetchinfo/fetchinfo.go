package fetchinfo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"github.com/megakuul/battleshiper/api/user/routecontext"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
)

type pipelineSpecsOutput struct {
	DailyBuilds      int64 `json:"daily_builds"`
	DailyDeployments int64 `json:"daily_deployments"`
}

type projectSpecsOutput struct {
	ProjectCount     int64 `json:"project_count"`
	AliasCount       int64 `bson:"alias_count"`
	PrerenderRoutes  int64 `json:"prerender_routes"`
	ServerStorage    int64 `json:"server_storage"`
	ClientStorage    int64 `json:"client_storage"`
	PrerenderStorage int64 `json:"prerender_storage"`
}

type cdnSpecsOutput struct {
	InstanceCount int64 `json:"instance_count"`
}

type subscriptionOutput struct {
	Id            string              `json:"id"`
	Name          string              `json:"name"`
	PipelineSpecs pipelineSpecsOutput `json:"pipeline_specs"`
	ProjectSpecs  projectSpecsOutput  `json:"project_specs"`
	CDNSpecs      cdnSpecsOutput      `json:"cdn_specs"`
}

type fetchInfoOutput struct {
	Id        string                 `json:"id"`
	Name      string                 `json:"name"`
	Roles     map[rbac.ROLE]struct{} `json:"roles"`
	Provider  string                 `json:"provider"`
	AvatarURL string                 `json:"avatar_url"`

	Subscription *subscriptionOutput `json:"subscription"`
}

// HandleFetchInfo fetches user information from the database cluster.
func HandleFetchInfo(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	response, code, err := runHandleFetchInfo(request, transportCtx, routeCtx)
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

func runHandleFetchInfo(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (*fetchInfoOutput, int, error) {

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
		// TODO: Remove debug verbose error output
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load user record from database: %v", err)
	}

	if userDoc.SubscriptionId == "" {
		return &fetchInfoOutput{
			Id:           userToken.Id,
			Name:         userToken.Username,
			Roles:        userDoc.Roles,
			Provider:     userToken.Provider,
			AvatarURL:    userToken.AvatarURL,
			Subscription: nil,
		}, http.StatusOK, nil
	}

	subscriptionDoc, err := database.GetSingle[subscription.Subscription](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: routeCtx.SubscriptionTable,
		Index: "",
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: userDoc.SubscriptionId},
		},
		ConditionExpr: "id = :id",
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("subscription not found")
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load subscription record from database")
	}

	return &fetchInfoOutput{
		Id:        userToken.Id,
		Name:      userToken.Username,
		Roles:     userDoc.Roles,
		Provider:  userToken.Provider,
		AvatarURL: userToken.AvatarURL,
		Subscription: &subscriptionOutput{
			Id:   subscriptionDoc.Id,
			Name: subscriptionDoc.Name,
			PipelineSpecs: pipelineSpecsOutput{
				DailyBuilds:      subscriptionDoc.PipelineSpecs.DailyBuilds,
				DailyDeployments: subscriptionDoc.PipelineSpecs.DailyDeployments,
			},
			ProjectSpecs: projectSpecsOutput{
				ProjectCount:     subscriptionDoc.ProjectSpecs.ProjectCount,
				AliasCount:       subscriptionDoc.ProjectSpecs.AliasCount,
				ServerStorage:    subscriptionDoc.ProjectSpecs.ServerStorage,
				ClientStorage:    subscriptionDoc.ProjectSpecs.ClientStorage,
				PrerenderStorage: subscriptionDoc.ProjectSpecs.PrerenderStorage,
				PrerenderRoutes:  subscriptionDoc.ProjectSpecs.PrerenderRoutes,
			},
			CDNSpecs: cdnSpecsOutput{
				InstanceCount: subscriptionDoc.CDNSpecs.InstanceCount,
			},
		},
	}, http.StatusOK, nil
}
