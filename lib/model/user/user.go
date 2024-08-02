// Contains database types for the user collection.
package user

const USER_COLLECTION = "users"

type Subscriptions struct {
	DailyPipelineExecutions int `bson:"daily_pipeline_executions"`
	Deployments             int `bson:"deployments"`
}

type User struct {
	ID              string   `bson:"id"`
	Provider        string   `bson:"provider"`
	Role            string   `bson:"role"`
	RefreshToken    string   `bson:"refresh_token"`
	SubscriptionIds []string `bson:"subscription_ids"`
	ProjectIds      []string `bson:"project_ids"`
}
