package initproject

import (
	"context"
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

// initializeDedicatedInfrastructure initializes the dedicated infrastructure components required by the project.
func initializeDedicatedInfrastructure(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	app := awscdk.NewApp(nil)
	stackName := fmt.Sprintf("battleshiper-project-stack-%s", projectDoc.Name)
	projectStack := awscdk.NewStack(app, aws.String("Project"), &awscdk.StackProps{
		StackName: aws.String(stackName),
		Tags: &map[string]*string{
			"Name": aws.String(projectDoc.Name),
		},
	})

	app.Synth(nil)

	// - create stack
	// - add event rule sending the request to the batch
	// - setup aws batch with permissions to insert to the asset-bucket and to send to battleshiper.deploy
}
