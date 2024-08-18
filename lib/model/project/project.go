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

type Project struct {
	Name                 string           `bson:"name"`
	OwnerId              string           `bson:"owner_id"`
	Deleted              bool             `bson:"deleted"`
	LogGroup             string           `bson:"log_group"`
	Repository           Repository       `bson:"repository"`
	BuildCommand         string           `bson:"build_command"`
	BuildAssetBucketPath string           `bson:"build_asset_bucket_path"`
	LastEventResult      EventResult      `bson:"last_event_result"`
	LastBuildResult      BuildResult      `bson:"last_build_result"`
	LastDeploymentResult DeploymentResult `bson:"last_deployment_result"`
}
