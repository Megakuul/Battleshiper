package refresh

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/go-github/v63/github"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/user"
	"golang.org/x/oauth2"
)

var logger = log.New(os.Stderr, "AUTH REFRESH: ", 0)

// HandleRefresh acquires a new access_token in tradeoff to the refresh_token.
func HandleRefresh(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	cookies, code, err := runHandleRefresh(request, transportCtx, routeCtx)
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

func runHandleRefresh(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) ([]string, int, error) {

	userTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("user_token")
	if err == nil {
		return refreshByUserToken(transportCtx, routeCtx, userTokenCookie.Value)
	}

	accessTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("access_token")
	if err == nil {
		return refreshByAccessToken(transportCtx, routeCtx, accessTokenCookie.Value)
	}

	return nil, http.StatusUnauthorized, fmt.Errorf("no valid user_token or access_token was present")
}

func refreshByUserToken(transportCtx context.Context, routeCtx routecontext.Context, userTokenRaw string) ([]string, int, error) {
	userToken, err := auth.ParseJWT(routeCtx.JwtOptions, userTokenRaw)
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

	tokenSource := routeCtx.OAuthConfig.TokenSource(transportCtx, &oauth2.Token{
		RefreshToken: userDoc.RefreshToken,
	})

	token, err := tokenSource.Token()
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to acquire new token")
	}

	// The refresh token is intentionally not processed further;
	// after the refresh token expires, the user is forced to log in again.

	oauthClient := oauth2.NewClient(transportCtx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token.AccessToken,
	}))
	githubClient := github.NewClient(oauthClient)

	githubUser, _, err := githubClient.Users.Get(transportCtx, "")
	if err != nil {
		logger.Printf("failed to acquire user information from github: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to acquire user information from github")
	}

	newUserToken, err := auth.CreateJWT(routeCtx.JwtOptions, strconv.Itoa(int(*githubUser.ID)), "github", *githubUser.Name, *githubUser.AvatarURL)
	if err != nil {
		logger.Printf("failed to create user_token: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create user_token")
	}

	userTokenCookie := &http.Cookie{
		Name:     "user_token",
		Value:    newUserToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/api",
		Expires:  time.Now().Add(routeCtx.JwtOptions.TTL),
	}

	accessTokenCookie := &http.Cookie{
		Name:     "access_token",
		Value:    token.AccessToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/api",
		Expires:  token.Expiry,
	}

	return []string{userTokenCookie.String(), accessTokenCookie.String()}, http.StatusOK, nil
}

func refreshByAccessToken(transportCtx context.Context, routeCtx routecontext.Context, accessTokenRaw string) ([]string, int, error) {
	oauthClient := oauth2.NewClient(transportCtx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessTokenRaw,
	}))
	githubClient := github.NewClient(oauthClient)

	githubUser, _, err := githubClient.Users.Get(transportCtx, "")
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to acquire user information from github")
	}

	userToken, err := auth.CreateJWT(routeCtx.JwtOptions, strconv.Itoa(int(*githubUser.ID)), "github", *githubUser.Name, *githubUser.AvatarURL)
	if err != nil {
		logger.Printf("failed to create user_token: %v\n", err)
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create user_token")
	}

	userTokenCookie := &http.Cookie{
		Name:     "user_token",
		Value:    userToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/api",
		Expires:  time.Now().Add(routeCtx.JwtOptions.TTL),
	}

	return []string{userTokenCookie.String()}, http.StatusOK, nil
}
