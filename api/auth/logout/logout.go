package logout

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
)

// HandleLogout logs the user out and revokes the used tokens.
func HandleLogout(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	cookie, code, err := runHandleLogout(request, transportCtx, routeCtx)
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

func runHandleLogout(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (string, int, error) {
	clearCookieHeader := fmt.Sprintf(
		"%s, %s",
		(&http.Cookie{Name: "access_token", Expires: time.Now().Add(-24 * time.Hour)}).String(),
		(&http.Cookie{Name: "refresh_token", Expires: time.Now().Add(-24 * time.Hour)}).String(),
	)

	// Parse cookie by creating a http.Request and reading the cookie from there.
	accessTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("access_token")
	if err != nil {
		return clearCookieHeader, http.StatusNoContent, nil
	}

	input := &cognitoidentityprovider.GlobalSignOutInput{
		AccessToken: aws.String(accessTokenCookie.Value),
	}

	_, err = routeCtx.CognitoClient.GlobalSignOut(transportCtx, input)
	if err != nil {
		return clearCookieHeader, http.StatusInternalServerError, fmt.Errorf("failed to sign out globally: %v", err)
	}

	return clearCookieHeader, http.StatusNoContent, nil
}
