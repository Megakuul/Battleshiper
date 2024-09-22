package database

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type GetSingleInput struct {
	Table           string
	Index           string
	AttributeValues map[string]dynamodbtypes.AttributeValue
	ConditionExpr   string
}

// GetSingle fetches a single item from the database and tries to deserialize it into the provided struct type.
// Set the index to "" to query the main table.
func GetSingle[T any](transportCtx context.Context, dynamoClient *dynamodb.Client, input *GetSingleInput) (*T, error) {
	output, err := dynamoClient.Query(transportCtx, &dynamodb.QueryInput{
		IndexName:                 aws.String(input.Index),
		TableName:                 aws.String(input.Table),
		ExpressionAttributeValues: input.AttributeValues,
		KeyConditionExpression:    aws.String(input.ConditionExpr),
		Limit:                     aws.Int32(1),
	})
	if err != nil {
		return nil, err
	}

	var outputStructureList []T
	err = attributevalue.UnmarshalListOfMaps(output.Items, &outputStructureList)
	if err != nil {
		return nil, fmt.Errorf("cannot deserialize database items")
	}

	if len(outputStructureList) < 1 {
		return nil, fmt.Errorf("item not found")
	}

	return &outputStructureList[0], nil
}

type GetManyInput struct {
	Table           string
	Index           string
	AttributeValues map[string]dynamodbtypes.AttributeValue
	ConditionExpr   string
	Limit           int32
}

// GetMany fetches items from the database and tries to deserialize it into a list of the provided struct type.
// Set the index to "" to query the main table.
// Set the limit to '-1' to fetch all items.
func GetMany[T any](transportCtx context.Context, dynamoClient *dynamodb.Client, input *GetManyInput) ([]T, error) {
	output, err := dynamoClient.Query(transportCtx, &dynamodb.QueryInput{
		TableName:                 aws.String(input.Table),
		IndexName:                 aws.String(input.Index),
		ExpressionAttributeValues: input.AttributeValues,
		KeyConditionExpression:    aws.String(input.ConditionExpr),
		Limit:                     aws.Int32(input.Limit),
	})
	if err != nil {
		return nil, err
	}

	var outputStructureList []T
	err = attributevalue.UnmarshalListOfMaps(output.Items, &outputStructureList)
	if err != nil {
		return nil, fmt.Errorf("cannot deserialize database items")
	}

	return outputStructureList, nil
}
