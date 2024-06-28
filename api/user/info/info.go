package info

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/megakuul/battleshiper/api/user/router"
)

// HandleInfo
func HandleInfo(request events.APIGatewayV2HTTPRequest, transportCtx context.Context, routeCtx router.RouteContext) (events.APIGatewayV2HTTPResponse, error) {
	// Parse cookie by creating a http.Request and reading the cookie from there.
	accessTokenCookie, err := (&http.Request{Header: http.Header{"Cookie": request.Cookies}}).Cookie("access_token")
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusUnauthorized,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "User did not provide a valid access_token",
		}, nil
	}

	userRequest := &cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(accessTokenCookie.Value),
	}

	userResponse, err := routeCtx.CognitoClient.GetUser(transportCtx, userRequest)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: http.StatusUnauthorized,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: fmt.Sprintf("Failed to acquire user information: %v", err),
		}, nil
	}

	attributes := map[string]string{}
	for _, attr := range userResponse.UserAttributes {
		attributes[*attr.Name] = *attr.Value
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "text/plain",
		},
		Body: fmt.Sprintf("User sub: %s . User rest: %v", attributes["sub"], attributes),
	}, nil
}
