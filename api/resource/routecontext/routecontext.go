package routecontext

import (
	"github.com/google/go-github/v63/github"
	"github.com/megakuul/battleshiper/lib/helper/auth"
	"go.mongodb.org/mongo-driver/mongo"
)

// Context provides data to route handlers.
type Context struct {
	JwtOptions      *auth.JwtOptions
	GithubAppClient *github.Client
	Database        *mongo.Database
}
