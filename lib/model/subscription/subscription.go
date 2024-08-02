// Contains database types for the subscription collection.
package subscription

const SUBSCRIPTION_COLLECTION = "subscription"

type Subscription struct {
	Id                      string `bson:"id"`
	Name                    string `bson:"name"`
	DailyPipelineExecutions int    `bson:"daily_pipeline_executions"`
	Deployments             int    `bson:"deployments"`
}
