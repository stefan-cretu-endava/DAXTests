package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	//port 8111
	ipv4EUEndpointURL    string = "dax://csc-cluster.cykcls.dax-clusters.eu-west-1.amazonaws.com"
	ipv4EndpointURL      string = "dax://csc-ipv4-ubuntu.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com" //3 nodes, r5.large
	ipv6EndpointURL      string = "dax://csc-ipv6.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"        //1 node, r5.large
	dualStackEndpointURL string = "dax://csc-dualstack.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"   //2 nodes, r5.large
	//TLS, port 9111
	ipv4TLSEndpointURL      string = "daxs://csc-tls-ipv4.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"      //3 node, r7i.24xlarge
	dualStackTLSEndpointURL string = "daxs://csc-tls-dualstack.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com" //1 node, r7i.24xlarge

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

func transactWriteItems(daxClient *dax.Dax) {
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
	if errors.As(err, &txErr) {
		for i, reason := range txErr.CancellationReasons {
			fmt.Printf("Reason %d: Code=%s, Message=%s\n", i, aws.ToString(reason.Code), aws.ToString(reason.Message))
		}
	} else {
		fmt.Printf("Unexpected error: %v\n", err)
	}
}

func main() {

	awsConfigUSEast1 := getAwsConfig(region)
	daxClient3Nodes, _ := getDaxClient(awsConfigUSEast1, ipv4EndpointURL)

	defer daxClient3Nodes.Close()

	transactWriteItems(daxClient3Nodes)
}
