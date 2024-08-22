// Contains database types for the subscription collection.
package subscription

const SUBSCRIPTION_COLLECTION = "subscription"

type Subscription struct {
	MongoID                  interface{} `bson:"_id"`
	Id                       string      `bson:"id"`
	Name                     string      `bson:"name"`
	DailyPipelineBuilds      int         `bson:"daily_pipeline_builds"`
	DailyPipelineDeployments int         `bson:"daily_pipeline_deployments"`
	Projects                 int         `bson:"projects"`
}
