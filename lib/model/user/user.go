// Contains database types for the user collection.
package user

import "github.com/megakuul/battleshiper/lib/model/rbac"

const USER_COLLECTION = "users"

type ExecutionLimitCounter struct {
	ExpirationTime      int64 `bson:"expiration_time"`
	PipelineBuilds      int   `bson:"pipeline_builds"`
	PipelineDeployments int   `bson:"pipeline_deployments"`
}

type Repository struct {
	Id       int64  `bson:"id"`
	Name     string `bson:"name"`
	FullName string `bson:"full_name"`
}

type GithubData struct {
	InstallationId int64        `bson:"installation_id"`
	Repositories   []Repository `bson:"repositories"`
}

type User struct {
	MongoID        interface{}            `bson:"_id"`
	Id             string                 `bson:"id"`
	Privileged     bool                   `bson:"privileged"`
	Provider       string                 `bson:"provider"`
	Roles          map[rbac.ROLE]struct{} `bson:"roles"`
	RefreshToken   string                 `bson:"refresh_token"`
	LimitCounter   ExecutionLimitCounter  `bson:"limit_counter"`
	SubscriptionId string                 `bson:"subscription_id"`
	GithubData     GithubData             `bson:"github_data"`
}
