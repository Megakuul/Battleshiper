package database

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type PutSingleInput[T any] struct {
	Table                   *string
	Item                    T
	ProtectionAttributeName *string
}

// PutSingle inserts a single item to the database.
// If you want to ensure no item is overwritten, you can set the ProtectionAttributeName
// to an attribute key that must NOT alread exist (usually the partition key is used for this).
func PutSingle[T any](transportCtx context.Context, dynamoClient *dynamodb.Client, input *PutSingleInput[T]) error {
	var conditionExpr *string = nil
	if input.ProtectionAttributeName != nil {
		conditionExpr = aws.String(fmt.Sprintf("attribute_not_exists(%s)", *input.ProtectionAttributeName))
	}

	inputStructure, err := attributevalue.MarshalMap(&input.Item)
	if err != nil || len(inputStructure) < 1 {
		return fmt.Errorf("cannot serialize input item")
	}

	_, err = dynamoClient.PutItem(transportCtx, &dynamodb.PutItemInput{
		TableName:           input.Table,
		Item:                inputStructure,
		ConditionExpression: conditionExpr,
		ReturnValues:        dynamodbtypes.ReturnValueNone,
	})
	if err != nil {
		return err
	}

	return nil
}
