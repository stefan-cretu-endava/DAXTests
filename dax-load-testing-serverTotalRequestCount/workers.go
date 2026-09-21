package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type OperationsCounter struct {
	getItemCntr              uint64
	batchGetItemCntr         uint64
	queryCntr                uint64
	putItemCntr              uint64
	updateItemCntr           uint64
	batchWriteItemCntr       uint64
	getItemPercentage        uint64
	batchGetItemPercentage   uint64
	queryPercentage          uint64
	putItemPercentage        uint64
	updateItemPercentage     uint64
	batchWriteItemPercentage uint64
	opsSum                   uint64
}

func (oc *OperationsCounter) resetCounters(read, write bool) {
	if read {
		atomic.StoreUint64(&oc.getItemCntr, 0)
		atomic.StoreUint64(&oc.batchGetItemCntr, 0)
		atomic.StoreUint64(&oc.queryCntr, 0)
	}
	if write {
		atomic.StoreUint64(&oc.putItemCntr, 0)
		atomic.StoreUint64(&oc.updateItemCntr, 0)
		atomic.StoreUint64(&oc.batchWriteItemCntr, 0)
	}
}

func (oc *OperationsCounter) scaleToAvgLoadperMinute(targetAvgLoadPerMinute uint64) {
	oc.opsSum = targetAvgLoadPerMinute
	scaleFactor := oc.opsSum / 100
	oc.getItemPercentage = oc.getItemPercentage * scaleFactor
	oc.batchGetItemPercentage = oc.batchGetItemPercentage * scaleFactor
	oc.queryPercentage = oc.queryPercentage * scaleFactor
	oc.putItemPercentage = oc.putItemPercentage * scaleFactor
	oc.updateItemPercentage = oc.updateItemPercentage * scaleFactor
	oc.batchWriteItemPercentage = oc.batchWriteItemPercentage * scaleFactor
}

func shallIncrementCounter(counter, percentage uint64) bool {
	return percentage != 0 && counter < percentage
}

type WorkerInterface interface {
	GetItemTraffic(ctx context.Context)
	BatchGetItemTraffic(ctx context.Context)
	QueryTraffic(ctx context.Context)
	ReadTraffic(ctx context.Context)
	PutItemTraffic(ctx context.Context)
	UpdateItemTraffic(ctx context.Context)
	BatchWriteItemTraffic(ctx context.Context)
	WriteTraffic(ctx context.Context)
	ReadWriteTraffic(ctx context.Context)
}

type SingleRequest interface {
	singleGetItemRequest(ctx context.Context) error
	singleBatchGetItemRequest(ctx context.Context, pk, sk *int) error
	singleQueryRequest(ctx context.Context, size int) error
	singlePutItemRequest(ctx context.Context) error
	singleUpdateItemRequest(ctx context.Context) error
	singleBatchWriteItemRequest(ctx context.Context, reqs []ddbtypes.WriteRequest) error
}

type Worker struct {
	metricChan          chan types.MetricDatum
	client              *dax.Dax
	tableName           string
	appConfig           *AppConfig
	throttleChan        chan bool
	opsCntr             *OperationsCounter
	targetAvgLoadPerMin uint64
}

/*
* WorkerInterface implementation
 */
// Function that executes GetItem requests for as long as the input contex deadline tells.
// It calls the singleGetItemRequest defined below, to record latency and throughput metrics, sending them to CloudWatch.
// In case error was encountered for the current request, it continues with the next iteration
func (w *Worker) GetItemTraffic(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		w.singleGetItemRequest(ctx)
	}
}

// Function that executes BatchGetItem requests for as long as the input contex deadline tells.
// It calls the singleBatchGetItemRequest defined below, to record latency and throughput metrics, sending them to CloudWatch.
// In case error was encountered for the current request, it continues with the next iteration
func (w *Worker) BatchGetItemTraffic(ctx context.Context) {
	currPK := 0
	currSK := -1
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		w.singleBatchGetItemRequest(ctx, &currPK, &currSK)
	}
}

// Function that executes Query requests for as long as the input contex deadline tells.
// It calls the singleQuery above, to record latency and throughput metrics, sending them to CloudWatch.
// In case error was encountered for the current request, it continues with the next iteration
func (w *Worker) QueryTraffic(ctx context.Context) {
	size := w.appConfig.TrafficConfig.ItemSizes["Query"]

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		w.singleQueryRequest(ctx, size)
	}
}

// Function that executes a comination of GetItem, batchgetItem and Query, for as long as the input ctx deadline indicates.
func (w *Worker) ReadTraffic(ctx context.Context) {
	batchgetItemCurrPK := 0
	batchGetItemCurrSK := -1
	querySize := w.appConfig.TrafficConfig.ItemSizes["Query"]
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		if w.opsCntr != nil && w.opsCntr.getItemPercentage+w.opsCntr.batchGetItemPercentage+w.opsCntr.queryPercentage == w.opsCntr.opsSum {
			// fmt.Println("Reading operations percentages", opsCntr.getItemPercentage, opsCntr.batchGetItemPercentage, opsCntr.queryPercentage,
			// 	"Counters:", opsCntr.getItemCntr, opsCntr.batchGetItemCntr, opsCntr.queryCntr)

			if w.opsCntr.getItemCntr+w.opsCntr.batchGetItemCntr+w.opsCntr.queryCntr == w.opsCntr.opsSum {
				fmt.Println("Resetting counters...", w.opsCntr.getItemCntr, w.opsCntr.batchGetItemCntr, w.opsCntr.queryCntr)
				w.opsCntr.resetCounters(true, false)
				fmt.Println("Reset counters", w.opsCntr.getItemCntr, w.opsCntr.batchGetItemCntr, w.opsCntr.queryCntr)
			}

			if shallIncrementCounter(w.opsCntr.getItemCntr, w.opsCntr.getItemPercentage) {
				if w.opsCntr.getItemCntr != 0 && shallIncrementCounter(w.opsCntr.queryCntr, w.opsCntr.queryPercentage) && (w.opsCntr.getItemCntr+w.opsCntr.batchGetItemCntr+w.opsCntr.queryCntr)%(w.opsCntr.opsSum/w.opsCntr.queryPercentage) == 0 {
					w.singleQueryRequest(ctx, querySize)
					atomic.AddUint64(&w.opsCntr.queryCntr, 1)
				} else if shallIncrementCounter(w.opsCntr.batchGetItemCntr, w.opsCntr.batchGetItemPercentage) && w.opsCntr.batchGetItemCntr < w.opsCntr.getItemCntr {
					w.singleBatchGetItemRequest(ctx, &batchgetItemCurrPK, &batchGetItemCurrSK)
					atomic.AddUint64(&w.opsCntr.batchGetItemCntr, 1)
				} else {
					w.singleGetItemRequest(ctx)
					atomic.AddUint64(&w.opsCntr.getItemCntr, 1)
				}
			} else if shallIncrementCounter(w.opsCntr.batchGetItemCntr, w.opsCntr.batchGetItemPercentage) {
				if w.opsCntr.batchGetItemCntr != 0 && shallIncrementCounter(w.opsCntr.queryCntr, w.opsCntr.queryPercentage) && (w.opsCntr.getItemCntr+w.opsCntr.batchGetItemCntr+w.opsCntr.queryCntr)%(w.opsCntr.opsSum/w.opsCntr.queryPercentage) == 0 {
					w.singleQueryRequest(ctx, querySize)
					atomic.AddUint64(&w.opsCntr.queryCntr, 1)
				} else {
					w.singleBatchGetItemRequest(ctx, &batchgetItemCurrPK, &batchGetItemCurrSK)
					atomic.AddUint64(&w.opsCntr.batchGetItemCntr, 1)
				}
			} else if shallIncrementCounter(w.opsCntr.queryCntr, w.opsCntr.queryPercentage) {
				w.singleQueryRequest(ctx, querySize)
				atomic.AddUint64(&w.opsCntr.queryCntr, 1)
			}
		} else {
			rand.Seed(time.Now().UnixNano())
			n := rand.Intn(100)

			if n < 75 {
				w.singleGetItemRequest(ctx)
			} else if n < 90 {
				w.singleQueryRequest(ctx, querySize)
			} else {
				w.singleBatchGetItemRequest(ctx, &batchgetItemCurrPK, &batchGetItemCurrSK)
			}
		}
	}
}

func (w *Worker) PutItemTraffic(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		w.singlePutItemRequest(ctx)
	}
}

func (w *Worker) UpdateItemTraffic(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		w.singleUpdateItemRequest(ctx)
	}
}

func (w *Worker) BatchWriteItemTraffic(ctx context.Context) {
	var reqs []ddbtypes.WriteRequest
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		w.singleBatchWriteItemRequest(ctx, reqs)

	}
}

func (w *Worker) WriteTraffic(ctx context.Context) {
	var reqs []ddbtypes.WriteRequest
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		if w.opsCntr != nil && w.opsCntr.putItemPercentage+w.opsCntr.batchWriteItemPercentage+w.opsCntr.updateItemPercentage == w.opsCntr.opsSum {
			// fmt.Println("Reading operations percentages", opsCntr.putItemPercentage, opsCntr.batchWriteItemPercentage, opsCntr.queryPercentage,
			// 	"Counters:", opsCntr.getItemCntr, opsCntr.batchGetItemCntr, opsCntr.queryCntr)

			if w.opsCntr.putItemCntr+w.opsCntr.batchWriteItemCntr+w.opsCntr.updateItemCntr == w.opsCntr.opsSum {
				fmt.Println("Resetting counters...", w.opsCntr.putItemCntr, w.opsCntr.batchWriteItemCntr, w.opsCntr.updateItemCntr)
				w.opsCntr.resetCounters(true, false)
				fmt.Println("Reset counters", w.opsCntr.putItemCntr, w.opsCntr.batchWriteItemCntr, w.opsCntr.updateItemCntr)
			}

			if shallIncrementCounter(w.opsCntr.putItemCntr, w.opsCntr.putItemPercentage) {
				if w.opsCntr.putItemCntr != 0 && shallIncrementCounter(w.opsCntr.updateItemCntr, w.opsCntr.updateItemPercentage) && (w.opsCntr.putItemCntr+w.opsCntr.batchWriteItemCntr+w.opsCntr.updateItemCntr)%(w.opsCntr.opsSum/w.opsCntr.updateItemPercentage) == 0 {
					w.singleUpdateItemRequest(ctx)
					atomic.AddUint64(&w.opsCntr.updateItemCntr, 1)
				} else if shallIncrementCounter(w.opsCntr.batchWriteItemCntr, w.opsCntr.batchWriteItemPercentage) && w.opsCntr.batchWriteItemCntr < w.opsCntr.putItemCntr {
					w.singleBatchWriteItemRequest(ctx, reqs)
					atomic.AddUint64(&w.opsCntr.batchWriteItemCntr, 1)
				} else {
					w.singlePutItemRequest(ctx)
					atomic.AddUint64(&w.opsCntr.putItemCntr, 1)
				}
			} else if shallIncrementCounter(w.opsCntr.batchWriteItemCntr, w.opsCntr.batchWriteItemPercentage) {
				if w.opsCntr.batchWriteItemCntr != 0 && shallIncrementCounter(w.opsCntr.updateItemCntr, w.opsCntr.updateItemPercentage) && (w.opsCntr.putItemCntr+w.opsCntr.batchWriteItemCntr+w.opsCntr.updateItemCntr)%(w.opsCntr.opsSum/w.opsCntr.updateItemPercentage) == 0 {
					w.singleUpdateItemRequest(ctx)
					atomic.AddUint64(&w.opsCntr.updateItemCntr, 1)
				} else {
					w.singleBatchWriteItemRequest(ctx, reqs)
					atomic.AddUint64(&w.opsCntr.batchWriteItemCntr, 1)
				}
			} else if shallIncrementCounter(w.opsCntr.updateItemCntr, w.opsCntr.updateItemPercentage) {
				w.singleUpdateItemRequest(ctx)
				atomic.AddUint64(&w.opsCntr.updateItemCntr, 1)
			}
		} else {
			rand.Seed(time.Now().UnixNano())
			n := rand.Intn(100)

			if n < 50 {
				w.singlePutItemRequest(ctx)
				//} //else if n < 90 {
				//w.singleUpdateItemRequest(ctx)
			} else {
				w.singleBatchWriteItemRequest(ctx, reqs)
			}
		}
	}
}

func (w *Worker) ReadWriteTraffic(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		if w.opsCntr != nil && w.opsCntr.putItemPercentage+w.opsCntr.getItemPercentage == w.opsCntr.opsSum {
			// fmt.Println("Reading operations percentages", opsCntr.putItemPercentage, opsCntr.batchWriteItemPercentage, opsCntr.queryPercentage,
			// 	"Counters:", opsCntr.getItemCntr, opsCntr.batchGetItemCntr, opsCntr.queryCntr)

			if w.opsCntr.putItemCntr+w.opsCntr.getItemCntr == w.opsCntr.opsSum {
				fmt.Println("Resetting counters...", w.opsCntr.putItemCntr, w.opsCntr.getItemCntr)
				w.opsCntr.resetCounters(true, false)
				fmt.Println("Reset counters", w.opsCntr.putItemCntr, w.opsCntr.getItemCntr)
			}

			if shallIncrementCounter(w.opsCntr.putItemCntr, w.opsCntr.putItemPercentage) {
				if shallIncrementCounter(w.opsCntr.getItemCntr, w.opsCntr.getItemPercentage) && w.opsCntr.getItemCntr < w.opsCntr.putItemCntr {
					w.singleGetItemRequest(ctx)
					atomic.AddUint64(&w.opsCntr.getItemCntr, 1)
				} else {
					w.singlePutItemRequest(ctx)
					atomic.AddUint64(&w.opsCntr.putItemCntr, 1)
				}
			} else if shallIncrementCounter(w.opsCntr.getItemCntr, w.opsCntr.getItemPercentage) {
				w.singleGetItemRequest(ctx)
				atomic.AddUint64(&w.opsCntr.getItemCntr, 1)
			}
		} else {
			rand.Seed(time.Now().UnixNano())
			n := rand.Intn(100)

			if n < 50 {
				w.singlePutItemRequest(ctx)
			} else {
				w.singleGetItemRequest(ctx)
			}
		}
	}
}

/*
* SingleRequest interface implementation
 */
// Function that executes a single GetItem request, recording latency and throughput metrics, sending them to CloudWatch. Returns the encountered error, if any.
// The GetItem operation returns a set of attributes for the item with the given primary key. If there is no matching item, GetItem does not return any data and
// there will be no Item element in the response. GetItem provides an eventually consistent read by default
func (w *Worker) singleGetItemRequest(ctx context.Context) error {
	//Record the time when the request is issued
	start := time.Now()
	//Make a GetItem request using the dax.Dax client. The interface requires a context and a op-specific struct that includes table name and key of the item to be gotten
	item, err := w.client.GetItem(
		ctx, //either ctx or context.Background() shall be used
		&dynamodb.GetItemInput{
			TableName: aws.String(w.tableName),
			Key: map[string]ddbtypes.AttributeValue{
				"pk": &ddbtypes.AttributeValueMemberN{
					Value: fmt.Sprintf("%d", getItemPKManager.next()),
				},
				"sk": &ddbtypes.AttributeValueMemberN{
					Value: fmt.Sprintf("%d", getItemSKManager.next()),
				},
			},
		},
	)
	//Record time when the request returned, to compute latency
	end := time.Now()
	//Increment the issued requests global count, regardless of their type
	atomic.AddUint64(&reqCount, 1)

	//Use the metric channel to send the duration for the issued request, measured in milliseconds. It makes use of the DynamoDB SDK struct MetricDatum
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.GetItem.latency_ms"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitMilliseconds,
		Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
	}

	//In case the request returned error, use the same metric channel to send a value recording the error count associated with GetItem error metric, then return.
	//If the error indicates throttling, notify on throttleChan, for adjusting the traffic accordingly
	if err != nil {
		log.Printf("GetItem() error: %v", err)
		w.metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.GetItem.error"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(1),
		}

		if strings.Contains(err.Error(), "ThrottlingException") {
			w.throttleChan <- true
		}

		return err
	}

	//If the request was successful, use the metric channel to update the counter associated with the op-specific successful request metric
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.GetItem.success"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitCount,
		Value:      aws.Float64(1),
	}

	_ = item

	return err
}

// Function that executes a single BatchGetItem request, recording latency and throughput metrics, sending them to CloudWatch. Returns the request's error, if any
// The BatchGetItem operation returns the attributes of one or more items from one or more tables, identifying requested items by primary key. (pk+sk)
// A single operation can retrieve up to 16 MB of data, which can contain as many as 100 items
func (w *Worker) singleBatchGetItemRequest(ctx context.Context, currPK, currSK *int) error {
	//Use a DynamoDB map to store the keys of the elements that will be retrieved in batch. Provide a limitation of 25 items.
	kas := ddbtypes.KeysAndAttributes{}
	for len(kas.Keys) < 25 {
		// Record the next element in batch by incrementing the sort key. If it reaches the limit, reset it to 0 and increment the partition key
		*currSK++
		if *currSK == numSKs {
			*currSK = 0
			*currPK++
		}
		// Reset to 0 the partition key, if it reaches the limit
		if *currPK == numPKs {
			*currPK = 0
		}
		// Store the keys in the map, so the elements' attributes will be identified based on the primary key composed from them
		kas.Keys = append(kas.Keys, map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberN{Value: strconv.Itoa(*currPK)},
			"sk": &ddbtypes.AttributeValueMemberN{Value: strconv.Itoa(*currSK)},
		})
	}

	//Record the time when the request is issued
	start := time.Now()
	//Make a BatchGetItem request using the dax.Dax client. The interface requires a context and a op-specific struct that includes the parimary keys of the items to be gotten
	items, err := w.client.BatchGetItem(
		ctx,
		&dynamodb.BatchGetItemInput{
			RequestItems: map[string]ddbtypes.KeysAndAttributes{
				w.tableName: kas,
			},
		})
	//Record time when the request returned, to compute latency
	end := time.Now()
	//Increment the issued requests global count, regardless of their type
	atomic.AddUint64(&reqCount, 1)

	//Use the metric channel to send the duration for the issued request, measured in milliseconds. It makes use of the DynamoDB SDK struct MetricDatum
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.BatchGetItem.latency_ms"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitMilliseconds,
		Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
	}

	//In case the request returned error, use the same metric channel to send a value recording the error count associated with GetItem error metric, then return.
	//If the error indicates throttling, notify on throttleChan, for adjusting the traffic accordingly
	if err != nil {
		log.Printf("BatchGetItem() error: %v", err)
		w.metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.BatchGetItem.error"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(1),
		}

		if strings.Contains(err.Error(), "ThrottlingException") {
			w.throttleChan <- true
		}

		return err
	}

	//If the request was successful, use the metric channel to update the counter associated with the op-specific successful request metric
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.BatchGetItem.success"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitCount,
		Value:      aws.Float64(1),
	}

	_ = items

	return err
}

// Function that executes a single Query request, recording latency and throughput metrics, sending them to CloudWatch. Returns the request's error, if any.
// Query API requires the name of the partition key attribute and a single value for that attribute. Use the KeyConditionExpression parameter to provide a specific
// value for the partition key. The Query operation will return all of the items from the table or index with that partition key value.
// You can optionally narrow the scope of the Query operation by specifying a sort key value and a comparison operator in KeyConditionExpression
func (w *Worker) singleQueryRequest(ctx context.Context, size int64) error {
	var exclusiveStartKey map[string]ddbtypes.AttributeValue
	//Randomly generate partition key in range [0;255]
	pk := rand.Intn(256)

	//Record the time when the request is issued
	start := time.Now()
	//Make a Query request using the dax.Dax client. The interface requires a context and a op-specific struct that includes the table name, partition key and
	// sort key range of the items to be gotten
	items, err := w.client.Query(
		ctx,
		&dynamodb.QueryInput{
			TableName:              aws.String(w.tableName),
			ExclusiveStartKey:      exclusiveStartKey,
			KeyConditionExpression: aws.String("pk = :pk and sk between :sk1 and :sk2"),
			ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
				":pk":  &ddbtypes.AttributeValueMemberN{Value: strconv.Itoa(pk)},
				":sk1": &ddbtypes.AttributeValueMemberN{Value: "0"},
				":sk2": &ddbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", size)},
			},
		})
	//Record time when the request returned, to compute latency
	end := time.Now()
	//Increment the issued requests global count, regardless of their type
	atomic.AddUint64(&reqCount, 1)

	//Use the metric channel to send the duration for the issued request, measured in milliseconds. It makes use of the DynamoDB SDK struct MetricDatum
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.Query.latency_ms"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitMilliseconds,
		Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
	}

	//In case the request returned error, use the same metric channel to send a value recording the error count associated with GetItem error metric, then return.
	//If the error indicates throttling, notify on throttleChan, for adjusting the traffic accordingly
	if err != nil {
		log.Printf("Query() error: %v", err)
		w.metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.Query.error"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(1),
		}

		if strings.Contains(err.Error(), "ThrottlingException") {
			w.throttleChan <- true
		}
		return err
	}

	//If the request was successful, use the metric channel to update the counter associated with the op-specific successful request metric
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.Query.success"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitCount,
		Value:      aws.Float64(1),
	}

	_ = items

	return err
}

// Creates a new item, or replaces an old item with a new item. If an item that has the same primary key as the new item already exists in the specified table,
// the new item completely replaces the existing item.
func (w *Worker) singlePutItemRequest(ctx context.Context) error {
	// Compute partition and sort keys for the item to be inserted
	pk := fmt.Sprintf("%d", putItemPKManager.next())
	sk := fmt.Sprintf("%d", putItemSKManager.next())
	//Store the pk and the sk in a DynamoDB SDK struct
	item := map[string]ddbtypes.AttributeValue{
		"pk": &ddbtypes.AttributeValueMemberN{
			Value: pk,
		},
		"sk": &ddbtypes.AttributeValueMemberN{
			Value: sk,
		},
	}

	//Compute the size left in table after storing pk and sk to infer the available dimesnion for inserting item's attributes
	sz := max(0, (int(w.appConfig.TrafficConfig.ItemSizes["PutItem"])-len(pk)-len(sk))/6)
	//Insert item's attributes in the DynamoDB SDK struct
	for col := range 6 {
		item[fmt.Sprintf("a%d", col)] = &ddbtypes.AttributeValueMemberS{
			Value: strings.Repeat(strconv.Itoa(col), sz),
		}
	}

	//Record the time when the request is issued
	start := time.Now()
	//Make a PutItem request using the dax.Dax client. The interface requires a context and a map containing item's partition and sort keys, as well as its attribute fields
	_, err := w.client.PutItem(
		ctx,
		&dynamodb.PutItemInput{
			TableName: aws.String(w.tableName),
			Item:      item,
		})
	//Record time when the request returned, to compute latency
	end := time.Now()
	//Increment the issued requests global count, regardless of their type
	atomic.AddUint64(&reqCount, 1)

	//Use the metric channel to send the duration for the issued request, measured in milliseconds. It makes use of the DynamoDB SDK struct MetricDatum
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.PutItem.latency_ms"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitMilliseconds,
		Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
	}

	//In case the request returned error, use the same metric channel to send a value recording the error count associated with GetItem error metric, then return.
	//If the error indicates throttling, notify on throttleChan, for adjusting the traffic accordingly
	if err != nil {
		log.Printf("PutItem() error: %v", err)
		w.metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.PutItem.error"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(1),
		}

		if strings.Contains(err.Error(), "ThrottlingException") {
			w.throttleChan <- true
		}

		return err
	}

	//If the request was successful, use the metric channel to update the counter associated with the op-specific successful request metric
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.PutItem.success"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitCount,
		Value:      aws.Float64(1),
	}

	return err
}

// Edits an existing item's attributes, or adds a new item to the table if it does not already exist. The item's attributes' values can be put, deleted, or added.
// It can also be performed a conditional update on an existing item: insert a new attribute name-value pair if it doesn't exist, or replace an existing name-value pair
// if it has certain expected attribute values).
func (w *Worker) singleUpdateItemRequest(ctx context.Context) error {
	// Compute partition and sort keys for the item to be updated
	pk := fmt.Sprintf("%d", updateItemPKManager.next())
	sk := fmt.Sprintf("%d", updateItemSKManager.next())
	//Store the pk and the sk in a DynamoDB SDK struct
	key := map[string]ddbtypes.AttributeValue{
		"pk": &ddbtypes.AttributeValueMemberN{Value: pk},
		"sk": &ddbtypes.AttributeValueMemberN{Value: sk},
	}

	//Compute the size left in table after storing pk and sk to infer the available dimesnion for inserting item's attributes
	sz := max(0, (int(w.appConfig.TrafficConfig.ItemSizes["UpdateItem"])-len(pk)-len(sk))/6)
	//Insert item's attributes in the DynamoDB SDK struct
	item := map[string]ddbtypes.AttributeValue{}
	for col := range 6 {
		item[fmt.Sprintf(":a%d", col)] = &ddbtypes.AttributeValueMemberS{
			Value: strings.Repeat(strconv.Itoa(col), sz),
		}
	}

	//Record the time when the request is issued
	start := time.Now()
	//Make a UpdateItem request using the dax.Dax client. The interface requires a context and a op-specific struct containing table name, item primary key, update expression
	_, err := w.client.UpdateItem(
		ctx,
		&dynamodb.UpdateItemInput{
			TableName:                 aws.String(w.tableName),
			Key:                       key,
			UpdateExpression:          aws.String("SET a0 = :a0, a1 = :a1, a2 = :a2, a3 = :a3, a4 = :a4, a5 = :a5"),
			ExpressionAttributeValues: item,
			ReturnValues:              ddbtypes.ReturnValueAllNew,
		})
	//Record time when the request returned, to compute latency
	end := time.Now()
	//Increment the issued requests global count, regardless of their type
	atomic.AddUint64(&reqCount, 1)

	//Use the metric channel to send the duration for the issued request, measured in milliseconds. It makes use of the DynamoDB SDK struct MetricDatum
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.UpdateItem.latency_ms"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitMilliseconds,
		Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
	}

	//In case the request returned error, use the same metric channel to send a value recording the error count associated with GetItem error metric, then return.
	//If the error indicates throttling, notify on throttleChan, for adjusting the traffic accordingly
	if err != nil {
		log.Printf("UpdateItem() error: %v", err)
		w.metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.UpdateItem.error"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(1),
		}

		if strings.Contains(err.Error(), "ThrottlingException") {
			w.throttleChan <- true
		}

		return err
	}

	//If the request was successful, use the metric channel to update the counter associated with the op-specific successful request metric
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.UpdateItem.success"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitCount,
		Value:      aws.Float64(1),
	}

	return err
}

// The BatchWriteItem operation puts or deletes multiple items in one or more tables. A single call to BatchWriteItem can transmit up to 16MB of data over the network,
// consisting of up to 25 item put or delete operations. While individual items can be up to 400 KB once stored, it's important to note that an item's representation
// might be greater than 400KB while being sent in DynamoDB's JSON format for the API call. If any requested operations fail because the table's provisioned throughput
// is exceeded or an internal processing failure occurs, the failed operations are returned in the UnprocessedItems response parameter. You can investigate and optionally resend the requests.
func (w *Worker) singleBatchWriteItemRequest(ctx context.Context, reqs []ddbtypes.WriteRequest) error {
	// Create a batch of 25 items
	for len(reqs) < 25 {
		// Compute partition and sort keys for the item to be modified
		pk := fmt.Sprintf("%d", batchWritePKManager.next())
		sk := fmt.Sprintf("%d", batchWriteSKManager.next())
		//Store the pk and the sk in a DynamoDB SDK struct
		item := map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberN{
				Value: pk,
			},
			"sk": &ddbtypes.AttributeValueMemberN{
				Value: sk,
			},
		}

		//Compute the size left in table after storing pk and sk to infer the available dimesnion for inserting item's attributes
		sz := max(0, (int(w.appConfig.TrafficConfig.ItemSizes["BatchWriteItem"])-len(pk)-len(sk))/6)
		//Insert item's attributes in the DynamoDB SDK struct
		for col := range 6 {
			item[fmt.Sprintf("a%d", col)] = &ddbtypes.AttributeValueMemberS{
				Value: strings.Repeat(strconv.Itoa(col), sz),
			}
		}

		// Store the current batch in the input slice, using Writerequest type, which represents an operation to perform - either DeleteItem or PutItem.
		// A single WriteRequest can only request one of these operations, not both.
		reqs = append(reqs, ddbtypes.WriteRequest{
			PutRequest: &ddbtypes.PutRequest{
				Item: item,
			},
		})
	}

	//Record the time when the request is issued
	start := time.Now()
	//Make a BatchWriteItem request using the dax.Dax client. The interface requires a context and a op-specific struct containing the write requests
	res, err := w.client.BatchWriteItem(
		ctx,
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]ddbtypes.WriteRequest{
				w.tableName: reqs,
			},
		})
	//Record time when the request returned, to compute latency
	end := time.Now()
	//Increment the issued requests global count, regardless of their type
	atomic.AddUint64(&reqCount, 1)

	//Use the metric channel to send the duration for the issued request, measured in milliseconds. It makes use of the DynamoDB SDK struct MetricDatum
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.BatchWriteItem.latency_ms"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitMilliseconds,
		Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
	}

	// Any unprocessed requests are retrieved from the result and inserted again in the input slice, to be reexecuted
	reqs = []ddbtypes.WriteRequest{}
	if res != nil && res.UnprocessedItems != nil {
		for c := range res.UnprocessedItems {
			reqs = append(res.UnprocessedItems[c])
		}
	}

	//In case the request returned error, use the same metric channel to send a value recording the error count associated with GetItem error metric, then return.
	//If the error indicates throttling, notify on throttleChan, for adjusting the traffic accordingly
	if err != nil {
		log.Printf("BatchWriteItem() error: %v", err)
		w.metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.BatchWriteItem.error"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(1),
		}

		if strings.Contains(err.Error(), "ThrottlingException") {
			w.throttleChan <- true
		}

		<-time.After(time.Millisecond * 10)

		return err
	}

	//If the request was successful, use the metric channel to update the counter associated with the op-specific successful request metric
	w.metricChan <- types.MetricDatum{
		MetricName: aws.String("dax.op.BatchWriteItem.success"),
		Timestamp:  aws.Time(end),
		Unit:       types.StandardUnitCount,
		Value:      aws.Float64(1),
	}

	return err
}
