package routecontext

import (
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserConfiguration struct {
	AdminUsername string
}

// Context provides data to route handlers.
type Context struct {
	JwtOptions        *auth.JwtOptions
	Database          *mongo.Database
	UserConfiguration *UserConfiguration
}
