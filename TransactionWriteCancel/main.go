package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	//port 8111
	nonExistentCluster   string = "dax://non-existent-cluster.cykcls.dax-clusters.eu-west-1.amazonaws.com"
	ipv4EUEndpointURL    string = "dax://csc-cluster.cykcls.dax-clusters.eu-west-1.amazonaws.com"
	ipv4EndpointURL      string = "dax://csc-ipv4.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"      //3 nodes, r5.large
	ipv6EndpointURL      string = "dax://csc-ipv6.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"      //3 node, r5.large
	dualStackEndpointURL string = "dax://csc-dualstack.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com" //3 nodes, r5.large
	//TLS, port 9111
	ipv4TLSEndpointURL      string = "daxs://csc-tls-ipv4.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"      //3 node, r7i.24xlarge
	dualStackTLSEndpointURL string = "daxs://csc-tls-dualstack.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com" //3 node, r7i.24xlarge

	tableName string = "CSC-DAX-Performance"
	region    string = "us-east-1"
)

type Record struct {
	PK      int64    `dynamodbav:"pk"` // must match key name and type
	SK      int64    `dynamodbav:"sk"` // must match key name and type
	URLs    []string `dynamodbav:"URLs"`
	version string
	test    string
}

// /
// Perform PutItem to store 2 new items, then invoke TransactWriteItems with the same PutItem requests, having a condition
// for a stored attrbute to not exist. When a condition in one of the condition expressions is not met => DynamoDB cancels
// the TransactWriteItems request and issues daxTransactionCanceledFailure.
// If the table name is worng or the table doesn't exist => DynamoDB cancels the TransactWriteItems request and issues daxRequestFailure.
//
// https://github.com/aws/aws-sdk-go-v2/blob/service/dynamodb/v1.26.8/service/dynamodb/types/errors.go#L891
func transactWriteItemsFailure(daxClient *dax.Dax, tableName string) {
	fmt.Println("------------------transactWriteItems----------------------")
	var records []Record = []Record{
		{
			PK: 12,
			SK: 1200,
			URLs: []string{
				"https://example.com/first/link",
				"https://example.com/second/url",
			},
		},
		{
			PK: 13,
			SK: 1300,
			URLs: []string{
				"https://example.com/first/link",
				"https://example.com/second/url",
			},
		},
	}

	item0, err := attributevalue.MarshalMap(records[0])
	if err != nil {
		fmt.Printf("failed to marshal Record, %s", err)
	}

	item1, err := attributevalue.MarshalMap(records[1])
	if err != nil {
		fmt.Printf("failed to marshal Record, %s", err)
	}

	input := &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName:           aws.String(tableName),
					Item:                item0,
					ConditionExpression: aws.String("attribute_not_exists(URLs)"),
				},
			},
			{
				Put: &types.Put{
					TableName:           aws.String(tableName),
					Item:                item1,
					ConditionExpression: aws.String("attribute_not_exists(URLs)"),
				},
			},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityTotal,
	}

	_, err = daxClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item0,
	})

	_, err = daxClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item1,
	})

	_, err = daxClient.TransactWriteItems(context.Background(), input)

	var txErr *types.TransactionCanceledException
	var rnfe *types.ResourceNotFoundException

	if err != nil {

		switch {
		case errors.As(err, &txErr):
			for i, reason := range txErr.CancellationReasons {
				fmt.Printf("[transactWriteItemsFailure] Reason %d: Code=%s, Message=%s\n", i, aws.ToString(reason.Code), aws.ToString(reason.Message))
			}
		case errors.As(err, &rnfe):
			fmt.Println("[transactWriteItemsFailure] Table not found:", rnfe.Error())
		default:
			fmt.Println("[transactWriteItemsFailure] Other error:", err.Error())
		}

	}
}

// /
// There is an ongoing TransactGetItems operation that conflicts with a concurrent PutItem, UpdateItem, DeleteItem or TransactWriteItems request.
// In this case the TransactGetItems operation fails with a TransactionCanceledException => dax geissuesnerates daxTransactionCanceledFailure,
// but not always generated due to non-deterministic goroutines scheduling
// https://github.com/aws/aws-sdk-go-v2/blob/service/dynamodb/v1.26.8/service/dynamodb/types/errors.go#L891
func transactGetItemsFailure(daxClient *dax.Dax, tableName string) {
	fmt.Println("------------------transactGetItems----------------------")
	var records []Record = []Record{
		{
			PK: 909090,
			SK: 909090,
			URLs: []string{
				"https://example.com/first/link",
				"https://example.com/second/url",
			},
		},
		{
			PK: 9090900,
			SK: 9090900,
			URLs: []string{
				"https://example.com/first/link",
				"https://example.com/second/url",
			},
		},
	}

	item0, err := attributevalue.MarshalMap(records[0])
	if err != nil {
		fmt.Printf("failed to marshal Record, %s", err)
	}

	item1, err := attributevalue.MarshalMap(records[1])
	if err != nil {
		fmt.Printf("failed to marshal Record, %s", err)
	}

	tgiInput := &dynamodb.TransactGetItemsInput{
		TransactItems: []types.TransactGetItem{
			{
				Get: &types.Get{
					TableName: aws.String(tableName),
					Key: map[string]types.AttributeValue{
						"pk": item0["pk"],
						"sk": item0["sk"],
					},
				},
			},
			{
				Get: &types.Get{
					TableName: aws.String(tableName),
					Key: map[string]types.AttributeValue{
						"pk": item1["pk"],
						"sk": item1["sk"],
					},
				},
			},
		},
	}

	twiInput := &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName: aws.String(tableName),
					Item:      item0,
				},
			},
			{
				Put: &types.Put{
					TableName: aws.String(tableName),
					Item:      item1,
				},
			},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityTotal,
	}

	twi2Input := &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Delete: &types.Delete{
					TableName: aws.String(tableName),
					Key: map[string]types.AttributeValue{
						"pk": item0["pk"],
						"sk": item0["sk"],
					},
				},
			},
			{
				Delete: &types.Delete{
					TableName: aws.String(tableName),
					Key: map[string]types.AttributeValue{
						"pk": item1["pk"],
						"sk": item1["sk"],
					},
				},
			},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityTotal,
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()

		_, err = daxClient.TransactWriteItems(context.Background(), twiInput)

		var txErr *types.TransactionCanceledException
		var rnfe *types.ResourceNotFoundException

		if err != nil {

			switch {
			case errors.As(err, &txErr):
				for i, reason := range txErr.CancellationReasons {
					fmt.Printf("[transactGetItemsFailure - TransactWriteItems] Reason %d: Code=%s, Message=%s\n", i, aws.ToString(reason.Code), aws.ToString(reason.Message))
				}
			case errors.As(err, &rnfe):
				fmt.Println("[transactGetItemsFailure - TransactWriteItems] Table not found:", rnfe.Error())
			default:
				fmt.Println("[transactGetItemsFailure - TransactWriteItems] Other error:", err.Error())
			}

		}
	}()

	go func() {
		defer wg.Done()

		time.Sleep(10 * time.Microsecond)
		ret, err := daxClient.TransactGetItems(context.Background(), tgiInput)

		var txErr *types.TransactionCanceledException
		var rnfe *types.ResourceNotFoundException

		if err != nil {

			switch {
			case errors.As(err, &txErr):
				for i, reason := range txErr.CancellationReasons {
					fmt.Printf("[transactGetItemsFailure] Reason %d: Code=%s, Message=%s\n", i, aws.ToString(reason.Code), aws.ToString(reason.Message))
				}
			case errors.As(err, &rnfe):
				fmt.Println("[transactGetItemsFailure] Table not found:", rnfe.Error())
			default:
				fmt.Println("[transactGetItemsFailure] Other error:", err.Error())
			}

		} else {
			fmt.Println("TransactGetItems:", ret)
		}
	}()

	go func() {
		defer wg.Done()

		time.Sleep(1 * time.Millisecond)

		_, err = daxClient.TransactWriteItems(context.Background(), twi2Input)

		var txErr *types.TransactionCanceledException
		var rnfe *types.ResourceNotFoundException

		if err != nil {

			switch {
			case errors.As(err, &txErr):
				for i, reason := range txErr.CancellationReasons {
					fmt.Printf("[transactGetItemsFailure - TransactWriteItems] Reason %d: Code=%s, Message=%s\n", i, aws.ToString(reason.Code), aws.ToString(reason.Message))
				}
			case errors.As(err, &rnfe):
				fmt.Println("[transactGetItemsFailure - TransactWriteItems] Table not found:", rnfe.Error())
			default:
				fmt.Println("[transactGetItemsFailure - TransactWriteItems] Other error:", err.Error())
			}

		}
	}()

	wg.Wait()
}

func main() {

	awsConfigUSEast1 := getAwsConfig(region)
	daxClient3Nodes, _ := getDaxClient(awsConfigUSEast1, ipv4EndpointURL)
	defer daxClient3Nodes.Close()

	// Generate daxTransactionCanceledFailure, by attempting to perform TransactWriteItems, embedding multiple PutItems with conditions that fail
	transactWriteItemsFailure(daxClient3Nodes, tableName)
	// Generate daxTransactionCanceledFailure, by attempting to perform TransactGetItems concurrently with 2 TransactWriteItems => not always generated
	transactGetItemsFailure(daxClient3Nodes, tableName)
	// Generate daxRequestFailure, by attempting to perform TransactWriteItems and TransactGetItems on a non-existent-table
	transactWriteItemsFailure(daxClient3Nodes, "NonExistentTable")
	transactGetItemsFailure(daxClient3Nodes, "NonExistentTable")
}
