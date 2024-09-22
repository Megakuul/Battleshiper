package routecontext

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/megakuul/battleshiper/lib/helper/auth"
)

type UserConfiguration struct {
	AdminUsername string
}

// Context provides data to route handlers.
type Context struct {
	DynamoClient      *dynamodb.Client
	UserTable         string
	SubscriptionTable string
	JwtOptions        *auth.JwtOptions
	UserConfiguration *UserConfiguration
}
