package refresh

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/go-github/v63/github"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/oauth2"
)

type RefreshResponse struct {
	AccessToken string `json:"AccessToken"`
	Error       string `json:"Error"`
}

// HandleRefresh acquires a new access_token in tradeoff to the refresh_token.
func HandleRefresh(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	cookie, code, err := runHandleRefresh(request, transportCtx, routeCtx)
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
			"Set-Cookie": cookie,
		},
	}, nil
}

func refreshByUserToken(transportCtx context.Context, routeCtx routecontext.Context, userTokenRaw string) (string, int, error) {

	userToken, err := auth.ParseJWT(routeCtx.JwtOptions, userTokenRaw)
	if err != nil {
		return "", http.StatusUnauthorized, fmt.Errorf("user_token is invalid: %v", err)
	}

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	var userDoc user.User
	err = userCollection.FindOne(transportCtx, bson.M{"id": userToken.Id}).Decode(&userDoc)
	if err == mongo.ErrNoDocuments {
		return "", http.StatusUnauthorized, fmt.Errorf("user does not exist")
	} else if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to read user record from database")
	}

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		RefreshToken: userDoc.RefreshToken,
	})

	token, err := tokenSource.Token()
	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf("failed to acquire new token")
	}

	// The refresh token is intentionally not processed further;
	// after the refresh token expires, the user is forced to log in again.

	oauthClient := oauth2.NewClient(transportCtx, oauth2.StaticTokenSource(token))
	githubClient := github.NewClient(oauthClient)

	githubUser, _, err := githubClient.Users.Get(transportCtx, "")
	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf("failed to acquire user information from github")
	}

	newUserToken, err := auth.CreateJWT(routeCtx.JwtOptions, githubUser.ID, "github", githubUser.Name, githubUser.AvatarURL)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to create user_token: %v", err)
	}

	userTokenCookie := &http.Cookie{
		Name:     "user_token",
		Value:    newUserToken,
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

	return fmt.Sprintf("%s, %s", accessTokenCookie, userTokenCookie), http.StatusFound, nil
}

func refreshByAccessToken(transportCtx context.Context, routeCtx routecontext.Context, accessTokenRaw string) (string, int, error) {
	oauthClient := oauth2.NewClient(transportCtx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: accessTokenRaw,
	}))
	githubClient := github.NewClient(oauthClient)

	githubUser, _, err := githubClient.Users.Get(transportCtx, "")
	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf("failed to acquire user information from github")
	}

	userToken, err := auth.CreateJWT(routeCtx.JwtOptions, githubUser.ID, "github", githubUser.Name, githubUser.AvatarURL)
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

	return fmt.Sprintf("%s", userTokenCookie), http.StatusOK, nil
}

func runHandleRefresh(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (string, int, error) {

	userTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("user_token")
	if err == nil {
		return refreshByUserToken(transportCtx, routeCtx, userTokenCookie.Value)
	}

	accessTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("access_token")
	if err == nil {
		return refreshByAccessToken(transportCtx, routeCtx, accessTokenCookie.Value)
	}

	return "", http.StatusUnauthorized, fmt.Errorf("no valid user_token or access_token was present")
}
