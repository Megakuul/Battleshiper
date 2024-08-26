// Contains types used for eventbus requests.
package event

type InitRequest struct {
	InitTicket string `json:"init_ticket"`
}

type BuildRequest struct {
	DeployTicket        string `json:"deploy_ticket"`
	ExecutionIdentifier string `json:"execution_identifier"`
	RepositoryURL       string `json:"repository_url"`
	BuildCommand        string `json:"build_command"`
	OutputDirectory     string `json:"output_directory"`
}

type DeployRequest struct {
	DeployTicket        string `json:"deploy_ticket"`
	ExecutionIdentifier string `json:"execution_identifier"`
	Successful          bool   `json:"successful"`
	BuildOutput         string `json:"build_output"`
}
