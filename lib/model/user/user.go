// Contains database types for the user collection.
package user

const USER_COLLECTION = "users"

type Subscriptions struct {
	DailyPipelineExecutions int `bson:"daily_pipeline_executions"`
	DefaultDeployments      int `bson:"default_deployments"`
}

type User struct {
	Sub           string        `bson:"sub"`
	Subscriptions Subscriptions `bson:"subscriptions"`
	ProjectIds    []string      `bson:"project_ids"`
}
