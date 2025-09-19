package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

func isServerError(err error) bool {
	// You can expand this to match specific DAX error types or use string matching
	return strings.Contains(err.Error(), "500") ||
		strings.Contains(err.Error(), "InternalServerError") ||
		strings.Contains(err.Error(), "ServiceUnavailable")
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
		var re *awshttp.ResponseError
		if errors.As(err, &re) {
			log.Printf("requestID: %s, error: %v", re.ServiceRequestID(), re.Unwrap())
		}
	} else {
		fmt.Println("PutItem succeeded unexpectedly")
	}

	// if err != nil {
	// 	var opErr *smithy.OperationError
	// 	if errors.As(err, &opErr) {
	// 		var apiErr smithy.APIError
	// 		if errors.As(opErr.Unwrap(), &apiErr) {
	// 			fmt.Printf("API Error: %s\nMessage: %s\nRequest ID: %s\n",
	// 				apiErr.ErrorCode(), apiErr.ErrorMessage(), apiErr.ErrorFault().String())
	// 		} else {
	// 			fmt.Printf("Operation error: %v\n", opErr)
	// 		}
	// 	} else {
	// 		fmt.Printf("Unexpected error: %v\n", err)
	// 	}
	// 	fmt.Println("Debug stack:", string(debug.Stack()))
	// }
}

func GetItemWithShortContext(daxClient *dax.Dax) {
	// Create a context with an extremely short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	input := &dynamodb.GetItemInput{
		TableName: aws.String("your_valid_table_name"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: "123"},
		},
	}

	_, err := daxClient.GetItem(ctx, input)
	if err != nil {
		var re *awshttp.ResponseError
		if errors.As(err, &re) {
			log.Printf("requestID: %s, error: %v", re.ServiceRequestID(), re.Unwrap())
		}
	} else {
		fmt.Println("GetItem succeeded unexpectedly")
	}
}

func main() {
	reboot := flag.Bool("reboot", false, "Enable cluster reboot")
	flag.Parse()
	awsConfigUSEast1 := getAwsConfig("us-east-1")

	daxClient, _ := getSecureDAXClientForTLS(awsConfigUSEast1, ipv4TLSEndpointURL)

	// Step 1: Restart all nodes in the cluster using AWS CLI
	clusterName := "csc-tls-ipv4"
	if *reboot {
		fmt.Println("Restarting all nodes in cluster:", clusterName)
		// restartCmd := fmt.Sprintf(`bash -c '
		// for NODE_ID in $(aws dax describe-cluster --endpoint-url https://elasticache-devo.us-east-1.amazonaws.com --no-verify-ssl --cluster-name %s --region %s --query "Cluster.Nodes[*].NodeId" --output text); do
		// echo "Rebooting node: $NODE_ID"
		// aws dax reboot-node --endpoint-url https://elasticache-devo.us-east-1.amazonaws.com --no-verify-ssl --cluster-name %s --node-id $NODE_ID --region %s
		// done'`, clusterName, region, clusterName, region)

		restartCmd := "aws dax reboot-node --endpoint-url https://elasticache-devo.us-east-1.amazonaws.com --no-verify-ssl --cluster-name csc-tls-ipv4 --region us-east-1 --node-id csc-tls-ipv4-a"
		cmd := exec.Command("bash", "-c", restartCmd)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to restart nodes: %v\nOutput: %s\n", err, string(output))
			return
		}
		var waitTime time.Duration = 25
		fmt.Println("Cluster reboot initiated. Waiting for ", int(waitTime), " seconds")
		time.Sleep(waitTime * time.Second)
	}
	// Step 2: Proceed with DAX PutItem
	const maxRetries int = 10
	fmt.Println("Retry PutItem for ", maxRetries, " times")
	// Retry loop
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Attempt %d:\n", attempt)
		putItem(daxClient)
		//time.Sleep(time.Duration(attempt) * time.Second) // Optional backoff
	}

	//GetItemWithShortContext(daxClient)

	daxClient.Close()
}
