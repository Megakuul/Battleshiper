package eventcontext

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"go.mongodb.org/mongo-driver/mongo"
)

// Context provides data to event handlers.
type Context struct {
	Database             *mongo.Database
	TicketOptions        *pipeline.TicketOptions
	CloudformationClient *cloudformation.Client
}
