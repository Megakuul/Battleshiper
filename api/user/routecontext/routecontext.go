package routecontext

import (
	"github.com/megakuul/battleshiper/lib/helper/auth"
)

type UserConfiguration struct {
	AdminUsername string
}

// Context provides data to route handlers.
type Context struct {
	JwtOptions        *auth.JwtOptions
	UserConfiguration *UserConfiguration
}
