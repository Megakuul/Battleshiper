package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"go.mongodb.org/mongo-driver/mongo"
)

// Context provides data to route handlers.
type Context struct {
	Database      *mongo.Database
	CognitoClient *cognitoidentityprovider.Client
	CognitoDomain string
	ClientID      string
	ClientSecret  string
}
