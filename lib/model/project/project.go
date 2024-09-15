// Contains database types for the project collection.
package project

const PROJECT_COLLECTION = "project"

type EventResult struct {
	ExecutionIdentifier string `bson:"execution_identifier"`
	Timepoint           int64  `bson:"timepoint"`
	Successful          bool   `bson:"successful"`
}

type BuildResult struct {
	ExecutionIdentifier string `bson:"execution_identifier"`
	Timepoint           int64  `bson:"timepoint"`
	Successful          bool   `bson:"successful"`
}

type DeploymentResult struct {
	ExecutionIdentifier string `bson:"execution_identifier"`
	Timepoint           int64  `bson:"timepoint"`
	Successful          bool   `bson:"successful"`
}

type Repository struct {
	Id     int64  `bson:"id"`
	URL    string `bson:"url"`
	Branch string `bson:"branch"`
}

type DedicatedInfrastructure struct {
	StackName      string `bson:"stack_name"`
	EventLogGroup  string `bson:"event_log_group"`
	BuildLogGroup  string `bson:"build_log_group"`
	DeployLogGroup string `bson:"deploy_log_group"`
	ServerLogGroup string `bson:"server_log_group"`
}

type SharedInfrastructure struct {
	StaticBucketPath     string            `bson:"static_bucket_path"`
	BuildAssetBucketPath string            `bson:"build_asset_bucket_path"`
	PrerenderPageKeys    map[string]string `bson:"prerender_page_keys"`
}

type CDNInfrastructure struct {
	Enabled   bool   `bson:"enabled"`
	StackName string `bson:"stack_name"`
}

type Project struct {
	MongoID     interface{} `bson:"_id"`
	Name        string      `bson:"name"`
	OwnerId     string      `bson:"owner_id"`
	Deleted     bool        `bson:"deleted"`
	Initialized bool        `bson:"initialized"`
	Status      string      `bson:"status"`

	Repository           Repository          `bson:"repository"`
	Aliases              map[string]struct{} `bson:"aliases"`
	BuildImage           string              `bson:"build_image"`
	BuildCommand         string              `bson:"build_command"`
	OutputDirectory      string              `bson:"output_directory"`
	LastEventResult      EventResult         `bson:"last_event_result"`
	LastBuildResult      BuildResult         `bson:"last_build_result"`
	LastDeploymentResult DeploymentResult    `bson:"last_deployment_result"`

	PipelineLock            bool                    `bson:"pipeline_lock"`
	DedicatedInfrastructure DedicatedInfrastructure `bson:"dedicated_infrastructure"`
	SharedInfrastructure    SharedInfrastructure    `bson:"shared_infrastructure"`
	CDNInfrastructure       CDNInfrastructure       `bson:"cdn_infrastructure"`
}
