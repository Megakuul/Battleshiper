package deployproject

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
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

	projectDoc := &project.Project{}
	// MIG: Possible with query item and primary key + condition on owner_id + deleted
	err = projectCollection.FindOne(transportCtx, bson.D{
		{Key: "name", Value: deployClaims.Project},
		{Key: "owner_id", Value: deployClaims.UserID},
		{Key: "deleted", Value: false},
	}).Decode(&projectDoc)
	if err != nil {
		return fmt.Errorf("failed to fetch project from database")
	}

	// Finish build step
	buildResult := project.BuildResult{
		ExecutionIdentifier: deployRequest.Parameters.ExecutionIdentifier,
		Timepoint:           time.Now().Unix(),
	}
	if strings.ToUpper(deployRequest.Status) != "SUCCEEDED" {
		buildResult.Successful = false
		// MIG: Possible with update item and primary key
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
		// MIG: Possible with update item and primary key
		result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
			"$set": bson.M{
				"last_build_result": buildResult,
			},
		})
		if err != nil && result.MatchedCount < 1 {
			return fmt.Errorf("failed to update project (last_build_result)")
		}
	}

	// MIG: Possible with update item and primary key + condition owner_id + deleted
	err = projectCollection.FindOneAndUpdate(transportCtx, bson.D{
		{Key: "name", Value: deployClaims.Project},
		{Key: "owner_id", Value: deployClaims.UserID},
		{Key: "deleted", Value: false},
	}, bson.M{
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

	// Start actual deployment step
	deploymentResult := project.DeploymentResult{
		ExecutionIdentifier: deployRequest.Parameters.ExecutionIdentifier,
	}
	if err := deployProject(transportCtx, eventCtx, projectDoc, deployClaims.UserID, deployRequest.Parameters.ExecutionIdentifier); err != nil {
		deploymentResult.Timepoint = time.Now().Unix()
		deploymentResult.Successful = false
		// MIG: Possible with update item and primary key
		result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
			"$set": bson.M{
				"last_deployment_result": deploymentResult,
				"status":                 fmt.Errorf("DEPLOYMENT FAILED: %v", err),
				"pipeline_lock":          false,
			},
		})
		if err != nil && result.MatchedCount < 1 {
			return fmt.Errorf("failed to update project (last_deployment_result)")
		}
		return nil
	} else {
		deploymentResult.Timepoint = time.Now().Unix()
		deploymentResult.Successful = true
		// MIG: Possible with update item and primary key
		result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
			"$set": bson.M{
				"last_deployment_result": deploymentResult,
				"status":                 "",
				"pipeline_lock":          false,
			},
		})
		if err != nil && result.MatchedCount < 1 {
			return fmt.Errorf("failed to update project (last_deployment_result)")
		}
	}

	return nil
}

func deployProject(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, userId string, execId string) error {
	cloudLogger, err := pipeline.NewCloudLogger(
		transportCtx,
		eventCtx.CloudwatchClient,
		projectDoc.DedicatedInfrastructure.DeployLogGroup,
		execId,
	)
	if err != nil {
		return err
	}

	cloudLogger.WriteLog("START DEPLOYMENT %s", execId)
	cloudLogger.WriteLog("loading user subscription...")

	subscriptionDoc := &subscription.Subscription{}
	// MIG: Possible with query item and primary key
	err = subscriptionCollection.FindOne(transportCtx, bson.M{
		"id": userId,
	}).Decode(&subscriptionDoc)
	if err != nil {
		cloudLogger.WriteLog("failed to fetch subscription from database")
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return fmt.Errorf("failed to fetch subscription from database")
	}

	cloudLogger.WriteLog("analyzing build assets...")
	buildInformation, err := analyzeBuildAssets(transportCtx, eventCtx, projectDoc, subscriptionDoc, execId)
	if err != nil {
		cloudLogger.WriteLog(err.Error())
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return err
	}

	// the stack state is validated to provide a more descriptive error message to the user
	cloudLogger.WriteLog("validating stack state...")
	err = validateStackState(transportCtx, eventCtx, projectDoc)
	if err != nil {
		cloudLogger.WriteLog(err.Error())
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return err
	}

	cloudLogger.WriteLog("creating stack changeset...")
	changeSetName, err := createChangeSet(transportCtx, eventCtx, projectDoc, execId, buildInformation.ServerObject)
	if err != nil {
		cloudLogger.WriteLog(err.Error())
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return err
	}

	cloudLogger.WriteLog("describing stack changeset...")
	changeSetDescription, err := describeChangeSet(transportCtx, eventCtx, projectDoc, changeSetName)
	if err != nil {
		cloudLogger.WriteLog(err.Error())
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return err
	}
	cloudLogger.WriteLog(changeSetDescription)

	cloudLogger.WriteLog("executing stack changeset...")
	if err := cloudLogger.PushLogs(); err != nil {
		return err
	}
	err = executeChangeSet(transportCtx, eventCtx, projectDoc, changeSetName)
	if err != nil {
		cloudLogger.WriteLog(err.Error())
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return err
	}

	cloudLogger.WriteLog("updating page keys on database...")
	if err := cloudLogger.PushLogs(); err != nil {
		return err
	}
	// MIG: Possible with update item and primary key
	result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
		"$set": bson.M{
			"shared_infrastructure.prerender_page_keys": buildInformation.PageKeys,
		},
	})
	if err != nil || result.MatchedCount < 1 {
		cloudLogger.WriteLog("failed to update page keys on database")
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return fmt.Errorf("failed to update page keys on database")
	}

	cloudLogger.WriteLog("updating page keys on cdn...")
	err = updateStaticPageKeys(transportCtx, eventCtx, buildInformation.PageKeys, projectDoc.SharedInfrastructure.PrerenderPageKeys)
	if err != nil {
		cloudLogger.WriteLog(err.Error())
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return err
	}

	cloudLogger.WriteLog("removing old static asset data...")
	if err := cloudLogger.PushLogs(); err != nil {
		return err
	}
	err = cleanStaticBucket(transportCtx, eventCtx, projectDoc)
	if err != nil {
		cloudLogger.WriteLog(err.Error())
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return err
	}

	cloudLogger.WriteLog("transferring new static assets...")
	err = copyStaticAssets(transportCtx, eventCtx, projectDoc, buildInformation.ClientObjects)
	if err != nil {
		cloudLogger.WriteLog(err.Error())
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return err
	}

	cloudLogger.WriteLog("transferring new static pages...")
	err = copyStaticPages(transportCtx, eventCtx, projectDoc, buildInformation.PrerenderedObjects)
	if err != nil {
		cloudLogger.WriteLog(err.Error())
		if err := cloudLogger.PushLogs(); err != nil {
			return err
		}
		return err
	}

	if err := cloudLogger.PushLogs(); err != nil {
		return err
	}

	return nil
}
