// Contains types used for eventbus requests.
package event

type InitRequest struct {
	InitTicket  string `json:"init_ticket"`
	ProjectName string `json:"project_name"`
}

type BuildRequest struct {
	DeployTicket         string `json:"deploy_ticket"`
	ExecutionIdentifier  string `json:"execution_identifier"`
	RepositoryURL        string `json:"repository_url"`
	EventBusName         string `json:"eventbus_name"`
	BuildCommand         string `json:"build_command"`
	BuildAssetBucketPath string `json:"build_asset_bucket_path"`
}

type DeployRequest struct {
	DeployTicket        string `json:"deploy_ticket"`
	ExecutionIdentifier string `json:"execution_identifier"`
	Successful          bool   `json:"successful"`
	BuildOutput         string `json:"build_output"`
}
