package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func main() {
	flags := getFlags()
	appConfig := getAppConfig(flags)

	sigCtx, sigCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer sigCancel()
	_ = sigCtx

	awsCfg := getAwsConfig()

	cw := getCloudwatch(awsCfg)
	daxSvc := getDaxSvc(awsCfg)

	fmt.Printf("New cluster with name:%s available at:%s Table:%s CW namespace:%s", flags.clusterName, flags.clusterEndpoint, appConfig.Table, flags.testNamespace)

	daxClient, err := getDaxClient(awsCfg, flags.clusterEndpoint, appConfig)
	if err != nil {
		panic(err)
	}

	if !isWriteOp(flags.op) {
		if flags.op == "read" || flags.op == "Read" {
			loadDataForRead(sigCtx, daxClient, appConfig.Table, appConfig.TrafficConfig.ItemSizes["GetItem"])
			loadDataForRead(sigCtx, daxClient, appConfig.Table, appConfig.TrafficConfig.ItemSizes["Query"])
			loadDataForRead(sigCtx, daxClient, appConfig.Table, appConfig.TrafficConfig.ItemSizes["BatchGetItem"])
		} else {
			fmt.Println("Load data for operation:", flags.op)
			loadDataForRead(sigCtx, daxClient, appConfig.Table, appConfig.TrafficConfig.ItemSizes[flags.op])
		}
	}

	go func() {
		<-time.After(time.Duration(flags.testDurationMinutes) * time.Minute)
		log.Println("Testing time limit reached!", time.Duration(flags.testDurationMinutes)*time.Minute)
		sigCancel()
	}()

	metricChan := make(chan types.MetricDatum, 500_000)

	go runMetricCollector(sigCtx, cw, metricChan, flags)

	if appConfig.TestConfig.Reboot > 0 {
		log.Printf("Will reboot random node once every %d ms", appConfig.TestConfig.Reboot)

		go func() {
			ticker := time.NewTicker(time.Millisecond * time.Duration(appConfig.TestConfig.Reboot))
			for {
				select {
				case <-sigCtx.Done():
					return
				case <-ticker.C:
					log.Println("Rebooting random node")
					daxSvc.RebootRandomNode(flags.clusterName)
					log.Println("Reboot command sent")
				}
			}
		}()
	}

	run(sigCtx, cw, metricChan, daxClient, appConfig, flags)
}

func collectMetricWorker(ctx context.Context, cw *cloudwatch.Client, metricChan chan types.MetricDatum, f *flags) {
	metricData := make([]types.MetricDatum, 0, 500)

	for shouldExit := false; !shouldExit; {
		shouldSend := false

		select {
		case <-ctx.Done():
			shouldExit = true
			log.Println("[CANCEL] Forcing send of:", len(metricData))
		case <-time.After(time.Second * 10):
			shouldSend = true
			log.Println("[TIMEOUT] Forcing send of:", len(metricData))
			break
		case md, ok := <-metricChan:
			if ok {
				md.Dimensions = []types.Dimension{
					{
						Name:  aws.String("Test"),
						Value: aws.String(f.test),
					},
					{
						Name:  aws.String("Method"),
						Value: aws.String(f.op),
					},
				}
				metricData = append(metricData, md)
			}
		}

		shouldSend = shouldSend || len(metricData) == 500 || shouldExit

		if shouldSend && len(metricData) > 0 {
			_, _ = cw.PutMetricData(context.Background(), &cloudwatch.PutMetricDataInput{
				Namespace:  aws.String(f.testNamespace),
				MetricData: metricData,
			})
			metricData = make([]types.MetricDatum, 0, 500)
		}
	}
}

func runMetricCollector(ctx context.Context, cw *cloudwatch.Client, metricChan chan types.MetricDatum, f *flags) {
	var cancelFuncs []context.CancelFunc

	for {
		select {
		case <-ctx.Done():
			for c := range cancelFuncs {
				cancelFuncs[c]()
			}
			return
		case <-time.After(time.Second):
		}

		currentQueueSize := len(metricChan)
		actual := len(cancelFuncs)
		expected := currentQueueSize / 1000
		if actual < 500 && currentQueueSize > 1000 {
			toStart := min(500-actual, expected, 500)

			// might be redundant, but let's be extra safe
			if toStart+actual > 500 {
				toStart = 500 - actual
			}

			log.Printf("Starting %d metric workers", toStart)
			for range toStart {
				nCtx, nCancel := context.WithCancel(context.Background())
				cancelFuncs = append(cancelFuncs, nCancel)

				go collectMetricWorker(nCtx, cw, metricChan, f)
			}
		} else if actual > 1 && currentQueueSize < 1000 {
			cancelFuncs[0]()
			cancelFuncs = cancelFuncs[1:]
		}
	}
}

// func chooseWorkerFunc(f *flags) workerFn {
// 	var worker workerFn
// 	r := NewRandom[int](100, 0)
// 	rnd := r.Next()

// 	if f.op == "write" {
// 		if rnd < 75 {
// 			worker = workerPutItem
// 		} else if rnd < 90 {
// 			worker = workerUpdateItem
// 		} else {
// 			worker = workerBatchWriteItem
// 		}
// 	} else {
// 		if rnd < 75 {
// 			worker = workerGetItem
// 		} else if rnd < 90 {
// 			worker = workerQuery
// 		} else {
// 			worker = workerBatchGetItem
// 		}
// 	}

// 	return worker
// }

func run(ctx context.Context, cw *cloudwatch.Client, metricChan chan types.MetricDatum, client *dax.Dax, appConfig *AppConfig, f *flags) {
	var cancelFuncs []context.CancelFunc
	ticker := time.NewTicker(time.Minute)
	throttleChan := make(chan bool)
	loadBias := 1.0

	tableName := appConfig.Table

	maxIncrease := 8
	if isWriteOp(f.op) {
		maxIncrease = 2
	}

	var worker workerFn
	switch f.op {
	case "GetItem":
		worker = workerGetItem
	case "BatchGetItem":
		worker = workerBatchGetItem
	case "Query":
		worker = workerQuery
	case "PutItem":
		worker = workerPutItem
	case "UpdateItem":
		worker = workerUpdateItem
	case "BatchWriteItem":
		worker = workerBatchWriteItem
	case "read", "Read":
		worker = workerRead
	default:
		worker = workerWrite
	}

	for range 16 {
		nCtx, nCancel := context.WithCancel(context.Background())
		cancelFuncs = append(cancelFuncs, nCancel)

		go worker(nCtx, metricChan, client, tableName, appConfig, throttleChan)
	}

	lastBiasChange := time.Now().Unix() - 3600

	for {
		select {
		case <-throttleChan:
			if time.Now().Unix()-lastBiasChange > 5 {
				loadBias *= 1.1
				// allow loadBias change once per minute
				lastBiasChange = time.Now().Unix()
				log.Println("Throttle detected, increased load loadBias to:", loadBias)
				if len(cancelFuncs) > 0 {
					cancelFuncs[0]()
					cancelFuncs = cancelFuncs[1:]
				}
			}
			continue
		case <-ctx.Done():
			for c := range cancelFuncs {
				cancelFuncs[c]()
			}

			fmt.Println("CTRL+C pressed")
			return

		case <-ticker.C:
			//
		}

		if time.Now().Unix()-lastBiasChange > 60 {
			if loadBias > 1.0 {
				loadBias /= 1.1
			}
			if loadBias < 1.0 {
				loadBias = 1.0
			}
		}

		statAvg := getLastMinuteStats(cw, f.clusterName, types.StatisticAverage)
		if statAvg == nil {
			log.Println("Failed to get last minute stats")
			continue
		}
		avgStatValue := *statAvg.Datapoints[0].Average

		statMax := getLastMinuteStats(cw, f.clusterName, types.StatisticMaximum)
		if statMax == nil {
			log.Println("Failed to get last minute stats")
			continue
		}
		maxStatValue := *statMax.Datapoints[0].Maximum
		log.Printf("Average total request count: %08f, Max total request count: %08f", avgStatValue, maxStatValue)

		targetAvgTotalRequestCount := 1_000_000.0
		avgStatValue = avgStatValue * loadBias
		if loadBias > 1.0 {
			targetAvgTotalRequestCount *= loadBias
			log.Printf("Total request count | target requests count with loadBias: %08f | %08f", avgStatValue, targetAvgTotalRequestCount)
		} else {
			targetAvgTotalRequestCount = 1_000_000.0
		}

		cancelFuncsCount := len(cancelFuncs)
		loadPerGoroutine := avgStatValue / float64(cancelFuncsCount)

		if lessThanOrEqualFloat64(avgStatValue, targetAvgTotalRequestCount) {
			log.Printf("Num goroutines: %d", len(cancelFuncs))
			log.Printf("Average load per goroutine: %08f", loadPerGoroutine)

			sAvg := int((targetAvgTotalRequestCount - avgStatValue) / loadPerGoroutine)
			s := sAvg
			if s == 0 {
				s = 1
			}
			if s > maxIncrease {
				s = maxIncrease
			}

			log.Printf("Will start %d goroutine(s)", s)
			for range s {
				nCtx, nCancel := context.WithCancel(context.Background())
				cancelFuncs = append(cancelFuncs, nCancel)

				<-time.After(time.Millisecond)
				go worker(nCtx, metricChan, client, tableName, appConfig, throttleChan)
			}
		} else {
			if len(cancelFuncs) == 0 {
				panic(fmt.Sprintf("Load with zero goroutines!, avg: %f, max: %f", avgStatValue, maxStatValue))
			}

			sAvg := int((avgStatValue - targetAvgTotalRequestCount) / loadPerGoroutine)
			s := sAvg
			if s == 0 {
				s = int((avgStatValue - targetAvgTotalRequestCount) / loadPerGoroutine) //min(int(avgStatValue-targetAvgTotalRequestCount), int(maxStatValue-targetMaxTotalrequestCount))
			}

			log.Printf("Will stop %d goroutine(s)", s)
			for range s {
				if len(cancelFuncs) > 0 {
					cancelFuncs[0]()
					cancelFuncs = cancelFuncs[1:]
				}
			}
		}

	}
}
