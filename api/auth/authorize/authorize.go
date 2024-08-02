package authorize

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/megakuul/battleshiper/api/auth/routecontext"
)

// HandleAuthorization redirects the user to the authorization endpoint of Github.
// https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps
func HandleAuthorization(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx routecontext.Context) (events.APIGatewayV2HTTPResponse, error) {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: 302,
		Headers: map[string]string{
			"Location": routeCtx.OAuthConfig.AuthCodeURL(""),
		},
	}, nil
}
