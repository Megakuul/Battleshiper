package initproject

import (
	"context"
	"fmt"

	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
	"go.mongodb.org/mongo-driver/bson"
)

// initSharedInfrastructure initializes the shared infrastructure components required by the project.
func initSharedInfrastructure(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	projectDoc.SharedInfrastructure = generateSharedInfrastructure(eventCtx, projectDoc.Name)

	projectCollection := eventCtx.Database.Collection(project.PROJECT_COLLECTION)

	result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
		"$set": bson.M{
			"shared_infrastructure": projectDoc.SharedInfrastructure,
		},
	})
	if err != nil || result.MatchedCount < 1 {
		return fmt.Errorf("failed to update project on database")
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
