package event

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/webhooks/v6/github"
	"github.com/megakuul/battleshiper/api/pipeline/routecontext"
	"github.com/megakuul/battleshiper/lib/helper/database"
	"github.com/megakuul/battleshiper/lib/model/user"
)

func handleRepoUpdate(transportCtx context.Context, routeCtx routecontext.Context, event github.InstallationRepositoriesPayload) (int, error) {
	userId := event.Installation.Account.ID

	userDoc, err := database.GetSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.GetSingleInput{
		Table: aws.String(routeCtx.UserTable),
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: strconv.Itoa(int(userId))},
		},
		ConditionExpr: aws.String("id = :id"),
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return http.StatusNotFound, fmt.Errorf("user not found")
		}
		logger.Printf("failed to load user record from database: %v\n", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
	}

	for _, repo := range event.RepositoriesAdded {
		userDoc.Repositories[repo.ID] = user.Repository{
			Id:       repo.ID,
			Name:     repo.Name,
			FullName: repo.FullName,
		}
	}
	for _, repo := range event.RepositoriesRemoved {
		delete(userDoc.Repositories, repo.ID)
	}

	repositories, err := attributevalue.Marshal(&userDoc.Repositories)
	if err != nil {
		logger.Printf("failed to serialize repositories: %v\n", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to serialize repositories")
	}

	_, err = database.UpdateSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
		Table: aws.String(routeCtx.UserTable),
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: userDoc.Id},
		},
		AttributeNames: map[string]string{
			"#repositories": "repositories",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":repositories": repositories,
		},
		UpdateExpr: aws.String("SET #repositories = :repositories"),
	})
	if err != nil {
		logger.Printf("failed to update user: %v\n", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to update user")
	}

	return http.StatusOK, nil
}
