package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type BatchGetWriteClient interface {
	BatchGetItem(ctx context.Context, params *dynamodb.BatchGetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
	BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
}

const (
	numPKs = 256
	numSKs = 256
)

func loadDataForRead(ctx context.Context, client BatchGetWriteClient, tableName string, size int64) {
	fmt.Println("Loading data for read")
	defer fmt.Println("Loaded data for read")

	numGoRoutines := 8

	times := make(chan int64, numPKs*numSKs)

	ch := make(chan ddbtypes.PutRequest, numPKs*numSKs)
	defer close(ch)

	wg := &sync.WaitGroup{}
	for c := range numGoRoutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			limit := numPKs / numGoRoutines
			startPK := limit * i

			sent := int64(0)
			for pk := startPK; pk < startPK+limit; pk++ {
				for sk := 0; sk < numSKs; sk++ {
					select {
					case <-ctx.Done():
						return
					default:
						//
					}

					pk := strconv.Itoa(pk)
					sk := strconv.Itoa(sk)
					m := map[string]ddbtypes.AttributeValue{
						"pk": &ddbtypes.AttributeValueMemberN{
							Value: pk,
						},
						"sk": &ddbtypes.AttributeValueMemberN{
							Value: sk,
						},
					}

					remainder := int(size) - len(pk) - len(sk)
					sz := remainder / 6

					for col := range 6 {
						m[fmt.Sprintf("a%d", col)] = &ddbtypes.AttributeValueMemberS{
							Value: strings.Repeat(strconv.Itoa(col), sz),
						}
					}

					atomic.AddInt64(&sent, 1)
					ch <- ddbtypes.PutRequest{
						Item: m,
					}
				}
			}
		}(c)
	}

	wg.Wait()

	for c := range numGoRoutines {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			for {
				start := time.Now()
				var wrs []ddbtypes.WriteRequest
				for run := true; run; {
					select {
					case <-ctx.Done():
						return
					case pr, ok := <-ch:
						if !ok {
							run = false
							break
						}
						wrs = append(wrs, ddbtypes.WriteRequest{
							PutRequest: &pr,
						})
						if len(wrs) == 25 {
							run = false
							break
						}
					case <-time.After(time.Second):
						run = false
						break
					}
				}

				if len(wrs) == 0 {
					break
				}

				res, err := client.BatchWriteItem(context.Background(), &dynamodb.BatchWriteItemInput{
					RequestItems: map[string][]ddbtypes.WriteRequest{
						tableName: wrs,
					},
				})
				if err != nil {
					fmt.Printf("[ERROR][%d] %v\n", i, err)
				}

				requeued := 0
				if res != nil && len(res.UnprocessedItems) > 0 {
					for _, uis := range res.UnprocessedItems {
						fmt.Printf("[WARNING][%d] Will now requeue %d unprocessed item(s)\n", i, len(uis))
						for _, ui := range uis {
							ch <- *ui.PutRequest
							requeued++
						}
					}
				}

				times <- time.Since(start).Milliseconds()
				if requeued > 0 {
					<-time.After(time.Millisecond * 8)
				}
			}
		}(c)
	}

	wg.Wait()

	total := int64(0)
	numEntries := int64(0)

	for len(times) > 0 {
		select {
		case d, ok := <-times:
			if ok {
				total += d
				numEntries++
			}
			break
		default:
			break
		}
	}
}
