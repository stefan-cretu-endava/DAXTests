package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	endpoint   = flag.String("endpoint", "", "DAX endpoint")
	region     = flag.String("region", "us-east-1", "AWS region")
	table      = flag.String("table", "", "Fixed table name (optional)")
	useDDB     = flag.Bool("ddb", false, "Use DynamoDB instead of DAX")
	duration   = flag.Int("duration", 60, "Test duration in seconds")
	totalItems = flag.Int("items", 10, "Total number of items to prepare")
)

// Predefined test configurations
var (
	operations        = []string{"getitem", "putitem", "query", "scan", "batchgetitem"}
	itemSizes         = []int{200, 2000}
	concurrencyLevels = struct {
		DAX []int
		DDB []int
	}{
		DAX: []int{1, 2, 8},
		DDB: []int{1, 8, 32},
	}
)

const (
	possible = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

type TestResult struct {
	Operation  string
	ItemSize   int
	Concurrent int
	Throughput float64
	Errors     float64
	P50        float64
	P99        float64
	P999       float64
	Min        float64
	Max        float64
}

type DynamoClient interface {
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(context.Context, *dynamodb.PutItemInput, ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(context.Context, *dynamodb.ScanInput, ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	BatchGetItem(context.Context, *dynamodb.BatchGetItemInput, ...func(*dynamodb.Options)) (*dynamodb.BatchGetItemOutput, error)
}

func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = possible[rand.Intn(len(possible))]
	}
	return string(b)
}

func randomItem(itemSize int) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"avS": &types.AttributeValueMemberS{Value: randomString(itemSize / 2)},
		"avB": &types.AttributeValueMemberB{Value: make([]byte, itemSize/2)},
		"avN": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", rand.Int31())},
	}
}

func createTable(ctx context.Context, client *dynamodb.Client, operation string, concurrency, itemSize, totalItems int) (string, error) {
	if *table != "" {
		// Check if table exists
		_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: table,
		})
		if err == nil {
			log.Printf("Using existing table: %s", *table)
			return *table, nil
		}
		return "", fmt.Errorf("specified table %s does not exist or is not accessible: %v", *table, err)
	}

	// Generate unique table name with timestamp
	newTableName := fmt.Sprintf("dax_perf_%s_%d_%d_%d_%d",
		operation,
		itemSize,
		concurrency,
		totalItems,
		time.Now().Unix(),
	)

	readCapacity := int64(500)
	writeCapacity := int64(10)
	if operation == "putitem" {
		writeCapacity = int64(5000)
		readCapacity = int64(10)
	}

	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("hk"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("rk"),
				AttributeType: types.ScalarAttributeTypeN,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("hk"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("rk"),
				KeyType:       types.KeyTypeRange,
			},
		},
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(readCapacity),
			WriteCapacityUnits: aws.Int64(writeCapacity),
		},
		TableName: aws.String(newTableName),
	}

	_, err := client.CreateTable(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create table: %w", err)
	}

	waiter := dynamodb.NewTableExistsWaiter(client)
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(newTableName),
	}, 5*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to wait for table creation: %w", err)
	}

	return newTableName, nil
}

func doRun(ctx context.Context, client DynamoClient, operation string, reqs []interface{}, concurrency int, duration time.Duration) (int64, int64, []float64) {
	var totalResponse int64
	var totalErrors int64
	latencies := make([]float64, 0)
	latenciesChan := make(chan float64, 1000)

	lg := NewLoadGen(func() error {
		reqIndex := atomic.LoadInt64(&totalResponse) % int64(len(reqs))
		start := time.Now()

		var err error
		switch operation {
		case "getitem":
			req := reqs[reqIndex].(*dynamodb.GetItemInput)
			_, err = client.GetItem(ctx, req)

		case "putitem":
			req := reqs[reqIndex].(*dynamodb.PutItemInput)
			newItem := map[string]types.AttributeValue{
				"hk": &types.AttributeValueMemberS{
					Value: fmt.Sprintf("hashkey-%d-%d", time.Now().UnixNano(), atomic.LoadInt64(&totalResponse)),
				},
				"rk": &types.AttributeValueMemberN{
					Value: fmt.Sprintf("%d", rand.Int63()),
				},
				"avS": &types.AttributeValueMemberS{
					Value: randomString(200),
				},
			}

			putReq := &dynamodb.PutItemInput{
				TableName: req.TableName,
				Item:      newItem,
			}
			_, err = client.PutItem(ctx, putReq)

		case "query":
			req := reqs[reqIndex].(*dynamodb.QueryInput)
			_, err = client.Query(ctx, req)

		case "scan":
			req := reqs[reqIndex].(*dynamodb.ScanInput)
			_, err = client.Scan(ctx, req)

		case "batchgetitem":
			req := reqs[reqIndex].(*dynamodb.BatchGetItemInput)
			_, err = client.BatchGetItem(ctx, req)
		}

		latency := time.Since(start)
		if err != nil {
			atomic.AddInt64(&totalErrors, 1)
			log.Printf("Operation error: %v", err)
			return err
		}

		atomic.AddInt64(&totalResponse, 1)
		latenciesChan <- float64(latency.Microseconds())
		return nil
	}, duration, concurrency)

	// Collect latencies in a thread-safe manner
	var latencyMutex sync.Mutex
	go func() {
		for latency := range latenciesChan {
			latencyMutex.Lock()
			latencies = append(latencies, latency)
			latencyMutex.Unlock()
		}
	}()

	lg.Start(ctx)
	close(latenciesChan)

	return totalResponse, totalErrors, latencies
}

func prepareItems(ctx context.Context, client DynamoClient, tableName string, itemSize, totalItems int, operation string) ([]interface{}, error) {
	items := make([]interface{}, totalItems)

	switch operation {
	case "putitem":
		for i := 0; i < totalItems; i++ {
			item := randomItem(itemSize)
			item["hk"] = &types.AttributeValueMemberS{Value: fmt.Sprintf("hashkey%d", i)}
			item["rk"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", i)}

			items[i] = &dynamodb.PutItemInput{
				TableName: aws.String(tableName),
				Item:      item,
			}
		}

	case "getitem":
		for i := 0; i < totalItems; i++ {
			items[i] = &dynamodb.GetItemInput{
				TableName: aws.String(tableName),
				Key: map[string]types.AttributeValue{
					"hk": &types.AttributeValueMemberS{Value: fmt.Sprintf("hashkey%d", i)},
					"rk": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", i)},
				},
			}
		}

	case "query":
		for i := 0; i < totalItems; i++ {
			items[i] = &dynamodb.QueryInput{
				TableName:              aws.String(tableName),
				KeyConditionExpression: aws.String("hk = :hk"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":hk": &types.AttributeValueMemberS{Value: fmt.Sprintf("hashkey%d", i)},
				},
			}
		}

	case "scan":
		items[0] = &dynamodb.ScanInput{
			TableName: aws.String(tableName),
			Limit:     aws.Int32(1000),
		}
		return items[:1], nil

	case "batchgetitem":
		itemCount := 0
		for i := 0; i < totalItems; i += 100 {
			keys := make([]map[string]types.AttributeValue, 0, 100)
			for j := i; j < min(i+100, totalItems); j++ {
				keys = append(keys, map[string]types.AttributeValue{
					"hk": &types.AttributeValueMemberS{Value: fmt.Sprintf("hashkey%d", j)},
					"rk": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", j)},
				})
			}
			items[itemCount] = &dynamodb.BatchGetItemInput{
				RequestItems: map[string]types.KeysAndAttributes{
					tableName: {
						Keys: keys,
					},
				},
			}
			itemCount++
		}
		return items[:itemCount], nil

	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}

	return items, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func cleanup(ctx context.Context, client *dynamodb.Client, tableName string) error {
	// Changed condition - delete if tableName is not empty and not a fixed table
	if tableName != "" && tableName != *table {
		log.Printf("Cleaning up table: %s", tableName)
		_, err := client.DeleteTable(ctx, &dynamodb.DeleteTableInput{
			TableName: aws.String(tableName),
		})
		if err != nil {
			log.Printf("Error deleting table %s: %v", tableName, err)
			return err
		}
		log.Printf("Successfully deleted table: %s", tableName)
	}
	return nil
}

func printResults(results []TestResult) {
	clientType := map[bool]string{true: "DynamoDB", false: "DAX"}[*useDDB]

	fmt.Printf("\n=== Performance Test Results ===\n")
	fmt.Printf("Client: %s\n", clientType)
	fmt.Printf("Duration: %d seconds\n", *duration)

	// Print detailed results
	fmt.Printf("\nDetailed Results:\n")
	fmt.Printf("%-12s %-8s %-12s %-12s %-12s %-10s %-10s %-10s %-10s %-10s\n",
		"Operation", "Size(B)", "Concurrency", "Throughput", "Errors/s",
		"p50 (μs)", "p99 (μs)", "p99.9 (μs)", "min (μs)", "max (μs)")
	fmt.Println(strings.Repeat("-", 120))

	for _, r := range results {
		fmt.Printf("%-12s %-8d %-12d %-12.2f %-12.2f %-10.2f %-10.2f %-10.2f %-10.2f %-10.2f\n",
			r.Operation,
			r.ItemSize,
			r.Concurrent,
			r.Throughput,
			r.Errors,
			r.P50,
			r.P99,
			r.P999,
			r.Min,
			r.Max,
		)
	}

	// Print operation-wise statistics
	fmt.Printf("\nOperation-wise Analysis:\n")
	opStats := make(map[string]struct {
		count                        int
		avgThroughput, maxThroughput float64
		avgP50, avgP99               float64
	})

	for _, r := range results {
		stats := opStats[r.Operation]
		stats.count++
		stats.avgThroughput += r.Throughput
		stats.avgP50 += r.P50
		stats.avgP99 += r.P99
		if r.Throughput > stats.maxThroughput {
			stats.maxThroughput = r.Throughput
		}
		opStats[r.Operation] = stats
	}

	fmt.Printf("\n%-12s %-15s %-15s %-12s %-12s\n",
		"Operation", "Avg Throughput", "Max Throughput", "Avg p50", "Avg p99")
	fmt.Println(strings.Repeat("-", 70))

	for op, stats := range opStats {
		fmt.Printf("%-12s %-15.2f %-15.2f %-12.2f %-12.2f\n",
			op,
			stats.avgThroughput/float64(stats.count),
			stats.maxThroughput,
			stats.avgP50/float64(stats.count),
			stats.avgP99/float64(stats.count),
		)
	}

	// Print scaling analysis
	fmt.Printf("\nConcurrency Scaling Analysis:\n")
	fmt.Printf("%-12s %-12s %-15s %-15s\n",
		"Operation", "Concurrency", "Avg Throughput", "Efficiency")
	fmt.Println(strings.Repeat("-", 60))

	baselineThroughput := make(map[string]float64)
	for _, r := range results {
		if r.Concurrent == 1 {
			if _, exists := baselineThroughput[r.Operation]; !exists {
				baselineThroughput[r.Operation] = r.Throughput
			}
		}
	}

	concurrencies := concurrencyLevels.DAX
	if *useDDB {
		concurrencies = concurrencyLevels.DDB
	}

	for op := range baselineThroughput {
		for _, concurrency := range concurrencies {
			var totalThroughput float64
			var count int
			for _, r := range results {
				if r.Operation == op && r.Concurrent == concurrency {
					totalThroughput += r.Throughput
					count++
				}
			}
			if count > 0 {
				avgThroughput := totalThroughput / float64(count)
				efficiency := avgThroughput / (float64(concurrency) * baselineThroughput[op]) * 100
				fmt.Printf("%-12s %-12d %-15.2f %-15.2f%%\n",
					op,
					concurrency,
					avgThroughput,
					efficiency,
				)
			}
		}
	}
}

func main() {
	flag.Parse()

	if *endpoint == "" && !*useDDB {
		log.Fatal("DAX endpoint is required when not using DynamoDB")
	}

	ctx := context.Background()

	// Initialize AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(*region))
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Initialize DynamoDB client
	ddbClient := dynamodb.NewFromConfig(cfg)

	// Initialize client based on flag
	var client DynamoClient
	if *useDDB {
		client = ddbClient
	} else {
		daxCfg := dax.DefaultConfig()
		daxCfg.HostPorts = []string{*endpoint}
		daxCfg.Region = *region
		daxCfg.WriteRetries = 3
		daxCfg.ReadRetries = 3

		daxClient, err := dax.New(daxCfg)
		if err != nil {
			log.Fatalf("Failed to create DAX client: %v", err)
		}
		defer daxClient.Close()
		client = daxClient
	}

	log.Printf("Starting performance test suite")
	log.Printf("Configuration:")
	log.Printf("- Client: %s", map[bool]string{true: "DynamoDB", false: "DAX"}[*useDDB])
	log.Printf("- Endpoint: %s", *endpoint)
	log.Printf("- Region: %s", *region)
	log.Printf("- Duration per test: %d seconds", *duration)
	log.Printf("- Total items: %d", *totalItems)
	log.Printf("- Operations: %v", operations)
	log.Printf("- Item sizes: %v bytes", itemSizes)
	log.Printf("- Concurrency levels: %v", concurrencyLevels.DAX)

	var allResults []TestResult
	startTime := time.Now()

	// Run tests for all combinations
	for _, operation := range operations {
		for _, itemSize := range itemSizes {
			for _, concurrency := range concurrencyLevels.DAX {
				// testStart := time.Now()
				log.Printf("\n========================================")
				log.Printf("Starting test iteration:")
				log.Printf("- Operation: %s", operation)
				log.Printf("- Item size: %d bytes", itemSize)
				log.Printf("- Concurrency: %d threads", concurrency)

				// Create table
				log.Printf("Creating/validating table...")
				tableName, err := createTable(ctx, ddbClient, operation, concurrency, itemSize, *totalItems)
				if err != nil {
					log.Printf("Failed to create table: %v", err)
					continue
				}
				log.Printf("Created table: %s", tableName)

				defer func(tn string) {
					if err := cleanup(ctx, ddbClient, tn); err != nil {
						log.Printf("Failed to cleanup table %s: %v", tn, err)
					}
				}(tableName)

				// Prepare test items
				log.Printf("Preparing test items...")
				reqs, err := prepareItems(ctx, client, tableName, itemSize, *totalItems, operation)
				if err != nil {
					log.Printf("Failed to prepare items: %v", err)
					if tableName == "" {
						cleanup(ctx, ddbClient, tableName)
					}
					continue
				}
				log.Printf("Prepared %d test items", len(reqs))

				// Run test with progress tracking
				log.Printf("Starting performance test for %d seconds...", *duration)
				testStart := time.Now()

				// Create context with timeout
				testCtx, cancel := context.WithTimeout(ctx, time.Duration(*duration)*time.Second)
				defer cancel()

				// Run the test
				totalResponse, totalErrors, latencies := doRun(testCtx, client,
					operation, reqs, concurrency, time.Duration(*duration)*time.Second)

				log.Printf("Test completed. Processing results...")

				// Calculate and log statistics
				log.Printf("\nTest completed. Calculating statistics...")
				histogram := NewHistogram(0.01, 60000000000.0)
				for _, latency := range latencies {
					histogram.Add(latency)
				}

				throughput := float64(totalResponse) / float64(*duration)
				errorRate := float64(totalErrors) / float64(*duration)

				log.Printf("Test Results:")
				log.Printf("- Total requests: %d", totalResponse)
				log.Printf("- Total errors: %d", totalErrors)
				log.Printf("- Throughput: %.2f ops/sec", throughput)
				log.Printf("- Error rate: %.2f errors/sec", errorRate)
				log.Printf("Latency (microseconds):")
				log.Printf("- p50: %.2f", histogram.Percentile(50.0))
				log.Printf("- p90: %.2f", histogram.Percentile(90.0))
				log.Printf("- p99: %.2f", histogram.Percentile(99.0))
				log.Printf("- p99.9: %.2f", histogram.Percentile(99.9))
				log.Printf("- min: %.2f", histogram.Minimum())
				log.Printf("- max: %.2f", histogram.Maximum())
				log.Printf("- mean: %.2f", histogram.Mean())
				log.Printf("- std dev: %.2f", histogram.StdDev())

				result := TestResult{
					Operation:  operation,
					ItemSize:   itemSize,
					Concurrent: concurrency,
					Throughput: throughput,
					Errors:     errorRate,
					P50:        histogram.Percentile(50.0),
					P99:        histogram.Percentile(99.0),
					P999:       histogram.Percentile(99.9),
					Min:        histogram.Minimum(),
					Max:        histogram.Maximum(),
				}

				allResults = append(allResults, result)

				// Cleanup
				if tableName == "" {
					log.Printf("Cleaning up test table...")
					cleanup(ctx, ddbClient, tableName)
				}

				testDuration := time.Since(testStart)
				log.Printf("Test iteration completed in %v", testDuration)
				log.Printf("========================================\n")
			}
		}
	}

	// Print final summary
	totalDuration := time.Since(startTime)
	log.Printf("\nTest suite completed in %v", totalDuration)
	log.Printf("Total test iterations: %d", len(allResults))

	// Print detailed results
	log.Printf("\nGenerating final report...")
	printResults(allResults)
}
