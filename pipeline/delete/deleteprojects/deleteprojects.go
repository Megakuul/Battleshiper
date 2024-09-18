package deleteprojects

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/megakuul/battleshiper/lib/model/project"
	"github.com/megakuul/battleshiper/pipeline/delete/eventcontext"
	"go.mongodb.org/mongo-driver/bson"
)

func HandleDeleteProjects(eventCtx eventcontext.Context) func(context.Context) error {
	return func(ctx context.Context) error {
		err := runHandleDeleteProjects(ctx, eventCtx)
		if err != nil {
			log.Printf("ERROR DELETEPROJECTS: %v\n", err)
			return err
		}
		return nil
	}
}

func runHandleDeleteProjects(transportCtx context.Context, eventCtx eventcontext.Context) error {
	// MIG: Possible with scan and filter to deleted
	projectCursor, err := projectCollection.Find(transportCtx, bson.D{
		{Key: "deleted", Value: true},
	})
	if err != nil {
		return fmt.Errorf("failed to fetch projects from database")
	}
	defer projectCursor.Close(transportCtx)

	var (
		deletionErrors     = []error{}
		deletionErrorsLock sync.Mutex
		deletionErrorsChan = make(chan error, 10)
		deletionWaitGroup  sync.WaitGroup
	)

	go func() {
		for err := range deletionErrorsChan {
			deletionErrorsLock.Lock()
			deletionErrors = append(deletionErrors, err)
			deletionErrorsLock.Unlock()
		}
	}()

	for projectCursor.Next(transportCtx) {
		projectDoc := &project.Project{}
		if err := projectCursor.Decode(&projectDoc); err != nil {
			deletionErrorsChan <- fmt.Errorf("[undefined]: %v", err)
			continue
		}
		if projectDoc.PipelineLock {
			deletionErrorsChan <- fmt.Errorf("[%s]: project pipeline is locked", projectDoc.Name)
			continue
		}

		deletionWaitGroup.Add(1)
		go func() {
			defer deletionWaitGroup.Done()
			if err := deleteProject(transportCtx, eventCtx, projectDoc); err != nil {
				deletionErrorsChan <- fmt.Errorf("[%s]: %v", projectDoc.Name, err)

				// MIG: Possible with query item and primary key
				result, err := projectCollection.UpdateByID(transportCtx, projectDoc.MongoID, bson.M{
					"$set": bson.M{
						"status": "DELETION FAILED: Contact an Administrator",
					},
				})
				if err != nil && result.MatchedCount < 1 {
					deletionErrorsChan <- fmt.Errorf("[%s]: failed to update project status", projectDoc.Name)
				}
			}
		}()
	}
	if projectCursor.Err() != nil {
		deletionErrorsChan <- fmt.Errorf("[undefined]: %s", projectCursor.Err())
	}
	deletionWaitGroup.Wait()
	close(deletionErrorsChan)

	deletionErrorsLock.Lock()
	defer deletionErrorsLock.Unlock()
	if len(deletionErrors) > 0 {
		deletionErrorMessage := "at least one error occured while deleting projects:\n"
		for _, delErr := range deletionErrors {
			deletionErrorMessage += fmt.Sprintf(" - %v\n", delErr)
		}
		return fmt.Errorf("%s", deletionErrorMessage)
	}
	return nil
}

func deleteProject(transportCtx context.Context, eventCtx eventcontext.Context, projectDoc *project.Project) error {
	if err := deleteStaticAssets(transportCtx, eventCtx, projectDoc); err != nil {
		return err
	}

	if err := deleteStaticPageKeys(transportCtx, eventCtx, projectDoc.SharedInfrastructure.PrerenderPageKeys); err != nil {
		return err
	}

	if err := deleteStack(transportCtx, eventCtx, projectDoc); err != nil {
		return err
	}

	// MIG: Possible with delete item and primary key
	_, err := projectCollection.DeleteOne(transportCtx, bson.M{"_id": projectDoc.MongoID})
	if err != nil {
		return fmt.Errorf("failed to delete project from database")
	}

	return nil
}
