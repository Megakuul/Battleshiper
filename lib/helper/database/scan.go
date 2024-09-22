package database

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type ScanManyInput struct {
	Table string
	Limit int32
}

// ScanMany reads all items from the database. Only use this on very very small datasets.
// Set the limit to '-1' to fetch all items.
func ScanMany[T any](transportCtx context.Context, dynamoClient *dynamodb.Client, input *ScanManyInput) ([]T, error) {
	output, err := dynamoClient.Scan(transportCtx, &dynamodb.ScanInput{
		TableName: aws.String(input.Table),
		Limit:     aws.Int32(input.Limit),
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
