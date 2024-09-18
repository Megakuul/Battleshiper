package routecontext

import (
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"golang.org/x/oauth2"
)

// Context provides data to route handlers.
type Context struct {
	JwtOptions          *auth.JwtOptions
	OAuthConfig         *oauth2.Config
	FrontendRedirectURI string
}
