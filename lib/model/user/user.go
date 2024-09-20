// Contains database types for the user collection.
package user

import "github.com/megakuul/battleshiper/lib/model/rbac"

type ExecutionLimitCounter struct {
	PipelineBuilds                int64 `dynamodbav:"pipeline_builds"`
	PipelineBuildsExpiration      int64 `dynamodbav:"pipeline_builds_exp"`
	PipelineDeployments           int64 `dynamodbav:"pipeline_deployments"`
	PipelineDeploymentsExpiration int64 `dynamodbav:"pipeline_deployments_exp"`
}

type Repository struct {
	Id       int64  `dynamodbav:"id"`
	Name     string `dynamodbav:"name"`
	FullName string `dynamodbav:"full_name"`
}

type User struct {
	MongoID        interface{}            `dynamodbav:"_id"`
	Id             string                 `dynamodbav:"id"`
	Privileged     bool                   `dynamodbav:"privileged"`
	Provider       string                 `dynamodbav:"provider"`
	Roles          map[rbac.ROLE]struct{} `dynamodbav:"roles"`
	RefreshToken   string                 `dynamodbav:"refresh_token"`
	LimitCounter   ExecutionLimitCounter  `dynamodbav:"limit_counter"`
	SubscriptionId string                 `dynamodbav:"subscription_id"`
	InstallationId int64                  `dynamodbav:"installation_id"`
	Repositories   []Repository           `dynamodbav:"repositories"`
}
