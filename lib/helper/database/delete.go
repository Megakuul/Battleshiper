package database

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DeleteSingleInput struct {
	Table      string
	PrimaryKey map[string]dynamodbtypes.AttributeValue
}

// DeleteSingle deletes a single item from the database.
func DeleteSingle[T any](transportCtx context.Context, dynamoClient *dynamodb.Client, input *DeleteSingleInput) error {
	_, err := dynamoClient.DeleteItem(transportCtx, &dynamodb.DeleteItemInput{
		TableName: aws.String(input.Table),
		Key:       input.PrimaryKey,
	})
	if err != nil {
		return err
	}

	return nil
}
