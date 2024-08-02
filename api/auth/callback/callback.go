package callback

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/go-github/v63/github"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
	"github.com/megakuul/battleshiper/lib/model/user"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/oauth2"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // Expiration time of the access token
}

// HandleCallback is the route the user is redirected from after authorization.
// It exchanges authCode, clientId and clientSecret with Access-, ID- and Refreshtoken.
// Follows the horrible OAuth2.0 standard which Cognito complies with:
// AccessToken request spec: https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.3
// with ClientSecret: https://datatracker.ietf.org/doc/html/rfc6749#section-2.3.1
// Authorization redirect spec: https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2
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

	_, err = userCollection.UpdateOne(transportCtx, bson.M{"id": githubUser.ID}, bson.M{
		"$set": bson.M{
			"refresh_token": token.RefreshToken,
		},
	}, options.Update().SetUpsert(false))
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("failed to update user refresh_token")
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
