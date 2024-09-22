// Contains database types for the project collection.
package project

const GSI_OWNER_ID = "gsi_owner_id"

type EventResult struct {
	ExecutionIdentifier string `dynamodbav:"execution_identifier"`
	Timepoint           int64  `dynamodbav:"timepoint"`
	Successful          bool   `dynamodbav:"successful"`
}

type BuildResult struct {
	ExecutionIdentifier string `dynamodbav:"execution_identifier"`
	Timepoint           int64  `dynamodbav:"timepoint"`
	Successful          bool   `dynamodbav:"successful"`
}

type DeploymentResult struct {
	ExecutionIdentifier string `dynamodbav:"execution_identifier"`
	Timepoint           int64  `dynamodbav:"timepoint"`
	Successful          bool   `dynamodbav:"successful"`
}

type Repository struct {
	Id     int64  `dynamodbav:"id"`
	URL    string `dynamodbav:"url"`
	Branch string `dynamodbav:"branch"`
}

type DedicatedInfrastructure struct {
	StackName      string `dynamodbav:"stack_name"`
	EventLogGroup  string `dynamodbav:"event_log_group"`
	BuildLogGroup  string `dynamodbav:"build_log_group"`
	DeployLogGroup string `dynamodbav:"deploy_log_group"`
	ServerLogGroup string `dynamodbav:"server_log_group"`
}

type SharedInfrastructure struct {
	StaticBucketPath     string            `dynamodbav:"static_bucket_path"`
	BuildAssetBucketPath string            `dynamodbav:"build_asset_bucket_path"`
	PrerenderPageKeys    map[string]string `dynamodbav:"prerender_page_keys"`
}

// structure is not implemented and will be used for dedicated cdn feature in the future.
type CDNInfrastructure struct {
	Enabled   bool   `dynamodbav:"enabled"`
	StackName string `dynamodbav:"stack_name"`
}

type Project struct {
	Name        string `dynamodbav:"name"`
	OwnerId     string `dynamodbav:"owner_id"`
	Deleted     bool   `dynamodbav:"deleted"`
	Initialized bool   `dynamodbav:"initialized"`
	Status      string `dynamodbav:"status"`

	Repository           Repository          `dynamodbav:"repository"`
	Aliases              map[string]struct{} `dynamodbav:"aliases"`
	BuildImage           string              `dynamodbav:"build_image"`
	BuildCommand         string              `dynamodbav:"build_command"`
	OutputDirectory      string              `dynamodbav:"output_directory"`
	LastEventResult      EventResult         `dynamodbav:"last_event_result"`
	LastBuildResult      BuildResult         `dynamodbav:"last_build_result"`
	LastDeploymentResult DeploymentResult    `dynamodbav:"last_deployment_result"`

	PipelineLock            bool                    `dynamodbav:"pipeline_lock"`
	DedicatedInfrastructure DedicatedInfrastructure `dynamodbav:"dedicated_infrastructure"`
	SharedInfrastructure    SharedInfrastructure    `dynamodbav:"shared_infrastructure"`
	CDNInfrastructure       CDNInfrastructure       `dynamodbav:"cdn_infrastructure"`
}
