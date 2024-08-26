package eventcontext

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"go.mongodb.org/mongo-driver/mongo"
)

type BucketConfiguration struct {
	StaticBucketName     string
	FunctionBucketName   string
	BuildAssetBucketName string
}

type BuildConfiguration struct {
	BuildEventbusName      string
	BuildEventSource       string
	BuildEventAction       string
	BuildJobQueueArn       string
	BuildJobQueuePolicyArn string
	BuildJobTimeout        time.Duration
	BuildJobVCPUS          int
	BuildJobMemory         int
}

// Context provides data to event handlers.
type Context struct {
	Database             *mongo.Database
	TicketOptions        *pipeline.TicketOptions
	CloudformationClient *cloudformation.Client
	DeploymentTimeout    time.Duration
	BucketConfiguration  *BucketConfiguration
	BuildConfiguration   *BuildConfiguration
}
