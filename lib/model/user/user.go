// Contains database types for the user collection.
package user

import "github.com/megakuul/battleshiper/lib/model/rbac"

const USER_COLLECTION = "users"

type Subscriptions struct {
	DailyPipelineExecutions int `bson:"daily_pipeline_executions"`
	Deployments             int `bson:"deployments"`
}

type User struct {
	Id             string                 `bson:"id"`
	Provider       string                 `bson:"provider"`
	Roles          map[rbac.ROLE]struct{} `bson:"roles"`
	RefreshToken   string                 `bson:"refresh_token"`
	SubscriptionId string                 `bson:"subscription_id"`
	ProjectIds     []string               `bson:"project_ids"`
}
