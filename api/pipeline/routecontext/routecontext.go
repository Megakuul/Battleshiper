package routecontext

import (
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/oauth2"
)

// Context provides data to route handlers.
type Context struct {
	Database            *mongo.Database
	JwtOptions          *auth.JwtOptions
	OAuthConfig         *oauth2.Config
	FrontendRedirectURI string
}
