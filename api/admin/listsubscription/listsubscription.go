package listsubscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/megakuul/battleshiper/api/admin/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
)

var logger = log.New(os.Stderr, "ADMIN LISTSUBSCRIPTIONS: ", 0)

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

	userDoc, err := database.GetSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(routeCtx.UserTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: userToken.Id},
		},
		ConditionExpr: aws.String("id = :id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return nil, http.StatusNotFound, fmt.Errorf("user not found")
		}
		logger.Printf("failed to load user record from database: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
	}

	if !rbac.CheckPermission(userDoc.Roles, rbac.READ_SUBSCRIPTION) {
		return nil, http.StatusForbidden, fmt.Errorf("user does not have sufficient permissions for this action")
	}

	foundSubscriptionDocs, err := database.ScanMany[subscription.Subscription](transportCtx, routeCtx.DynamoClient, &database.ScanManyInput{
		Table: aws.String(routeCtx.SubscriptionTable),
	})
	if err != nil {
		logger.Printf("failed to fetch subscriptions: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to fetch subscriptions")
	}

	foundSubscriptionOutput := []subscriptionOutput{}
	for _, sub := range foundSubscriptionDocs {
		foundSubscriptionOutput = append(foundSubscriptionOutput, subscriptionOutput{
			Id:   sub.Id,
			Name: sub.Name,
			PipelineSpecs: pipelineSpecsOutput{
				DailyBuilds:      sub.PipelineSpecs.DailyBuilds,
				DailyDeployments: sub.PipelineSpecs.DailyDeployments,
			},
			ProjectSpecs: projectSpecsOutput{
				ProjectCount:     sub.ProjectSpecs.ProjectCount,
				AliasCount:       sub.ProjectSpecs.AliasCount,
				ServerStorage:    sub.ProjectSpecs.ServerStorage,
				ClientStorage:    sub.ProjectSpecs.ClientStorage,
				PrerenderStorage: sub.ProjectSpecs.PrerenderStorage,
				PrerenderRoutes:  sub.ProjectSpecs.PrerenderRoutes,
			},
			CDNSpecs: cdnSpecsOutput{
				InstanceCount: sub.CDNSpecs.InstanceCount,
			},
		})
	}

	return &listSubscriptionOutput{
		Message:       "subscriptions fetched",
		Subscriptions: foundSubscriptionOutput,
	}, http.StatusOK, nil
}
