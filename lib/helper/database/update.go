package database

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type UpdateSingleInput struct {
	Table           string
	PrimaryKey      map[string]dynamodbtypes.AttributeValue
	Upsert          bool
	ReturnOld       bool
	ConditionExpr   string
	AttributeNames  map[string]string
	AttributeValues map[string]dynamodbtypes.AttributeValue
	UpdateExpr      string
}

// UpdateSingle updates a single item on the database.
func UpdateSingle[T any](transportCtx context.Context, dynamoClient *dynamodb.Client, input *UpdateSingleInput) (*T, error) {
	conditionExpr := input.ConditionExpr
	// If upsert is disabled, the primary key attributes must be present to update.
	if !input.Upsert {
		for key := range input.PrimaryKey {
			if conditionExpr == "" {
				conditionExpr = fmt.Sprintf("attribute_exists(%s)", key)
			} else {
				conditionExpr = fmt.Sprintf("%s AND attribute_exists(%s)", conditionExpr, key)
			}
		}
	}

	returnValue := dynamodbtypes.ReturnValueAllNew
	if input.ReturnOld {
		returnValue = dynamodbtypes.ReturnValueAllOld
	}

	output, err := dynamoClient.UpdateItem(transportCtx, &dynamodb.UpdateItemInput{
		TableName:                 aws.String(input.Table),
		Key:                       input.PrimaryKey,
		ExpressionAttributeNames:  input.AttributeNames,
		ExpressionAttributeValues: input.AttributeValues,
		UpdateExpression:          aws.String(input.UpdateExpr),
		ConditionExpression:       aws.String(conditionExpr),
		ReturnValues:              returnValue,
	})
	if err != nil {
		return nil, err
	}

	var outputStructure T
	err = attributevalue.UnmarshalMap(output.Attributes, &outputStructure)
	if err != nil {
		return nil, fmt.Errorf("cannot deserialize database item")
	}

	return &outputStructure, nil
}
