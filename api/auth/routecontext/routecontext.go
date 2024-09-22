package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"golang.org/x/oauth2"
)

// Context provides data to route handlers.
type Context struct {
	DynamoClient        *dynamodb.Client
	UserTable           string
	ProjectTable        string
	JwtOptions          *auth.JwtOptions
	OAuthConfig         *oauth2.Config
	FrontendRedirectURI string
}
