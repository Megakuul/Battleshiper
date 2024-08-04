// Contains database types for the project collection.
package project

const PROJECT_COLLECTION = "project"

type Repository struct {
	Id     int64  `bson:"id"`
	URL    string `bson:"url"`
	Branch string `bson:"branch"`
}

type Project struct {
	Name         string     `bson:"name"`
	Deleted      bool       `bson:"deleted"`
	Repository   Repository `bson:"repository"`
	BuildCommand string     `bson:"build_command"`
	OwnerId      string     `bson:"owner_id"`
}
