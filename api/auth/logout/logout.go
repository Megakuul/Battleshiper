package logout

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/megakuul/battleshiper/api/auth/router"
)

// HandleLogout logs the user out and revokes the used tokens.
func HandleLogout(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx router.RouteContext) (events.APIGatewayV2HTTPResponse, error) {

	clearCookieHeader := fmt.Sprintf(
		"%s, %s",
		(&http.Cookie{Name: "access_token", Expires: time.Now().Add(-24 * time.Hour)}).String(),
		(&http.Cookie{Name: "refresh_token", Expires: time.Now().Add(-24 * time.Hour)}).String(),
	)

	// Parse cookie by creating a http.Request and reading the cookie from there.
	accessTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("access_token")
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusOK,
			Headers: map[string]string{
				"Content-Type": "text/plain",
				"Set-Cookie":   clearCookieHeader,
			},
			Body: "User is already logged out",
		}, nil
	}

	input := &cognitoidentityprovider.GlobalSignOutInput{
		AccessToken: aws.String(accessTokenCookie.Value),
	}

	_, err = routeCtx.CognitoClient.GlobalSignOut(transportCtx, input)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusInternalServerError,
			Headers: map[string]string{
				"Content-Type": "text/plain",
				"Set-Cookie":   clearCookieHeader,
			},
			Body: fmt.Sprintf("Failed to sign out globally: %v", err),
		}, nil
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "text/plain",
			"Set-Cookie":   clearCookieHeader,
		},
		Body: "Successfully logged out",
	}, nil
}
