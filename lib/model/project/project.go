// Contains database types for the project collection.
package project

const PROJECT_COLLECTION = "project"

type Project struct {
	Id         string `bson:"id"`
	Deleted    bool   `bson:"deleted"`
	Name       string `bson:"name"`
	Repository string `bson:"repository"`
	OwnerId    string `bson:"owner_id"`
}
