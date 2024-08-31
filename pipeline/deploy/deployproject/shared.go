package deployproject

import (
	"context"
	"fmt"

	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// initializeSharedInfrastructure initializes the shared infrastructure components required by the project.
func initializeSharedInfrastructure(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) (*project.Project, error) {
	projectCollection := eventCtx.Database.Collection(project.PROJECT_COLLECTION)

	updatedDoc := &project.Project{}
	err := projectCollection.FindOneAndUpdate(transportCtx, bson.M{"_id": projectDoc.MongoID}, bson.M{
		"$set": bson.M{
			"api_route_path":          fmt.Sprintf("project/%s", projectDoc.Name),
			"static_bucket_path":      fmt.Sprintf("%s/%s", eventCtx.BucketConfiguration.StaticBucketName, projectDoc.Name),
			"function_bucket_path":    fmt.Sprintf("%s/%s", eventCtx.BucketConfiguration.FunctionBucketName, projectDoc.Name),
			"build_asset_bucket_path": fmt.Sprintf("%s/%s", eventCtx.BucketConfiguration.BuildAssetBucketName, projectDoc.Name),
		},
	}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updatedDoc)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("failed to update project on database: project not found")
	} else if err != nil {
		return nil, fmt.Errorf("failed to update project on database")
	}
	return updatedDoc, nil
}
