package initproject

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/init/eventcontext"
)

// initSharedInfrastructure initializes the shared infrastructure components required by the project.
func initSharedInfrastructure(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	projectDoc.SharedInfrastructure = generateSharedInfrastructure(eventCtx, projectDoc.Name)

	sharedInfrastructureAttributes, err := attributevalue.Marshal(&projectDoc.SharedInfrastructure)
	if err != nil {
		return fmt.Errorf("failed to serialize shared infrastructure")
	}

	_, err = database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(eventCtx.ProjectTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.Name},
		},
		AttributeNames: map[string]string{
			"#shared_infrastructure": "shared_infrastructure",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":shared_infrastructure": sharedInfrastructureAttributes,
		},
		UpdateExpr: aws.String("SET #shared_infrastructure = :shared_infrastructure"),
	})
	if err != nil {
		return fmt.Errorf("failed to update project: %v", err)
	}
	return nil
}

// generateSharedInfrastructure generates the initial shared infrastructure config based on eventCtx and the projectName.
func generateSharedInfrastructure(eventCtx eventcontext.Context, projectName string) project.SharedInfrastructure {
	infrastructure := project.SharedInfrastructure{}

	infrastructure.StaticBucketPath = fmt.Sprintf("%s/%s", eventCtx.BucketConfiguration.StaticBucketName, projectName)
	infrastructure.BuildAssetBucketPath = fmt.Sprintf("%s/%s", eventCtx.BucketConfiguration.BuildAssetBucketName, projectName)
	infrastructure.PrerenderPageKeys = map[string]string{}

	return infrastructure
}
