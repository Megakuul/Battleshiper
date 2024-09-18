package event

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-playground/webhooks/v6/github"
	"github.com/megakuul/battleshiper/api/pipeline/routecontext"
	"github.com/megakuul/battleshiper/lib/model/user"
	"go.mongodb.org/mongo-driver/bson"
)

func handleRepoUpdate(transportCtx context.Context, routeCtx routecontext.Context, event github.InstallationRepositoriesPayload) (int, error) {
	userId := event.Installation.Account.ID

	addedRepos := []user.Repository{}
	for _, repo := range event.RepositoriesAdded {
		addedRepos = append(addedRepos, user.Repository{
			Id:       repo.ID,
			Name:     repo.Name,
			FullName: repo.FullName,
		})
	}
	removedRepos := []user.Repository{}
	for _, repo := range event.RepositoriesRemoved {
		addedRepos = append(addedRepos, user.Repository{
			Id:       repo.ID,
			Name:     repo.Name,
			FullName: repo.FullName,
		})
	}

	// MIG: Possible with update item and primary key
	result, err := userCollection.UpdateOne(transportCtx, bson.M{"id": userId}, bson.M{
		"$push": bson.M{
			"github_data.repositories": bson.M{
				"$each": addedRepos,
			},
			"$pull": bson.M{
				"github_data.repositories": bson.M{
					"$in": removedRepos,
				},
			},
		},
	})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to update user on the database")
	}

	if result.MatchedCount < 1 {
		return http.StatusNotFound, fmt.Errorf("user not found")
	}

	return http.StatusOK, nil
}
