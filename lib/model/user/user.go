package user

type Subscriptions struct {
	DailyPipelineExecutions int `bson:"daily_pipeline_executions"`
	DefaultDeployments      int `bson:"default_deployments"`
}

type User struct {
	Sub           string        `bson:"sub"`
	Subscriptions Subscriptions `bson:"subscriptions"`
}
