package deleteprojects

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/delete/eventcontext"
	"go.mongodb.org/mongo-driver/bson"
)

func HandleDeleteProjects(eventCtx eventcontext.Context) func(context.Context, events.CloudWatchEvent) error {
	return func(ctx context.Context, event events.CloudWatchEvent) error {
		err := runHandleDeleteProjects(event, ctx, eventCtx)
		if err != nil {
			log.Printf("ERROR DELETEPROJECTS: %v\n", err)
			return err
		}
		return nil
	}
}

func runHandleDeleteProjects(request events.CloudWatchEvent, transportCtx context.Context, eventCtx eventcontext.Context) error {
	projectCollection := eventCtx.Database.Collection(project.PROJECT_COLLECTION)

	projectDoc := &project.Project{}
	err := projectCollection.FindOneAndUpdate(transportCtx, bson.D{
		{Key: "name", Value: deployClaims.Project},
		{Key: "owner_id", Value: deployClaims.UserID},
	}, bson.M{
		// Lock the pipeline, this step is used to ensure only one deployment runs at a time.
		// running multiple deployments at the same time should not cause major issues,
		// however it can cause weird or unintended behavior for the project.
		"$set": bson.M{
			"pipeline_lock": true,
		},
	}).Decode(&projectDoc)
	if err != nil {
		return fmt.Errorf("failed to fetch project from database")
	}
	if projectDoc.PipelineLock {
		return fmt.Errorf("project locked")
	}

	// Finish build step
	buildResult := project.BuildResult{
		ExecutionIdentifier: deployRequest.Parameters.ExecutionIdentifier,
		Timepoint:           time.Now().Unix(),
	}
	if strings.ToUpper(deployRequest.Status) != "SUCCEEDED" {
		buildResult.Successful = false
		result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
			"$set": bson.M{
				"last_build_result": buildResult,
				"status":            fmt.Errorf("BUILD FAILED: %v", err),
			},
		})
		if err != nil && result.MatchedCount < 1 {
			return fmt.Errorf("failed to update project (last_build_result)")
		}
		return nil
	} else {
		buildResult.Successful = true
		result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
			"$set": bson.M{
				"last_build_result": buildResult,
			},
		})
		if err != nil && result.MatchedCount < 1 {
			return fmt.Errorf("failed to update project (last_build_result)")
		}
	}

	// Start actual deployment step
	deploymentResult := project.DeploymentResult{
		ExecutionIdentifier: deployRequest.Parameters.ExecutionIdentifier,
	}
	if err := deployProject(transportCtx, eventCtx, projectDoc, deployClaims.UserID, deployRequest.Parameters.ExecutionIdentifier); err != nil {
		deploymentResult.Timepoint = time.Now().Unix()
		deploymentResult.Successful = false
		result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
			"$set": bson.M{
				"last_deployment_result": deploymentResult,
				"status":                 fmt.Errorf("DEPLOYMENT FAILED: %v", err),
			},
		})
		if err != nil && result.MatchedCount < 1 {
			return fmt.Errorf("failed to update project (last_deployment_result)")
		}
		return nil
	} else {
		deploymentResult.Timepoint = time.Now().Unix()
		deploymentResult.Successful = true
		result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
			"$set": bson.M{
				"last_deployment_result": deploymentResult,
				"status":                 "",
			},
		})
		if err != nil && result.MatchedCount < 1 {
			return fmt.Errorf("failed to update project (last_deployment_result)")
		}
	}

	return nil
}
