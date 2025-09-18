package main

import (
	"context"
	"errors"
	"log"
	"runtime/debug"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// UpdateItem
func UpdateItem_ReturnValuesOnConditionCheckFailure(daxClient *dax.Dax, useDynamoDB bool) {
	item := MakeItem(
		uint(12),
		uint(1200),
		4096,
		6,
		false,
	)

	key := map[string]types.AttributeValue{
		FieldPK: item[FieldPK],
		FieldSK: item[FieldSK],
	}
	updateItemInputQuery := &dynamodb.UpdateItemInput{
		TableName:           aws.String(tableName),
		Key:                 key,
		UpdateExpression:    aws.String("SET test = :T, version = :VN"),
		ConditionExpression: aws.String("#V=:V"),
		ExpressionAttributeNames: map[string]string{
			"#V": "version",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":T":  &types.AttributeValueMemberS{Value: "ok"},
			":V":  &types.AttributeValueMemberN{Value: "1"},
			":VN": &types.AttributeValueMemberN{Value: "2"},
		},
		ReturnValues:                        types.ReturnValueAllNew,
		ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureAllOld,
	}

	if useDynamoDB {
		log.Println("[UpdateItemDynamoDB_ReturnValuesOnConditionCheckFailure]: before dynamodb.UpdateItemInput")

		retData := updateItemInputQuery

		log.Println("[UpdateItemDynamoDB_ReturnValuesOnConditionCheckFailure]: DynamoDB retData *dynamodb.UpdateItemInput:", retData)
	} else {
		log.Println("[UpdateItemDynamoDB_ReturnValuesOnConditionCheckFailure]: before dax.UpdateItemInput")

		retData, err := daxClient.UpdateItem(context.Background(), updateItemInputQuery)

		if err != nil {
			var cfeErr *types.ConditionalCheckFailedException
			if errors.As(err, &cfeErr) {
				log.Println("[UpdateItemDAX_ReturnValuesOnConditionCheckFailure]: Old item on ConditionalCheckFailedException", cfeErr.Item)
				log.Println("[UpdateItemDAX_ReturnValuesOnConditionCheckFailure]: DAX retData *dynamodb.UpdateItemOutput", retData)
			}
			log.Fatal(err)
		}
		log.Println("[UpdateItemDAX_ReturnValuesOnConditionCheckFailure]: DAX retData *dynamodb.UpdateItemOutput:", retData)
	}
	log.Println("[UpdateItemDynamoDB_ReturnValuesOnConditionCheckFailure]: Debug stack:", string(debug.Stack()))
}

// PutItem
func PutItem_ReturnValuesOnConditionCheckFailure(daxClient *dax.Dax, useDynamoDB bool) {
	item := MakeItem(
		uint(13),
		uint(1300),
		512,
		1,
		false,
	)

	// key := map[string]types.AttributeValue{
	// 	FieldPK: item[FieldPK],
	// 	FieldSK: item[FieldSK],
	// }
	putItemInputQuery := &dynamodb.PutItemInput{
		TableName:                           aws.String(tableName),
		Item:                                item,
		ConditionExpression:                 aws.String("attribute_not_exists(version)"),
		ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureAllOld,
	}

	if useDynamoDB {
		log.Println("[PutItem_ReturnValuesOnConditionCheckFailure]: before dynamodb.PutItemInput")

		retData := putItemInputQuery

		log.Println("[PutItem_ReturnValuesOnConditionCheckFailure]: dynamodb retData *dynamodb.PutItemInput ", retData)
	} else {
		log.Println("[UpdateItemDynamoDB_ReturnValuesOnConditionCheckFailure]: before dax.PutItemInput")

		retData, err := daxClient.PutItem(context.Background(), putItemInputQuery)

		if err != nil {
			var cfeErr *types.ConditionalCheckFailedException
			if errors.As(err, &cfeErr) {
				log.Println("[PutItem_ReturnValuesOnConditionCheckFailure]: Old item on ConditionalCheckFailedException", cfeErr.Item)
				log.Println("[PutItem_ReturnValuesOnConditionCheckFailure]: DAX retData *dynamodb.PutItemOutput", retData)
			}
			log.Fatal(err)
		}
		log.Println("[PutItemReturnValuesOnConditionCheckFailure]: DAX retData *dynamodb.PutItemOutput", retData)

	}

	log.Println("[PutItem_ReturnValuesOnConditionCheckFailure]: Debug stack:", string(debug.Stack()))
}

// DeleteItem
func DeleteItem_ReturnValuesOnConditionCheckFailure(daxClient *dax.Dax, useDynamoDB bool) {
	item := MakeItem(
		uint(13),
		uint(1300),
		512,
		1,
		false,
	)

	key := map[string]types.AttributeValue{
		FieldPK: item[FieldPK],
		FieldSK: item[FieldSK],
	}

	deleteItemInputQuery := &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),

		Key:                                 key,
		ConditionExpression:                 aws.String("attribute_not_exists(version)"),
		ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureAllOld,
	}

	if useDynamoDB {
		log.Println("[DeleteItem_ReturnValuesOnConditionCheckFailure]: before dynamodb.DeleteItemInput")

		retData := deleteItemInputQuery

		log.Println("[DeleteItem_ReturnValuesOnConditionCheckFailure]: dynamodb retData *dynamodb.DeleteItemInput ", retData)
	} else {
		log.Println("[DeleteItemDynamoDB_ReturnValuesOnConditionCheckFailure]: before dax.DeleteItemInput")

		retData, err := daxClient.DeleteItem(context.Background(), deleteItemInputQuery)

		if err != nil {
			var cfeErr *types.ConditionalCheckFailedException
			if errors.As(err, &cfeErr) {
				log.Println("[DeleteItem_ReturnValuesOnConditionCheckFailure]: Old item on ConditionalCheckFailedException", cfeErr.Item)
				log.Println("[DeleteItem_ReturnValuesOnConditionCheckFailure]: DAX retData *dynamodb.DeleteItemOutput", retData)
			}
			log.Fatal(err)
		}
		log.Println("[DeleteItem_ReturnValuesOnConditionCheckFailure]: DAX retData *dynamodb.DeleteItemOutput", retData)

	}

	log.Println("[DeleteItem_ReturnValuesOnConditionCheckFailure]: Debug stack:", string(debug.Stack()))
}
