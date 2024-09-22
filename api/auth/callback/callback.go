package callback

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/go-github/v63/github"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/user"
	"golang.org/x/oauth2"
)

// HandleCallback is the route the user is redirected from after authorization.
// It exchanges authCode, clientId and clientSecret with Access- and Refreshtoken.
func HandleCallback(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {

	cookie, code, err := runHandleCallback(request, transportCtx, routeCtx)
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
		Headers: map[string]string{
			"Location":   routeCtx.FrontendRedirectURI,
			"Set-Cookie": cookie,
		},
	}, nil
}

func runHandleCallback(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (string, int, error) {
	authCode := request.QueryStringParameters["code"]

	token, err := routeCtx.OAuthConfig.Exchange(transportCtx, authCode)
	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf("failed to exchange authorization code")
	}

	oauthClient := oauth2.NewClient(transportCtx, oauth2.StaticTokenSource(token))
	githubClient := github.NewClient(oauthClient)

	githubUser, _, err := githubClient.Users.Get(transportCtx, "")
	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf("failed to acquire user information from github")
	}

	_, err = database.UpdateSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
		Table: routeCtx.UserTable,
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: strconv.Itoa(int(*githubUser.ID))},
		},
		AttributeNames: map[string]string{
			"#refresh_token": "refresh_token",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":refresh_token": &dynamodbtypes.AttributeValueMemberS{Value: token.RefreshToken},
		},
		UpdateExpr: "SET #refresh_token = :refresh_token",
	})
	if err != nil {
		// if the user is not registered, setting the refresh token is simply skipped (no error is emitted).
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); !ok {
			return "", http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
		}
	}

	userToken, err := auth.CreateJWT(routeCtx.JwtOptions, strconv.Itoa(int(*githubUser.ID)), "github", *githubUser.Name, *githubUser.AvatarURL)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to create user_token: %v", err)
	}

	userTokenCookie := &http.Cookie{
		Name:     "user_token",
		Value:    userToken,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Now().Add(routeCtx.JwtOptions.TTL),
	}

	accessTokenCookie := &http.Cookie{
		Name:     "access_token",
		Value:    token.AccessToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  token.Expiry,
	}

	return fmt.Sprintf("%s, %s", accessTokenCookie.String(), userTokenCookie.String()), http.StatusFound, nil
}
