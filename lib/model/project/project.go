// Contains database types for the project collection.
package project

const PROJECT_COLLECTION = "project"

type EventResult struct {
	ExecutionIdentifier string `bson:"execution_identifier"`
	Timepoint           int64  `bson:"timepoint"`
	Successful          bool   `bson:"successful"`
	EventOutput         string `bson:"event_output"`
}

type BuildResult struct {
	ExecutionIdentifier string `bson:"execution_identifier"`
	Timepoint           int64  `bson:"timepoint"`
	Successful          bool   `bson:"successful"`
	BuildOutput         string `bson:"build_output"`
}

type DeploymentResult struct {
	ExecutionIdentifier string `bson:"execution_identifier"`
	Timepoint           int64  `bson:"timepoint"`
	Successful          bool   `bson:"successful"`
	DeploymentOutput    string `bson:"deployment_output"`
}

type Repository struct {
	Id     int64  `bson:"id"`
	URL    string `bson:"url"`
	Branch string `bson:"branch"`
}

type DedicatedInfrastructure struct {
	StackName    string `bson:"stack_name"`
	LogGroupName string `bson:"log_group_name"`
}

type SharedInfrastructure struct {
	ApiRoutePath         string `bson:"api_route_path"`
	StaticBucketPath     string `bson:"static_bucket_path"`
	FunctionBucketPath   string `bson:"function_bucket_path"`
	BuildAssetBucketPath string `bson:"build_asset_bucket_path"`
}

type Project struct {
	MongoID              interface{}      `bson:"_id"`
	Name                 string           `bson:"name"`
	OwnerId              string           `bson:"owner_id"`
	Deleted              bool             `bson:"deleted"`
	Initialized          bool             `bson:"initialized"`
	Status               string           `bson:"status"`
	Repository           Repository       `bson:"repository"`
	BuildImage           string           `bson:"build_image"`
	BuildCommand         string           `bson:"build_command"`
	OutputDirectory      string           `bson:"output_directory"`
	LastEventResult      EventResult      `bson:"last_event_result"`
	LastBuildResult      BuildResult      `bson:"last_build_result"`
	LastDeploymentResult DeploymentResult `bson:"last_deployment_result"`

	DedicatedInfrastructure DedicatedInfrastructure `bson:"dedicated_infrastructure"`
	SharedInfrastructure    SharedInfrastructure    `bson:"dedicated_infrastructure"`
}
