package authorize

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
)

// HandleAuthorization redirects the user to the authorization endpoint of the Cognito provider.
// Based on https://docs.aws.amazon.com/cognito/latest/developerguide/authorization-endpoint.html
func HandleAuthorization(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	// Hardcoded default scopes are openid (to enable openid connect) and profile (to acquire basic user information).
	authScopes := "openid profile email"

	// Format is based on the cognito endpoint spec: https://docs.aws.amazon.com/cognito/latest/developerguide/authorization-endpoint.html
	authUrl := fmt.Sprintf(
		"%s/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=%s",
		routeCtx.CognitoDomain, routeCtx.ClientID, routeCtx.RedirectURI, authScopes,
	)

	return events.APIGatewayV2HTTPResponse{
		StatusCode: 302,
		Headers: map[string]string{
			"Location": authUrl,
		},
	}, nil
}
