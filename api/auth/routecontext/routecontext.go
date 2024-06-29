package routecontext

import "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

// Context provides data to route handlers.
type Context struct {
	CognitoClient       *cognitoidentityprovider.Client
	CognitoDomain       string
	ClientID            string
	ClientSecret        string
	RedirectURI         string
	FrontendRedirectURI string
}
