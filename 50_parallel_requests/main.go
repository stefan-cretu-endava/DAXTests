package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
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

	ret, err := daxClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(tableName),
	})

	if err != nil {
		var re *awshttp.ResponseError
		if errors.As(err, &re) {
			log.Printf("requestID: %s, error: %v", re.ServiceRequestID(), re.Unwrap())
		}
	} else {
		fmt.Println("PutItem succeeded!", *ret)
	}
}

func GetItemWithId(daxClient *dax.Dax, ctx context.Context, id uint) {

	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberN{Value: "12"},
			"sk": &types.AttributeValueMemberN{Value: "1200"},
		},
	}

	_, err := daxClient.GetItem(ctx, input)
	if err != nil {
		var re *awshttp.ResponseError
		if errors.As(err, &re) {
			log.Printf("[GetItemWithId] requestID: %s, error: %v", re.ServiceRequestID(), re.Unwrap())
		} else {
			fmt.Println("[GetItemWithId] err:", err)
		}
	} else {
		fmt.Println("[GetItemWithId] GetItem succeeded:", id)
	}
}

func main() {
	useTLS := flag.Bool("useTLS", false, "Enable usaing TLS cluster")
	requestsNo := flag.Uint("requestsNo", 50, "# of goroutines")
	flag.Parse()

	awsConfigUSEast1 := getAwsConfig(region)
	wg := sync.WaitGroup{}

	if *useTLS {
		daxClientTLS, _ := getSecureDAXClientForTLS(awsConfigUSEast1, ipv4TLSEndpointURL)
		putItem(daxClientTLS)

		ctxCancel, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelFunc()

		//use semaphore to allow max 20 concurrent goroutines
		sem := make(chan struct{}, 20)

		fmt.Printf("Starting %d goroutines to GetItem on 3-node TLS cluster\n", *requestsNo)

		for idx := range *requestsNo {
			wg.Add(1)
			go func(idx uint) {
				defer wg.Done()
				sem <- struct{}{}
				GetItemWithId(daxClientTLS, ctxCancel, idx)
				<-sem
			}(idx)
		}

		wg.Wait()
		fmt.Printf("Finished %d goroutines to GetItem on 3-node TLS cluster.Cancling context and closing client...\n", *requestsNo)

		daxClientTLS.Close()
	} else {
		daxClient3Nodes, _ := getDaxClient(awsConfigUSEast1, ipv4EndpointURL)
		putItem(daxClient3Nodes)

		ctxCancel, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelFunc()

		//use semaphore to allow max 20 concurrent goroutines
		//sem := make(chan struct{}, 20)

		fmt.Printf("Starting %d goroutines to GetItem on 3-node cluster\n", *requestsNo)

		for idx := range *requestsNo {
			wg.Add(1)
			go func(idx uint) {
				defer wg.Done()
				//acquire semaphore, by increasing count of sent empty objects
				//sem <- struct{}{}
				GetItemWithId(daxClient3Nodes, ctxCancel, idx)
				//release semaphore by reading written empty objects
				//<-sem
			}(idx)
		}

		wg.Wait()
		fmt.Printf("Finished %d goroutines to GetItem on 3-node cluster. Cancelling context and closing the client...\n", *requestsNo)

		daxClient3Nodes.Close()
	}
}
