package logout

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
)

var logger = log.New(os.Stderr, "AUTH LOGOUT: ", 0)

// HandleLogout logs the user out and revokes the used tokens.
func HandleLogout(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	cookies, code, err := runHandleLogout(request, transportCtx, routeCtx)
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
		Cookies:    cookies,
	}, nil
}

func runHandleLogout(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) ([]string, int, error) {
	clearCookies := []string{
		(&http.Cookie{
			Name:     "user_token",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/api",
			Expires:  time.Now().Add(-24 * time.Hour)}).String(),
		(&http.Cookie{
			Name:     "access_token",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Path:     "/api",
			Expires:  time.Now().Add(-24 * time.Hour)}).String(),
	}

	userTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("user_token")
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("no user_token provided")
	}

	userToken, err := auth.ParseJWT(routeCtx.JwtOptions, userTokenCookie.Value)
	if err != nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("user_token is invalid: %v", err)
	}

	_, err = database.UpdateSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(routeCtx.UserTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: userToken.Id},
		},
		AttributeNames: map[string]string{
			"#refresh_token": "refresh_token",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":refresh_token": &dynamodbtypes.AttributeValueMemberS{Value: ""},
		},
		UpdateExpr: aws.String("SET #refresh_token = :refresh_token"),
	})
	if err != nil {
		// if the user is not registered, deleting the refresh token is simply skipped (no error is emitted).
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); !ok {
			logger.Printf("failed to update user on database: %v\n", err)
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to update user on database")
		}
	}

	return clearCookies, http.StatusOK, nil
}
