// Contains database types for the subscription collection.
package subscription

type PipelineSpecs struct {
	DailyBuilds      int64 `dynamodbav:"daily_builds"`
	DailyDeployments int64 `dynamodbav:"daily_deployments"`
}

type ProjectSpecs struct {
	ProjectCount     int64 `dynamodbav:"project_count"`
	AliasCount       int64 `dynamodbav:"alias_count"`
	PrerenderRoutes  int64 `dynamodbav:"prerender_routes"`
	ServerStorage    int64 `dynamodbav:"server_storage"`
	ClientStorage    int64 `dynamodbav:"client_storage"`
	PrerenderStorage int64 `dynamodbav:"prerender_storage"`
}

type CDNSpecs struct {
	InstanceCount int64 `dynamodbav:"instance_count"`
}

type Subscription struct {
	MongoID       interface{}   `dynamodbav:"_id"`
	Id            string        `dynamodbav:"id"`
	Name          string        `dynamodbav:"name"`
	PipelineSpecs PipelineSpecs `dynamodbav:"pipeline_specs"`
	ProjectSpecs  ProjectSpecs  `dynamodbav:"project_specs"`
	CDNSpecs      CDNSpecs      `dynamodbav:"cdn_specs"`
}
