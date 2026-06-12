package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type workerFn func(context.Context, chan types.MetricDatum, *dax.Dax, string, *AppConfig, chan bool)

func workerGetItem(ctx context.Context, metricChan chan types.MetricDatum, client *dax.Dax, tableName string, appConfig *AppConfig, throttleChan chan bool) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		start := time.Now()
		item, err := client.GetItem(
			context.Background(),
			&dynamodb.GetItemInput{
				TableName: aws.String(tableName),
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
		end := time.Now()
		metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.GetItem.latency_ms"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitMilliseconds,
			Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
		}

		if err != nil {
			log.Printf("GetItem() error: %v", err)
			metricChan <- types.MetricDatum{
				MetricName: aws.String("dax.op.GetItem.error"),
				Timestamp:  aws.Time(end),
				Unit:       types.StandardUnitCount,
				Value:      aws.Float64(1),
			}
			continue
		}

		metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.GetItem.success"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(1),
		}

		_ = item
	}
}
func workerBatchGetItem(ctx context.Context, metricChan chan types.MetricDatum, client *dax.Dax, tableName string, appConfig *AppConfig, throttleChan chan bool) {
	currPK := 0
	currSK := -1
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		kas := ddbtypes.KeysAndAttributes{}
		for len(kas.Keys) < 25 {
			currSK++
			if currSK == numSKs {
				currSK = 0
				currPK++
			}
			if currPK == numPKs {
				currPK = 0
			}
			kas.Keys = append(kas.Keys, map[string]ddbtypes.AttributeValue{
				"pk": &ddbtypes.AttributeValueMemberN{Value: strconv.Itoa(currPK)},
				"sk": &ddbtypes.AttributeValueMemberN{Value: strconv.Itoa(currSK)},
			})
		}

		start := time.Now()
		items, err := client.BatchGetItem(context.Background(), &dynamodb.BatchGetItemInput{
			RequestItems: map[string]ddbtypes.KeysAndAttributes{
				tableName: kas,
			},
		})
		end := time.Now()
		metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.BatchGetItem.latency_ms"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitMilliseconds,
			Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
		}

		if err != nil {
			log.Printf("BatchGetItem() error: %v", err)
			metricChan <- types.MetricDatum{
				MetricName: aws.String("dax.op.BatchGetItem.error"),
				Timestamp:  aws.Time(end),
				Unit:       types.StandardUnitCount,
				Value:      aws.Float64(1),
			}
			continue
		}

		metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.BatchGetItem.success"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(1),
		}

		_ = items
	}
}
func workerQuery(ctx context.Context, metricChan chan types.MetricDatum, client *dax.Dax, tableName string, appConfig *AppConfig, throttleChan chan bool) {
	size := appConfig.TrafficConfig.ItemSizes["Query"]

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		var exclusiveStartKey map[string]ddbtypes.AttributeValue

		for pk := range numPKs {
			start := time.Now()
			items, err := client.Query(context.Background(), &dynamodb.QueryInput{
				TableName:              aws.String(tableName),
				ExclusiveStartKey:      exclusiveStartKey,
				KeyConditionExpression: aws.String("pk = :pk and sk between :sk1 and :sk2"),
				ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
					":pk":  &ddbtypes.AttributeValueMemberN{Value: strconv.Itoa(pk)},
					":sk1": &ddbtypes.AttributeValueMemberN{Value: "0"},
					":sk2": &ddbtypes.AttributeValueMemberN{Value: fmt.Sprintf("%d", size)},
				},
			})

			end := time.Now()
			metricChan <- types.MetricDatum{
				MetricName: aws.String("dax.op.Query.latency_ms"),
				Timestamp:  aws.Time(end),
				Unit:       types.StandardUnitMilliseconds,
				Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
			}

			if err != nil {
				log.Printf("Query() error: %v", err)
				metricChan <- types.MetricDatum{
					MetricName: aws.String("dax.op.Query.error"),
					Timestamp:  aws.Time(end),
					Unit:       types.StandardUnitCount,
					Value:      aws.Float64(1),
				}
				continue
			}

			metricChan <- types.MetricDatum{
				MetricName: aws.String("dax.op.Query.success"),
				Timestamp:  aws.Time(end),
				Unit:       types.StandardUnitCount,
				Value:      aws.Float64(1),
			}

			_ = items
		}
	}
}

func workerRead(ctx context.Context, metricChan chan types.MetricDatum, client *dax.Dax, tableName string, appConfig *AppConfig, throttleChan chan bool) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		rand.Seed(time.Now().UnixNano())
		n := rand.Intn(100)

		if n < 75 {
			workerGetItem(ctx, metricChan, client, tableName, appConfig, throttleChan)
		} else if n < 90 {
			workerQuery(ctx, metricChan, client, tableName, appConfig, throttleChan)
		} else {
			workerBatchGetItem(ctx, metricChan, client, tableName, appConfig, throttleChan)
		}
	}
}

func workerWrite(ctx context.Context, metricChan chan types.MetricDatum, client *dax.Dax, tableName string, appConfig *AppConfig, throttleChan chan bool) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		rand.Seed(time.Now().UnixNano())
		n := rand.Intn(100)

		if n < 75 {
			workerPutItem(ctx, metricChan, client, tableName, appConfig, throttleChan)
		} else if n < 90 {
			workerUpdateItem(ctx, metricChan, client, tableName, appConfig, throttleChan)
		} else {
			workerBatchWriteItem(ctx, metricChan, client, tableName, appConfig, throttleChan)
		}
	}
}

func workerPutItem(ctx context.Context, metricChan chan types.MetricDatum, client *dax.Dax, tableName string, appConfig *AppConfig, throttleChan chan bool) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		pk := fmt.Sprintf("%d", putItemPKManager.next())
		sk := fmt.Sprintf("%d", putItemSKManager.next())
		m := map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberN{
				Value: pk,
			},
			"sk": &ddbtypes.AttributeValueMemberN{
				Value: sk,
			},
		}

		//remainder := int(appConfig.TrafficConfig.ItemSizes["PutItem"]) - len(pk) - len(sk)
		//sz := remainder / 6
		sz := max(0, (int(appConfig.TrafficConfig.ItemSizes["PutItem"])-len(pk)-len(sk))/6)

		for col := range 6 {
			m[fmt.Sprintf("a%d", col)] = &ddbtypes.AttributeValueMemberS{
				Value: strings.Repeat(strconv.Itoa(col), sz),
			}
		}

		start := time.Now()
		_, err := client.PutItem(context.Background(), &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      m,
		})

		end := time.Now()
		metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.PutItem.latency_ms"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitMilliseconds,
			Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
		}

		if err != nil {
			log.Printf("PutItem() error: %v", err)
			metricChan <- types.MetricDatum{
				MetricName: aws.String("dax.op.PutItem.error"),
				Timestamp:  aws.Time(end),
				Unit:       types.StandardUnitCount,
				Value:      aws.Float64(1),
			}

			if strings.Contains(err.Error(), "ThrottlingException") {
				throttleChan <- true
			}

			continue
		}

		metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.PutItem.success"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(1),
		}
	}
}

func workerUpdateItem(ctx context.Context, metricChan chan types.MetricDatum, client *dax.Dax, tableName string, appConfig *AppConfig, throttleChan chan bool) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		pk := fmt.Sprintf("%d", updateItemPKManager.next())
		sk := fmt.Sprintf("%d", updateItemSKManager.next())
		key := map[string]ddbtypes.AttributeValue{
			"pk": &ddbtypes.AttributeValueMemberN{Value: pk},
			"sk": &ddbtypes.AttributeValueMemberN{Value: sk},
		}

		//remainder := int(appConfig.TrafficConfig.ItemSizes["PutItem"]) - len(pk) - len(sk)
		//sz := remainder / 6
		sz := max(0, (int(appConfig.TrafficConfig.ItemSizes["UpdateItem"])-len(pk)-len(sk))/6)

		m := map[string]ddbtypes.AttributeValue{}
		for col := range 6 {
			m[fmt.Sprintf(":a%d", col)] = &ddbtypes.AttributeValueMemberS{
				Value: strings.Repeat(strconv.Itoa(col), sz),
			}
		}

		start := time.Now()
		_, err := client.UpdateItem(context.Background(), &dynamodb.UpdateItemInput{
			TableName:                 aws.String(tableName),
			Key:                       key,
			UpdateExpression:          aws.String("SET a0 = :a0, a1 = :a1, a2 = :a2, a3 = :a3, a4 = :a4, a5 = :a5"),
			ExpressionAttributeValues: m,
			ReturnValues:              ddbtypes.ReturnValueAllNew,
		})

		end := time.Now()
		metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.UpdateItem.latency_ms"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitMilliseconds,
			Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
		}

		if err != nil {
			log.Printf("UpdateItem() error: %v", err)
			metricChan <- types.MetricDatum{
				MetricName: aws.String("dax.op.UpdateItem.error"),
				Timestamp:  aws.Time(end),
				Unit:       types.StandardUnitCount,
				Value:      aws.Float64(1),
			}

			if strings.Contains(err.Error(), "ThrottlingException") {
				throttleChan <- true
			}

			continue
		}

		metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.UpdateItem.success"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(1),
		}
	}
}

func workerBatchWriteItem(ctx context.Context, metricChan chan types.MetricDatum, client *dax.Dax, tableName string, appConfig *AppConfig, throttleChan chan bool) {
	var reqs []ddbtypes.WriteRequest
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do not block! :)
		}

		for len(reqs) < 25 {
			pk := fmt.Sprintf("%d", batchWritePKManager.next())
			sk := fmt.Sprintf("%d", batchWriteSKManager.next())
			m := map[string]ddbtypes.AttributeValue{
				"pk": &ddbtypes.AttributeValueMemberN{
					Value: pk,
				},
				"sk": &ddbtypes.AttributeValueMemberN{
					Value: sk,
				},
			}

			//remainder := int(appConfig.TrafficConfig.ItemSizes["BatchWriteItem"]) - len(pk) - len(sk)
			//sz := remainder / 6
			sz := max(0, (int(appConfig.TrafficConfig.ItemSizes["BatchWriteItem"])-len(pk)-len(sk))/6)

			for col := range 6 {
				m[fmt.Sprintf("a%d", col)] = &ddbtypes.AttributeValueMemberS{
					Value: strings.Repeat(strconv.Itoa(col), sz),
				}
			}

			reqs = append(reqs, ddbtypes.WriteRequest{
				PutRequest: &ddbtypes.PutRequest{
					Item: m,
				},
			})
		}

		start := time.Now()
		res, err := client.BatchWriteItem(context.Background(), &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]ddbtypes.WriteRequest{
				tableName: reqs,
			},
		})

		end := time.Now()
		metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.BatchWriteItem.latency_ms"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitMilliseconds,
			Value:      aws.Float64(float64(end.UnixMilli() - start.UnixMilli())),
		}

		reqs = []ddbtypes.WriteRequest{}
		for c := range res.UnprocessedItems {
			reqs = append(res.UnprocessedItems[c])
		}

		if err != nil {
			log.Printf("BatchWriteItem() error: %v", err)
			metricChan <- types.MetricDatum{
				MetricName: aws.String("dax.op.BatchWriteItem.error"),
				Timestamp:  aws.Time(end),
				Unit:       types.StandardUnitCount,
				Value:      aws.Float64(1),
			}

			if strings.Contains(err.Error(), "ThrottlingException") {
				throttleChan <- true
			}

			<-time.After(time.Millisecond * 10)

			continue
		}

		metricChan <- types.MetricDatum{
			MetricName: aws.String("dax.op.BatchWriteItem.success"),
			Timestamp:  aws.Time(end),
			Unit:       types.StandardUnitCount,
			Value:      aws.Float64(1),
		}
	}
}
