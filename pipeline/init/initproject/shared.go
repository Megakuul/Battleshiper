package initproject

import (
	"context"
	"fmt"

	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
	"go.mongodb.org/mongo-driver/bson"
)

// initializeSharedInfrastructure initializes the shared infrastructure components required by the project.
func initializeSharedInfrastructure(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	projectCollection := eventCtx.Database.Collection(project.PROJECT_COLLECTION)

	result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
		"$set": bson.M{
			"api_route_path":          fmt.Sprintf("project/%s", projectDoc.Name),
			"static_bucket_path":      projectDoc.Name,
			"function_bucket_path":    projectDoc.Name,
			"build_asset_bucket_path": projectDoc.Name,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update project on database")
	} else if result.MatchedCount < 1 {
		return fmt.Errorf("failed to update project: project not found")
	}
	return nil
}
