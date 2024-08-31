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

// the deploy request is not created manually, but emitted by aws.batch
// https://docs.aws.amazon.com/batch/latest/userguide/batch_cwe_events.html

type DeployParameters struct {
	DeployTicket        string `json:"deploy_ticket"`
	ExecutionIdentifier string `json:"execution_identifier"`
}

type DeployRequest struct {
	Parameters   DeployParameters `json:"parameters"`
	Status       string           `json:"status"`
	StatusReason string           `json:"statusReason"`
}
