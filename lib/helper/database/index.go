package database

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Represents a structure for one mongo index.
type Index struct {
	// Sorted structure containing all field names forming the compoundKeys for the index.
	// The first key can be used to efficiently query on its own OR combined with the further keys
	// however queries that only go to the further keys, do not profit from the index performance.
	FieldNames []string
	// SortingOrder used for all field names.
	SortingOrder int
	// Tag identifying wheter the combination of all FieldNames is enforced to be unique.
	Unique bool
}

// SetupIndexes adds the provided struct as indexes to the specified collection.
// indexes expects a struct with fields annotated with `bson:"key"` tags using the sortOrder as value (ASC = 1; DESC = -1;)
// This may be called on every initialization of the mongo client with the indexes required for the current endpoint.
// Mongodb handles applying indexModels idempotently and should therefore only cause a minimal overhead.
func SetupIndexes(collection *mongo.Collection, transportCtx context.Context, indexes []Index) error {
	for _, index := range indexes {
		var compoundKeys bson.D
		for _, fieldName := range index.FieldNames {
			compoundKeys = append(compoundKeys, bson.E{Key: fieldName, Value: index.SortingOrder})
		}

		indexModel := mongo.IndexModel{
			Keys:    compoundKeys,
			Options: options.Index().SetUnique(index.Unique),
		}

		_, err := collection.Indexes().CreateOne(transportCtx, indexModel)
		if err != nil {
			return err
		}
	}
	return nil
}
