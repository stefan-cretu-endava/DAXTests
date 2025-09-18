package main

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/smithy-go"
)

const (
	//port 8111
	ipv4EUEndpointURL    string = "dax://csc-cluster.cykcls.dax-clusters.eu-west-1.amazonaws.com"
	ipv4EndpointURL      string = "dax://csc-ipv4-ubuntu.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com" //3 nodes, r5.large
	ipv6EndpointURL      string = "dax://csc-ipv6.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"        //1 node, r5.large
	dualStackEndpointURL string = "dax://csc-dualstack.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"   //3 nodes, r5.large
	//TLS, port 9111
	ipv4TLSEndpointURL      string = "daxs://csc-tls-ipv4.6lzwui.alpha-dax-clusters.us-east-1.amazonaws.com"      //1 node, r7i.24xlarge
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

func putItem(daxClient *dax.Dax) {

	r := Record{
		PK: 12,
		SK: 1200,
		URLs: []string{
			"https://example.com/first/link",
			"https://example.com/second/url",
		},
	}
	item, err := attributevalue.MarshalMap(r)
	if err != nil {
		fmt.Printf("failed to marshal Record, %s", err)
	}

	_, err = daxClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(tableName),
	})

	if err != nil {
		fmt.Printf("PutItem failed: %+v\n", err)

		var respErr smithy.APIError
		if errors.As(err, &respErr) {
			fmt.Printf("Error Code: %s\n", respErr.ErrorCode())
			fmt.Printf("Error Message: %s\n", respErr.ErrorMessage())
			fmt.Println("Debug stack:", string(debug.Stack()))
		}

	} else {
		fmt.Println("PutItem succeeded")
	}
}

func main() {

	awsConfigUSEast1 := getAwsConfig("us-east-1")

	daxClient, err := getDaxClient(awsConfigUSEast1, ipv4EndpointURL)
	if err != nil {
		panic(err)
	}

	UpdateItem_ReturnValuesOnConditionCheckFailure(daxClient, true)
	UpdateItem_ReturnValuesOnConditionCheckFailure(daxClient, false)
	// PutItem_ReturnValuesOnConditionCheckFailure(daxClient, true)
	// PutItem_ReturnValuesOnConditionCheckFailure(daxClient, false)
	// DeleteItem_ReturnValuesOnConditionCheckFailure(daxClient, true)
	// DeleteItem_ReturnValuesOnConditionCheckFailure(daxClient, false)
	daxClient.Close()
}
