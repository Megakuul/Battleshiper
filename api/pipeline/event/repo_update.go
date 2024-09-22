package event

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

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
		Table: routeCtx.UserTable,
		Index: "",
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":id": &dynamodbtypes.AttributeValueMemberS{Value: strconv.Itoa(int(userId))},
		},
		ConditionExpr: "id = :id",
	})
	if err != nil {
		var cErr *dynamodbtypes.ConditionalCheckFailedException
		if ok := errors.As(err, &cErr); ok {
			return http.StatusNotFound, fmt.Errorf("user not found")
		}
		return http.StatusInternalServerError, fmt.Errorf("failed to load user record from database")
	}

	for _, addedRepo := range event.RepositoriesAdded {
		userDoc.Repositories = append(userDoc.Repositories, user.Repository{
			Id:       addedRepo.ID,
			Name:     addedRepo.Name,
			FullName: addedRepo.FullName,
		})
	}

	for _, removedRepo := range event.RepositoriesRemoved {
		for i, installedRepo := range userDoc.Repositories {
			if installedRepo.Id == removedRepo.ID {
				userDoc.Repositories = append(userDoc.Repositories[:i], userDoc.Repositories[i+1:]...)
			}
		}
	}

	repositories, err := attributevalue.Marshal(&userDoc.Repositories)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to serialize repositories")
	}

	_, err = database.UpdateSingle[user.User](transportCtx, routeCtx.DynamoClient, &database.UpdateSingleInput{
		Table: routeCtx.UserTable,
		PrimaryKey: map[string]dynamodbtypes.AttributeValue{
			"id": &dynamodbtypes.AttributeValueMemberS{Value: userDoc.Id},
		},
		AttributeNames: map[string]string{
			"#repositories": "repositories",
		},
		AttributeValues: map[string]dynamodbtypes.AttributeValue{
			":repositories": repositories,
		},
		UpdateExpr: "SET #repositories = :repositories",
	})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to update user: %v", err)
	}

	return http.StatusOK, nil
}
