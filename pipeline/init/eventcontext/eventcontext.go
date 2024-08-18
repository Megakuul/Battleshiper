package eventcontext

import (
	"go.mongodb.org/mongo-driver/mongo"
)

// Context provides data to event handlers.
type Context struct {
	Database *mongo.Database
}
