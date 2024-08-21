package eventcontext

import (
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"go.mongodb.org/mongo-driver/mongo"
)

// Context provides data to event handlers.
type Context struct {
	Database      *mongo.Database
	TicketOptions *pipeline.TicketOptions
}
