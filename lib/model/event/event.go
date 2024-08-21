// Contains types used for eventbus requests.
package event

type EventEndpoint struct {
	EventBus string `json:"eventbus"`
	Source   string `json:"source"`
	Action   string `json:"action"`
	Ticket   string `json:"ticket"`
}

type InitRequest struct {
	InitTicket  string `json:"init_ticket"`
	ProjectName string `json:"project_name"`
}

type BuildRequest struct {
	ExecutionIdentifier  string        `json:"execution_identifier"`
	RepositoryURL        string        `json:"repository_url"`
	BuildCommand         string        `json:"build_command"`
	BuildAssetBucketPath string        `json:"build_asset_bucket_path"`
	DeployEndpoint       EventEndpoint `json:"deploy_endpoint"`
}

type DeployRequest struct {
	DeployTicket        string `json:"deploy_ticket"`
	ExecutionIdentifier string `json:"execution_identifier"`
	Successful          bool   `json:"successful"`
	BuildOutput         string `json:"build_output"`
}
