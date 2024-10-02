package registeruser

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/megakuul/battleshiper/api/user/routecontext"

	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/rbac"
	"github.com/megakuul/battleshiper/lib/model/user"
)

var logger = log.New(os.Stderr, "USER REGISTERUSER: ", 0)

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
		InstallationId: 0,
		Repositories:   map[int64]user.Repository{},
	}

	if routeCtx.UserConfiguration.AdminUsername != "" {
		// Github usernames are case insensitive
		if strings.EqualFold(userToken.Username, routeCtx.UserConfiguration.AdminUsername) {
			newDoc.Privileged = true
			newDoc.Roles = map[rbac.ROLE]struct{}{rbac.USER: {}, rbac.ROLE_MANAGER: {}}
		}
	}

	err = database.PutSingle(transportCtx, routeCtx.DynamoClient, &database.PutSingleInput[user.User]{
		Table:                   aws.String(routeCtx.UserTable),
		Item:                    newDoc,
		ProtectionAttributeName: aws.String("id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return http.StatusOK, nil
		}
		logger.Printf("failed to add user to database: %v\n", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to add user to database")
	}

	return http.StatusOK, nil
}
