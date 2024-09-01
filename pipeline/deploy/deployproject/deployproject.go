package deployproject

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
	"go.mongodb.org/mongo-driver/bson"
)

func HandleDeployProject(eventCtx eventcontext.Context) func(context.Context, events.CloudWatchEvent) error {
	return func(ctx context.Context, event events.CloudWatchEvent) error {
		err := runHandleDeployProject(event, ctx, eventCtx)
		if err != nil {
			log.Printf("ERROR DEPLOYPROJECT: %v\n", err)
			return err
		}
		return nil
	}
}

func runHandleDeployProject(request events.CloudWatchEvent, transportCtx context.Context, eventCtx eventcontext.Context) error {
	deployRequest := &event.DeployRequest{}
	if err := json.Unmarshal(request.Detail, &deployRequest); err != nil {
		return fmt.Errorf("failed to deserialize deploy request")
	}

	deployClaims, err := pipeline.ParseTicket(eventCtx.TicketOptions, deployRequest.Parameters.DeployTicket)
	if err != nil {
		return fmt.Errorf("failed to parse ticket: %v", err)
	}

	if deployClaims.Action != request.DetailType {
		return fmt.Errorf("action mismatch: provided ticket was not issued for the specified action")
	}

	projectCollection := eventCtx.Database.Collection(project.PROJECT_COLLECTION)

	projectDoc := &project.Project{}
	err = projectCollection.FindOne(transportCtx, bson.D{
		{Key: "name", Value: deployClaims.Project},
		{Key: "owner_id", Value: deployClaims.UserID},
	}).Decode(&projectDoc)
	if err != nil {
		return fmt.Errorf("failed to project from database")
	}

	if err := finishBuild(transportCtx, eventCtx, projectDoc, deployRequest); err != nil {
		return fmt.Errorf("failed to finish build job: %v", err)
	}

	if err := deployProject(transportCtx, eventCtx, projectDoc, deployRequest); err != nil {
		return fmt.Errorf("failed to deploy project: %v", err)
	}

	return nil
}

func finishBuild(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, deployRequest *event.DeployRequest) error {
	buildResult := project.BuildResult{
		ExecutionIdentifier: deployRequest.Parameters.ExecutionIdentifier,
		Timepoint:           time.Now().Unix(),
	}

	projectCollection := eventCtx.Database.Collection(project.PROJECT_COLLECTION)

	if strings.ToUpper(deployRequest.Status) != "SUCCEEDED" {
		buildResult.Successful = false
		result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
			"$set": bson.M{
				"last_build_result": buildResult,
			},
		})
		if err != nil && result.MatchedCount < 1 {
			return fmt.Errorf("failed to update last_build_result")
		}
		return nil
	}

	buildResult.Successful = true
	result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
		"$set": bson.M{
			"last_build_result": buildResult,
		},
	})
	if err != nil && result.MatchedCount < 1 {
		return fmt.Errorf("failed to update last_build_result")
	}

	return nil
}

func deployProject(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, deployRequest *event.DeployRequest) error {
	deploymentResult := project.DeploymentResult{
		ExecutionIdentifier: deployRequest.Parameters.ExecutionIdentifier,
	}

	logStreamIdentifier := fmt.Sprintf("%s/%s", time.Now().Format("2006/01/02"), deployRequest.Parameters.ExecutionIdentifier)
	_, err := eventCtx.CloudwatchClient.CreateLogStream(transportCtx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(projectDoc.DedicatedInfrastructure.DeployLogGroup),
		LogStreamName: aws.String(logStreamIdentifier),
	})
	if err != nil {
		return fmt.Errorf("failed to create logstream on %s. %v", projectDoc.DedicatedInfrastructure.DeployLogGroup, err)
	}

	logEvents := []cloudwatchtypes.InputLogEvent{cloudwatchtypes.InputLogEvent{
		Message:   aws.String(fmt.Sprintf("START DEPLOYMENT %s", deployRequest.Parameters.ExecutionIdentifier)),
		Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
	}}

	_, err = eventCtx.CloudwatchClient.PutLogEvents(transportCtx, &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(projectDoc.DedicatedInfrastructure.DeployLogGroup),
		LogStreamName: aws.String(logStreamIdentifier),
		LogEvents:     logEvents,
	})
	if err != nil {
		return fmt.Errorf("failed to send logevents to %s. %v", projectDoc.DedicatedInfrastructure.DeployLogGroup, err)
	}

	cancelDeployment := false

	err = analyzeBuildAssets(transportCtx, routeCtx, execIdentifier, userDoc, projectDoc)
	if err != nil {
		cancelDeployment = true
		logEvents = append(logEvents, cloudwatchtypes.InputLogEvent{
			Message:   aws.String(err.Error()),
			Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
		})
	}

	err = updateDedicatedInfrastructure(transportCtx, routeCtx, execIdentifier, userDoc, projectDoc)
	if err != nil {
		deploymentResult.Successful = false
		logEvents = append(logEvents, cloudwatchtypes.InputLogEvent{
			Message:   aws.String(err.Error()),
			Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
		})
	}

	deploymentResult.Timepoint = time.Now().Unix()

	projectCollection := eventCtx.Database.Collection(project.PROJECT_COLLECTION)
	result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
		"$set": bson.M{
			"last_deployment_result": deploymentResult,
		},
	})
	if err != nil && result.MatchedCount < 1 {
		return fmt.Errorf("failed to update last_deployment_result")
	}

	return nil
}
