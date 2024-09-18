package fetchinfo

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

	Subscription *subscriptionOutput `json:"subscriptions"`
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

	// MIG: Possible with query item and primary key
	userDoc := &user.User{}
	err = userCollection.FindOne(transportCtx, bson.M{"id": userToken.Id}).Decode(&userDoc)
	if err == mongo.ErrNoDocuments {
		return nil, http.StatusUnauthorized, fmt.Errorf("user does not exist")
	} else if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to read user record from database")
	}

	// MIG: Possible with query item and primary key
	subscriptionDoc := &subscription.Subscription{}
	err = subscriptionCollection.FindOne(transportCtx, bson.M{"id": userDoc.SubscriptionId}).Decode(subscriptionDoc)
	if err == mongo.ErrNoDocuments {
		return &fetchInfoOutput{
			Id:           userToken.Id,
			Name:         userToken.Username,
			Roles:        userDoc.Roles,
			Provider:     userToken.Provider,
			AvatarURL:    userToken.AvatarURL,
			Subscription: nil,
		}, http.StatusOK, nil
	} else if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to read user subscription from database")
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
