package refresh

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/megakuul/battleshiper/api/auth/routecontext"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
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

func runHandleRefresh(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (string, int, error) {

	// Parse cookie by creating a http.Request and reading the cookie from there.
	oldRefreshTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("refresh_token")
	if err != nil {
		return "", http.StatusUnauthorized, fmt.Errorf("user is not logged in")
	}

	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeRefreshTokenAuth,
		AuthParameters: map[string]string{
			"REFRESH_TOKEN": oldRefreshTokenCookie.Value,
		},
		ClientId: aws.String(routeCtx.ClientID),
	}

	res, err := routeCtx.CognitoClient.InitiateAuth(transportCtx, input)
	if err != nil {
		return "", http.StatusUnauthorized, fmt.Errorf("failed to acquire refresh token: %s", err.Error())
	}

	accessTokenCookie := &http.Cookie{
		Name:     "access_token",
		Value:    *res.AuthenticationResult.AccessToken,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  time.Now().Add(time.Duration(res.AuthenticationResult.ExpiresIn) * time.Second),
	}

	refreshTokenCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    *res.AuthenticationResult.RefreshToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}

	return fmt.Sprintf("%s, %s", accessTokenCookie, refreshTokenCookie), http.StatusOK, nil
}
