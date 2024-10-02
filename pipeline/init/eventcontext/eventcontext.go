package eventcontext

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
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
	EventLogPrefix    string
	BuildLogPrefix    string
	DeployLogPrefix   string
	ServerLogPrefix   string
	LogRetentionDays  int
	BuildEventbusName string
	BuildEventSource  string
	BuildEventAction  string
	BuildJobQueueArn  string
	BuildJobTimeout   time.Duration
	BuildJobVCPUS     string
	BuildJobMemory    string
}

// Context provides data to event handlers.
type Context struct {
	DynamoClient            *dynamodb.Client
	UserTable               string
	ProjectTable            string
	SubscriptionTable       string
	TicketOptions           *pipeline.TicketOptions
	CloudformationClient    *cloudformation.Client
	DeploymentConfiguration *DeploymentConfiguration
	BucketConfiguration     *BucketConfiguration
	ProjectConfiguration    *ProjectConfiguration
}
