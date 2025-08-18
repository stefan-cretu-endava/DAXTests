package main

import (
	"context"
	"log"
	rand "math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	daxsvc "github.com/aws/aws-sdk-go-v2/service/dax"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DataLoader struct {
	daxClient      *dax.Dax
	dynamoDBClient *dynamodb.Client
	adminClient    *daxsvc.Client

	// pks
	firstPkBatchWriteItemKey atomic.Int64
	firstPkGetItemKey        atomic.Int64
	firstPkPutItemKey        atomic.Int64
	//private AtomicInteger firstPkBatchWriteItemKey = new AtomicInteger(10000000);
	//private AtomicInteger firstPkGetItemKey = new AtomicInteger(60000);
	//private AtomicInteger firstPkPutItemKey = new AtomicInteger(1000000);

	// key counters
	keyCounter             KeyCounter[int]
	keyCounterPutItem      KeyCounter[int]
	keyCounterBatchPutItem KeyCounter[int]

	// cache miss
	randomGetItemMiss      *Random[int]
	randomBatchGetItemMiss *Random[int]
	randomQueryMiss        *Random[int]
	// cache hit
	randomGetItemHit      *Random[int]
	randomBatchGetItemHit *Random[int]
	randomQueryHit        *Random[int]

	metricsService Metrics
}

func (dl *DataLoader) loadSampleData(startPartitionKey int, endPartitionKey int, nrOfSortingKey int, itemSizeKB int, throughDax bool) {
	var ch = make(chan map[string]types.AttributeValue, (endPartitionKey-startPartitionKey+1)*nrOfSortingKey)

	wg := sync.WaitGroup{}

	// generate the items in a buffer
	for pk := range rangeClosed(startPartitionKey, endPartitionKey) {
		for sk := range rangeClosed(1, nrOfSortingKey) {
			item := MakeItem(
				uint(pk),
				uint(sk),
				uint(itemSizeKB),
				6,
				false,
			)
			ch <- item
		}
	}

	// determine the client
	var client dax.DynamoDBAPI
	if throughDax {
		client = dl.daxClient
	} else {
		client = dl.dynamoDBClient
	}

	// start consumer threads
	for i := range 10 {
		wg.Add(1)
		go func(id int, cl dax.DynamoDBAPI) {
			defer wg.Done()

			processed := 0

			for {
				items := []types.WriteRequest{}
				canRun := true
				for len(items) < 25 && canRun {
					select {
					case item, ok := <-ch:
						if ok && item != nil {
							items = append(items, types.WriteRequest{
								PutRequest: &types.PutRequest{
									Item: item,
								},
							})
						}
					case <-time.After(time.Second):
						canRun = false
					}
				}

				if len(items) == 0 {
					break
				}
				batch := &dynamodb.BatchWriteItemInput{
					RequestItems: map[string][]types.WriteRequest{
						Flags.AWS.DynamoDB.TableName: items,
					},
				}

				processed += len(items)

				res, err := cl.BatchWriteItem(context.Background(), batch)
				if res != nil && len(res.UnprocessedItems) > 0 && len(res.UnprocessedItems[Flags.AWS.DynamoDB.TableName]) > 0 {
					for _, item := range res.UnprocessedItems[Flags.AWS.DynamoDB.TableName] {
						ch <- item.PutRequest.Item
					}
					processed -= len(res.UnprocessedItems[Flags.AWS.DynamoDB.TableName])
					log.Printf("Added %d unprocessed items to the queue.", len(res.UnprocessedItems[Flags.AWS.DynamoDB.TableName]))
				}
				if err != nil {
					log.Printf("BatchWriteItemRequest error! %v", err)
					time.Sleep(time.Second / 10)
				} else {
					log.Print("BatchWriteItemRequest completed!")
				}
			}

			log.Printf("Thread %d processed %d item(s)", id, processed)
		}(i, client)
	}

	wg.Wait()
}

func (dl *DataLoader) restartRandomNode() (string, error) {
	res, err := dl.adminClient.DescribeClusters(context.Background(), &daxsvc.DescribeClustersInput{
		ClusterNames: []string{Flags.App.DaxClusterName},
	})
	if err != nil {
		return "", err
	}

	nodeIds := []string{}
	for _, c := range res.Clusters {
		for _, n := range c.Nodes {
			if aws.ToString(n.NodeStatus) == "available" {
				nodeIds = append(nodeIds, aws.ToString(n.NodeId))
			}
		}
	}

	idx := rand.IntN(len(nodeIds))

	_, err = dl.adminClient.RebootNode(context.Background(), &daxsvc.RebootNodeInput{
		ClusterName: aws.String(Flags.App.DaxClusterName),
		NodeId:      aws.String(nodeIds[idx]),
	})

	return nodeIds[idx], err
}
