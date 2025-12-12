package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
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

func DeleteRecordItem(daxClient *dax.Dax, inputRecord Record) {
	fmt.Println("------------------DeleteRecordItem----------------------")
	key := map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberN{Value: strconv.FormatInt(inputRecord.PK, 10)},
		"sk": &types.AttributeValueMemberN{Value: strconv.FormatInt(inputRecord.SK, 10)},
	}

	serverReply, err := daxClient.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key:       key,
	})

	if err != nil {
		fmt.Println("[DeleteRecordItem] Failed: ", err.Error())
	} else if serverReply != nil {
		fmt.Println("[DeleteRecordItem] Succeeded!: ", *serverReply)
	}
}

func PutRecordItem(daxClient *dax.Dax, inputRecord Record) {
	fmt.Println("------------------PutRecordItem----------------------")
	item, err := attributevalue.MarshalMap(inputRecord)
	if err != nil {
		fmt.Printf("[PutRecordItem] Failed to marshal Record, %s", err)
	}

	serverReply, err := daxClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(tableName),
	})

	if err != nil {
		fmt.Printf("[PutRecordItem] Failed: %+v\n", err)

		var respErr smithy.APIError
		if errors.As(err, &respErr) {
			fmt.Printf("[PutRecordItem] Error Code: %s\n", respErr.ErrorCode())
			fmt.Printf("[PutRecordItem] Error Message: %s\n", respErr.ErrorMessage())
			fmt.Println("[PutRecordItem] Debug stack:", string(debug.Stack()))
		}

	} else if serverReply != nil {
		fmt.Println("[PutRecordItem] Succeeded! Server reply: ", serverReply)
	}
}

func UpdateRecordItem(daxClient *dax.Dax, inputRecord Record) {
	fmt.Println("------------------UpdateRecordItem----------------------")

	// item, err := attributevalue.MarshalMap(inputRecord)
	// if err != nil {
	// 	fmt.Printf("failed to marshal Record, %s", err)
	// }

	key := map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberN{Value: strconv.FormatInt(inputRecord.PK, 10)},
		"sk": &types.AttributeValueMemberN{Value: strconv.FormatInt(inputRecord.SK, 10)},
	}

	newURLs := []string{"code_with_changes", "first_proposed_solution_impl"}

	serverReply, err := daxClient.UpdateItem(context.Background(), &dynamodb.UpdateItemInput{
		TableName:                aws.String(tableName),
		Key:                      key,
		UpdateExpression:         aws.String("SET #urls = :urls"),
		ExpressionAttributeNames: map[string]string{"#urls": "URLs"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":urls": &types.AttributeValueMemberL{
				Value: func() []types.AttributeValue {
					out := make([]types.AttributeValue, 0, len(newURLs))
					for _, u := range newURLs {
						out = append(out, &types.AttributeValueMemberS{Value: u})
					}
					return out
				}(),
			},
		},
		ReturnValues: types.ReturnValueUpdatedNew, // return the new image (updated attributes)
	})

	var rnfe *types.ResourceNotFoundException

	if err != nil {

		switch {
		case errors.As(err, &rnfe):
			fmt.Println("[UpdateRecordItem] Resource not found:", rnfe.Error())
		default:
			fmt.Println("[UpdateRecordItem] Other error:", err.Error())
		}
	} else if serverReply != nil {
		fmt.Println("[UpdateRecordItem] Succeeded! Server reply:", *serverReply)
	}
}

func TransactWriteRecordItem(daxClient *dax.Dax, records []Record) {
	fmt.Println("------------------TransactWriteRecordItem----------------------")

	item1, err := attributevalue.MarshalMap(records[1])
	if err != nil {
		fmt.Printf("failed to marshal Record, %s", err)
	}

	key0 := map[string]types.AttributeValue{
		"pk": &types.AttributeValueMemberN{Value: strconv.FormatInt(records[0].PK, 10)},
		"sk": &types.AttributeValueMemberN{Value: strconv.FormatInt(records[0].SK, 10)},
	}

	//newURLs := []string{"code_with_changes", "first_proposed_solution_impl"}

	input := &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Delete: &types.Delete{
					TableName: aws.String(tableName),
					Key:       key0,
				},
			},
			{
				Put: &types.Put{
					TableName: aws.String(tableName),
					Item:      item1,
				},
			},
			// {
			// 	Update: &types.Update{
			// 		TableName:                aws.String(tableName),
			// 		Key:                      key,
			// 		UpdateExpression:         aws.String("SET #urls = :urls"),
			// 		ExpressionAttributeNames: map[string]string{"#urls": "URLs"},
			// 		ExpressionAttributeValues: map[string]types.AttributeValue{
			// 			":urls": &types.AttributeValueMemberL{
			// 				Value: func() []types.AttributeValue {
			// 					out := make([]types.AttributeValue, 0, len(newURLs))
			// 					for _, u := range newURLs {
			// 						out = append(out, &types.AttributeValueMemberS{Value: u})
			// 					}
			// 					return out
			// 				}(),
			// 			},
			// 		},
			// 	},
			// },
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityTotal,
	}

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

func main() {
	userRequestTimeout := flag.Float64("requestTimeout", 60000, "Value in milliseconds for requestTimeout")
	useTransactWrite := flag.Bool("useTransactWrite", false, "Use TransactWriteCancel")
	flag.Parse()

	var requestTimeoutSeconds = time.Duration(*userRequestTimeout) * time.Millisecond
	fmt.Println("Creating DAX client with request timeout:", requestTimeoutSeconds.String())

	awsConfigUSEast1 := getAwsConfig(region)
	daxClient3Nodes, _ := getDaxClientWithRequesTimeout(awsConfigUSEast1, ipv4EndpointURL, requestTimeoutSeconds)
	defer daxClient3Nodes.Close()

	if *useTransactWrite {
		TransactWriteRecordItem(daxClient3Nodes, records)
	} else {
		DeleteRecordItem(daxClient3Nodes, records[0])
		PutRecordItem(daxClient3Nodes, records[0])
	}
	// UpdateRecordItem(daxClient3Nodes, records[0])
}
