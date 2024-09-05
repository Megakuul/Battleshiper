package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type CloudLogger struct {
	client       *cloudwatchlogs.Client
	transportCtx context.Context
	logGroup     string
	logStream    string
	logBuffer    []cloudwatchtypes.InputLogEvent
}

func NewCloudLogger(transportCtx context.Context, client *cloudwatchlogs.Client, logGroupName, logStreamSuffix string) (*CloudLogger, error) {
	logStreamName := fmt.Sprintf("%s/%s", time.Now().Format("2006/01/02"), logStreamSuffix)
	_, err := client.CreateLogStream(transportCtx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(logGroupName),
		LogStreamName: aws.String(logStreamName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create logstream on %s", logGroupName)
	}
	return &CloudLogger{
		client:       client,
		transportCtx: transportCtx,
		logGroup:     logGroupName,
		logStream:    logStreamName,
		logBuffer:    []cloudwatchtypes.InputLogEvent{},
	}, nil
}

// WriteLog writes a log event to a local buffer.
func (c *CloudLogger) WriteLog(format string, args ...interface{}) {
	c.logBuffer = append(c.logBuffer, cloudwatchtypes.InputLogEvent{
		Message:   aws.String(fmt.Sprintf(format, args...)),
		Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
	})
}

// PushLogs pushes the current log event buffer to cloudwatch.
func (c *CloudLogger) PushLogs() error {
	if len(c.logBuffer) < 1 {
		return nil
	}

	_, err := c.client.PutLogEvents(c.transportCtx, &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(c.logGroup),
		LogStreamName: aws.String(c.logStream),
		LogEvents:     c.logBuffer,
	})
	if err != nil {
		return fmt.Errorf("failed to send logevents to %s - %s", c.logGroup, c.logStream)
	}
	c.logBuffer = []cloudwatchtypes.InputLogEvent{}
	return nil
}
