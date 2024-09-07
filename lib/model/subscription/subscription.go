// Contains database types for the subscription collection.
package subscription

const SUBSCRIPTION_COLLECTION = "subscription"

type PipelineSpecs struct {
	DailyBuilds      int64 `bson:"daily_builds"`
	DailyDeployments int64 `bson:"daily_deployments"`
}

type ProjectSpecs struct {
	ProjectCount     int64 `bson:"project_count"`
	AliasCount       int64 `bson:"alias_count"`
	PrerenderRoutes  int64 `bson:"prerender_routes"`
	ServerStorage    int64 `bson:"server_storage"`
	ClientStorage    int64 `bson:"client_storage"`
	PrerenderStorage int64 `bson:"prerender_storage"`
}

type CDNSpecs struct {
	InstanceCount int64 `bson:"instance_count"`
}

type Subscription struct {
	MongoID       interface{}   `bson:"_id"`
	Id            string        `bson:"id"`
	Name          string        `bson:"name"`
	PipelineSpecs PipelineSpecs `bson:"pipeline_specs"`
	ProjectSpecs  ProjectSpecs  `bson:"project_specs"`
	CDNSpecs      CDNSpecs      `bson:"cdn_specs"`
}
