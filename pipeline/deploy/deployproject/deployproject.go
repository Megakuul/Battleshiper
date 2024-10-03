package deployproject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/aws/aws-lambda-go/events"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/helper/pipeline"
	"github.com/megakuul/battleshiper/lib/model/event"
	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/lib/model/subscription"
	"github.com/megakuul/battleshiper/lib/model/user"
	"github.com/megakuul/battleshiper/pipeline/deploy/eventcontext"
)

var logger = log.New(os.Stderr, "DEPLOY DEPLOYPROJECT: ", 0)

func HandleDeployProject(eventCtx eventcontext.Context) func(context.Context, events.CloudWatchEvent) error {
	return func(ctx context.Context, event events.CloudWatchEvent) error {
		err := runHandleDeployProject(event, ctx, eventCtx)
		if err != nil {
			logger.Printf("%v\n", err)
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

	userDoc, err := database.GetSingle[user.User](transportCtx, eventCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(eventCtx.UserTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: deployClaims.UserID},
		},
		ConditionExpr: aws.String("id = :id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to load user record from database: %v", err)
	}

	projectDoc, err := database.GetSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(eventCtx.ProjectTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":project_name": &dynamodbtypes.AttributeValueMemberS{Value: deployClaims.Project},
		},
		ConditionExpr: aws.String("project_name = :project_name"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return fmt.Errorf("project not found")
		}
		return fmt.Errorf("failed to load project from database: %v", err)
	}
	if projectDoc.OwnerId != deployClaims.UserID {
		return fmt.Errorf("user '%s' is not authorized to deploy this project", deployClaims.UserID)
	}
	if projectDoc.Deleted {
		return fmt.Errorf("project cannot be deployed: it is marked for deletion")
	}

	// Finish build step
	buildResult := project.BuildResult{
		ExecutionIdentifier: deployRequest.Parameters.ExecutionIdentifier,
		Timepoint:           time.Now().Unix(),
	}
	if strings.ToUpper(deployRequest.Status) != "SUCCEEDED" {
		buildResult.Successful = false

		buildResultAttributes, sErr := attributevalue.Marshal(&buildResult)
		if sErr != nil {
			return fmt.Errorf("failed to serialize buildresult")
		}

		_, uErr := database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
			Table: aws.String(eventCtx.ProjectTable),
			PrimaryKey: map[string]dynamodbtypes.AttributeValue{
				"project_name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.ProjectName},
			},
			AttributeNames: map[string]string{
				"#last_build_result": "last_build_result",
				"#status":            "status",
			},
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":last_build_result": buildResultAttributes,
				":status":            &dynamodbtypes.AttributeValueMemberS{Value: "BUILD FAILED"},
			},
			UpdateExpr: aws.String("SET #last_build_result = :last_build_result, #status = :status"),
		})
		if uErr != nil {
			return fmt.Errorf("failed to update project: %v", uErr)
		}
		return nil
	} else {
		buildResult.Successful = true

		buildResultAttributes, sErr := attributevalue.Marshal(&buildResult)
		if sErr != nil {
			return fmt.Errorf("failed to serialize buildresult")
		}

		_, uErr := database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
			Table: aws.String(eventCtx.ProjectTable),
			PrimaryKey: map[string]dynamodbtypes.AttributeValue{
				"project_name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.ProjectName},
			},
			AttributeNames: map[string]string{
				"#last_build_result": "last_build_result",
			},
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":last_build_result": buildResultAttributes,
			},
			UpdateExpr: aws.String("SET #last_build_result = :last_build_result"),
		})
		if uErr != nil {
			return fmt.Errorf("failed to update project: %v", uErr)
		}
	}

	projectDoc, err = database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(eventCtx.ProjectTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"project_name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.ProjectName},
		},
		AttributeNames: map[string]string{
			"#deleted":       "deleted",
			"#pipeline_lock": "pipeline_lock",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":deleted":       &dynamodbtypes.AttributeValueMemberBOOL{Value: false},
			":pipeline_lock": &dynamodbtypes.AttributeValueMemberBOOL{Value: true},
		},
		ConditionExpr: aws.String("#deleted = :deleted"),
		UpdateExpr:    aws.String("SET #pipeline_lock = :pipeline_lock"),
		ReturnOld:     true,
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return fmt.Errorf("project not found")
		}
		return fmt.Errorf("failed to lock project on database: %v", err)
	}
	if projectDoc.PipelineLock {
		return fmt.Errorf("project locked")
	}

	// Start actual deployment step
	deploymentResult := project.DeploymentResult{
		ExecutionIdentifier: deployRequest.Parameters.ExecutionIdentifier,
	}
	if err := deployProject(transportCtx, eventCtx, projectDoc, userDoc.SubscriptionId, deployRequest.Parameters.ExecutionIdentifier); err != nil {
		deploymentResult.Timepoint = time.Now().Unix()
		deploymentResult.Successful = false

		deploymentResultAttributes, sErr := attributevalue.Marshal(&deploymentResult)
		if sErr != nil {
			return fmt.Errorf("failed to serialize deployresult")
		}

		_, uErr := database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
			Table: aws.String(eventCtx.ProjectTable),
			PrimaryKey: map[string]dynamodbtypes.AttributeValue{
				"project_name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.ProjectName},
			},
			AttributeNames: map[string]string{
				"#last_deployment_result": "last_deployment_result",
				"#status":                 "status",
				"#pipeline_lock":          "pipeline_lock",
			},
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":last_deployment_result": deploymentResultAttributes,
				":status":                 &dynamodbtypes.AttributeValueMemberS{Value: fmt.Sprintf("DEPLOYMENT FAILED: %v", err)},
				":pipeline_lock":          &dynamodbtypes.AttributeValueMemberBOOL{Value: false},
			},
			UpdateExpr: aws.String("SET #last_deployment_result = :last_deployment_result, #status = :status, #pipeline_lock = :pipeline_lock"),
		})
		if uErr != nil {
			return fmt.Errorf("failed to update project: %v", uErr)
		}
		return nil
	} else {
		deploymentResult.Timepoint = time.Now().Unix()
		deploymentResult.Successful = true

		deploymentResultAttributes, sErr := attributevalue.Marshal(&deploymentResult)
		if sErr != nil {
			return fmt.Errorf("failed to serialize deployresult")
		}

		_, uErr := database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
			Table: aws.String(eventCtx.ProjectTable),
			PrimaryKey: map[string]dynamodbtypes.AttributeValue{
				"project_name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.ProjectName},
			},
			AttributeNames: map[string]string{
				"#last_deployment_result": "last_deployment_result",
				"#status":                 "status",
				"#pipeline_lock":          "pipeline_lock",
			},
			AttributeValues: map[string]dynamodbtypes.AttributeValue{
				":last_deployment_result": deploymentResultAttributes,
				":status":                 &dynamodbtypes.AttributeValueMemberS{Value: ""},
				":pipeline_lock":          &dynamodbtypes.AttributeValueMemberBOOL{Value: false},
			},
			UpdateExpr: aws.String("SET #last_deployment_result = :last_deployment_result, #status = :status, #pipeline_lock = :pipeline_lock"),
		})
		if uErr != nil {
			return fmt.Errorf("failed to update project: %v", uErr)
		}
	}

	return nil
}

func deployProject(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project, subscriptionId string, execId string) error {
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

	subscriptionDoc, err := database.GetSingle[subscription.Subscription](transportCtx, eventCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(eventCtx.SubscriptionTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: subscriptionId},
		},
		ConditionExpr: aws.String("id = :id"),
	})
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

	pageKeyAttributes, err := attributevalue.Marshal(&buildInformation.PageKeys)
	if err != nil {
		return fmt.Errorf("failed to serialize page key attributes")
	}

	_, err = database.UpdateSingle[project.Project](transportCtx, eventCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(eventCtx.ProjectTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"project_name": &dynamodbtypes.AttributeValueMemberS{Value: projectDoc.ProjectName},
		},
		AttributeNames: map[string]string{
			"#shared_infrastructure": "shared_infrastructure",
			"#prerender_page_keys":   "prerender_page_keys",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":prerender_page_keys": pageKeyAttributes,
		},
		UpdateExpr: aws.String("SET #shared_infrastructure.#prerender_page_keys = :prerender_page_keys"),
	})
	if err != nil {
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
