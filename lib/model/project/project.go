// Contains database types for the project collection.
package project

const PROJECT_COLLECTION = "project"

type BuildResult struct {
	Successful       bool   `bson:"successful"`
	DeploymentOutput string `bson:"deployment_output"`
	BuildOutput      string `bson:"build_output"`
}

type Repository struct {
	Id     int64  `bson:"id"`
	URL    string `bson:"url"`
	Branch string `bson:"branch"`
}

type Project struct {
	Name            string      `bson:"name"`
	Deleted         bool        `bson:"deleted"`
	Repository      Repository  `bson:"repository"`
	BuildCommand    string      `bson:"build_command"`
	LogGroup        string      `bson:"log_group"`
	LastBuildResult BuildResult `bson:"last_build_result"`
	OwnerId         string      `bson:"owner_id"`
}
