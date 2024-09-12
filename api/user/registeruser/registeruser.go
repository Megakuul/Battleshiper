package registeruser

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/megakuul/battleshiper/api/user/routecontext"

	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/user"
)

// HandleRegisterUser registers a user in the database (if not existent).
func HandleRegisterUser(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	code, err := runHandleRegisterUser(request, transportCtx, routeCtx)
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

func runHandleRegisterUser(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (int, error) {
	userTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("user_token")
	if err != nil {
		return http.StatusUnauthorized, fmt.Errorf("no user_token provided")
	}

	userToken, err := auth.ParseJWT(routeCtx.JwtOptions, userTokenCookie.Value)
	if err != nil {
		return http.StatusUnauthorized, fmt.Errorf("user_token is invalid: %v", err)
	}

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	var userDoc user.User
	err = userCollection.FindOne(transportCtx, bson.M{"id": userToken.Id}).Decode(&userDoc)
	if err == mongo.ErrNoDocuments {
		newDoc := user.User{
			Id:             userToken.Id,
			Privileged:     false,
			Provider:       "github",
			Roles:          map[rbac.ROLE]struct{}{rbac.USER: {}},
			RefreshToken:   "",
			SubscriptionId: "",
			LimitCounter: user.ExecutionLimitCounter{
				PipelineBuildsExpiration:      0,
				PipelineBuilds:                0,
				PipelineDeploymentsExpiration: 0,
				PipelineDeployments:           0,
			},
			GithubData: user.GithubData{
				InstallationId: 0,
				Repositories:   []user.Repository{},
			},
		}
		_, err := userCollection.InsertOne(transportCtx, newDoc)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("failed to insert default user record to database")
		}
	} else if err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to read user records from database")
	}

	// Operation is idempotent; returns OK whether the document already existed or was freshly inserted.
	return http.StatusOK, nil
}
