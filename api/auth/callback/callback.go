package callback

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/go-github/v63/github"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"github.com/megakuul/battleshiper/lib/model/user"
	"go.mongodb.org/mongo-driver/bson"
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

	userCollection := routeCtx.Database.Collection(user.USER_COLLECTION)

	// MIG: Possible with update item
	_, err = userCollection.UpdateOne(transportCtx, bson.M{"id": githubUser.ID}, bson.M{
		"$set": bson.M{
			"refresh_token": token.RefreshToken,
		},
	})
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to update user refresh_token")
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

	return fmt.Sprintf("%s, %s", accessTokenCookie, userTokenCookie), http.StatusFound, nil
}
