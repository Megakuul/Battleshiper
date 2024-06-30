package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

// FetchUserAttributes reads the "access_token" cookie from the request and tries to fetch the user attributes from the cognitoClient.
// Returns a map containing the fetched user attributes / claims.
// The attributes included are determined by the scopes requested on authorization: https://auth0.com/docs/get-started/apis/scopes/openid-connect-scopes
func FetchUserAttributes(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, cognitoClient *cognitoidentityprovider.Client) (map[string]string, error) {
	// Parse cookie by creating a http.Request and reading the cookie from there.
	accessTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("access_token")
	if err != nil {
		return nil, fmt.Errorf("user did not provide a valid access_token")
	}

	userRequest := &cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(accessTokenCookie.Value),
	}

	userResponse, err := cognitoClient.GetUser(transportCtx, userRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire user information: %v", err)
	}

	attributes := map[string]string{}
	for _, attr := range userResponse.UserAttributes {
		attributes[*attr.Name] = *attr.Value
	}

	return attributes, nil
}
