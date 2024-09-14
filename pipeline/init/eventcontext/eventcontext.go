package eventcontext

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"go.mongodb.org/mongo-driver/mongo"
)

type DeploymentConfiguration struct {
	ServiceRoleArn string
	Timeout        time.Duration
}

type BucketConfiguration struct {
	StaticBucketName     string
	BuildAssetBucketName string
}

type ProjectConfiguration struct {
	EventLogPrefix         string
	BuildLogPrefix         string
	DeployLogPrefix        string
	ServerLogPrefix        string
	LogRetentionDays       int
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
	Database                *mongo.Database
	TicketOptions           *pipeline.TicketOptions
	CloudformationClient    *cloudformation.Client
	DeploymentConfiguration *DeploymentConfiguration
	BucketConfiguration     *BucketConfiguration
	ProjectConfiguration    *ProjectConfiguration
}
