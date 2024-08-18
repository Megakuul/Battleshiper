// Contains types used for eventbus requests.
package event

type BuildRequest struct {
	ExecutionIdentifier  string `json:"execution_identifier"`
	RepositoryURL        string `json:"repository_url"`
	EventBusName         string `json:"eventbus_name"`
	BuildCommand         string `json:"build_command"`
	BuildAssetBucketPath string `json:"build_asset_bucket_path"`
}

type DeployRequest struct {
	ExecutionIdentifier string `json:"execution_identifier"`
	Successful          bool   `json:"successful"`
	BuildOutput         string `json:"build_output"`
}
