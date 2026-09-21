package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func getGOMAXPROCS() int {
	return runtime.GOMAXPROCS(0)
}

func main() {
	flags := getFlags()
	appConfig := getAppConfig(flags)

	sigCtx, sigCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer sigCancel()
	_ = sigCtx

	awsCfg := getAwsConfig()

	cw := getCloudwatch(awsCfg)
	daxSvc := getDaxSvc(awsCfg)

	fmt.Printf("Connected to cluster with name:%s\n in region:%s\n available at:%s\n having #nodes:%d\n using table:%s\n and logging metrics to CW namespace:%s\n test name:%s\n GOMAXPROCS is %d\n",
		flags.clusterName,
		awsCfg.Region,
		flags.clusterEndpoint,
		flags.nodes,
		appConfig.Table,
		flags.testNamespace,
		appConfig.TestConfig.Name,
		getGOMAXPROCS())

	daxClient, err := getDaxClient(awsCfg, flags, appConfig)
	if err != nil {
		panic(err)
	}

	/*
		if !isWriteOp(flags.op) {
			if flags.op == "read" || flags.op == "Read" {
				loadDataForRead(sigCtx, daxClient, appConfig.Table, appConfig.TrafficConfig.ItemSizes["GetItem"])
				loadDataForRead(sigCtx, daxClient, appConfig.Table, appConfig.TrafficConfig.ItemSizes["Query"])
				loadDataForRead(sigCtx, daxClient, appConfig.Table, appConfig.TrafficConfig.ItemSizes["BatchGetItem"])
			} else {
				fmt.Println("Load data for operation:", flags.op)
				loadDataForRead(sigCtx, daxClient, appConfig.Table, appConfig.TrafficConfig.ItemSizes[flags.op])
			}
		}*/

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

	run(sigCtx, metricChan, daxClient, appConfig, flags)
	daxClient.Close()
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

func run(ctx context.Context, metricChan chan types.MetricDatum, client *dax.Dax, appConfig *AppConfig, f *flags) {
	// Each launched worker has a dedicated context with a cancel function kept in the list below, such that it can be cancelled on demand
	var cancelFuncs []context.CancelFunc
	// Used to measure the request count every minute time interval
	ticker := time.NewTicker(10 * time.Second)
	// Workers use this chan to send true if the request returned an error containing ThrottlingException
	throttleChan := make(chan bool)
	// A factor which helps in mitigating throttling, being increased in such cases, thus artifically increase the request count by it
	loadBias := 1.0

	// Record the request count in the previous minute time interval
	var lastReqCount uint64
	// Average load per minute
	var avgLoadPerMinute float64
	tableName := appConfig.Table

	var maxIncrease int = 4
	var startingWorkersCount int = 16

	var trafficFunc func(ctx context.Context)
	var workerObj Worker = Worker{
		metricChan:          metricChan,
		client:              client,
		tableName:           tableName,
		appConfig:           appConfig,
		throttleChan:        throttleChan,
		opsCntr:             nil,
		targetAvgLoadPerMin: 1_040_000,
	}

	switch f.op {
	case "GetItem":
		maxIncrease = 4
		startingWorkersCount = 16
		fmt.Println("Starting GetItem test targetting avg load/min:", workerObj.targetAvgLoadPerMin)
		trafficFunc = workerObj.GetItemTraffic

	case "BatchGetItem":
		maxIncrease = 4
		startingWorkersCount = 4
		fmt.Println("Starting BatchGetItem test targetting avg load/min:", workerObj.targetAvgLoadPerMin)
		trafficFunc = workerObj.BatchGetItemTraffic

	case "Query":
		maxIncrease = 4
		startingWorkersCount = 4
		workerObj.targetAvgLoadPerMin = 450_000
		fmt.Println("Starting Query test targetting avg load/min:", workerObj.targetAvgLoadPerMin)
		trafficFunc = workerObj.QueryTraffic

	case "PutItem":
		maxIncrease = 4
		startingWorkersCount = 4
		workerObj.targetAvgLoadPerMin = 300_000
		if f.requestTimeoutMillis <= 150 {
			maxIncrease = 1
			startingWorkersCount = 2
			workerObj.targetAvgLoadPerMin = 80_000
		}
		fmt.Println("Starting PutItem test targetting avg load/min:", workerObj.targetAvgLoadPerMin, "maxIncrease", maxIncrease, "startingWorkersCount", startingWorkersCount)
		trafficFunc = workerObj.PutItemTraffic

	case "UpdateItem":
		maxIncrease = 4
		startingWorkersCount = 4
		workerObj.targetAvgLoadPerMin = 300_000
		if f.requestTimeoutMillis <= 150 {
			maxIncrease = 1
			startingWorkersCount = 2
			workerObj.targetAvgLoadPerMin = 100_000
		}
		fmt.Println("Starting UpdateItem test targeting avg load/min:", workerObj.targetAvgLoadPerMin, "maxIncrease", maxIncrease, "startingWorkersCount", startingWorkersCount)
		trafficFunc = workerObj.UpdateItemTraffic

	case "BatchWriteItem":
		maxIncrease = 2
		startingWorkersCount = 2
		workerObj.targetAvgLoadPerMin = 300_000
		if f.requestTimeoutMillis <= 150 {
			maxIncrease = 1
			startingWorkersCount = 1
			workerObj.targetAvgLoadPerMin = 30_000
		}
		fmt.Println("Starting BatchWriteItem test targeting avg load/min:", workerObj.targetAvgLoadPerMin, "maxIncrease", maxIncrease, "startingWorkersCount", startingWorkersCount)
		trafficFunc = workerObj.BatchWriteItemTraffic

	case "read", "Read":
		maxIncrease = 4
		startingWorkersCount = 4
		// workerObj.opsCntr = &OperationsCounter{
		// 	getItemPercentage:      50,
		// 	batchGetItemPercentage: 50,
		// 	queryPercentage:        0,
		// }
		// fmt.Println("Read operations percentages. GetItem:", workerObj.opsCntr.getItemPercentage, " BatchGetItem:", workerObj.opsCntr.batchGetItemPercentage, " Query:", workerObj.opsCntr.queryPercentage)
		// workerObj.opsCntr.scaleToAvgLoadperMinute(workerObj.targetAvgLoadPerMin)
		// fmt.Println("Scaled percentages to target avg load/min:", workerObj.targetAvgLoadPerMin, "GetItem:", workerObj.opsCntr.getItemPercentage, " BatchGetItem:", workerObj.opsCntr.batchGetItemPercentage, " Query:", workerObj.opsCntr.queryPercentage)
		fmt.Println("Starting read test targetting avg load/min:", workerObj.targetAvgLoadPerMin, "maxIncrease", maxIncrease, "startingWorkersCount", startingWorkersCount)
		trafficFunc = workerObj.ReadTraffic

	case "write", "Write":
		maxIncrease = 4
		startingWorkersCount = 4
		workerObj.targetAvgLoadPerMin = 300_000

		if f.requestTimeoutMillis <= 150 {
			maxIncrease = 1
			startingWorkersCount = 2
			workerObj.targetAvgLoadPerMin = 40_000
		}
		// workerObj.opsCntr = &OperationsCounter{
		// 	putItemPercentage:        50,
		// 	batchWriteItemPercentage: 50,
		// 	updateItemPercentage:     0,
		// }
		// fmt.Println("Read operations percentages. GetItem:", workerObj.opsCntr.putItemPercentage, " BatchGetItem:", workerObj.opsCntr.batchWriteItemPercentage, " Query:", workerObj.opsCntr.updateItemPercentage)
		// workerObj.opsCntr.scaleToAvgLoadperMinute(workerObj.targetAvgLoadPerMin)
		// fmt.Println("Scaled percentages to target avg load/min:", workerObj.targetAvgLoadPerMin, "GetItem:", workerObj.opsCntr.putItemPercentage, " BatchGetItem:", workerObj.opsCntr.batchWriteItemPercentage, " Query:", workerObj.opsCntr.updateItemPercentage)
		fmt.Println("Starting write test targetting avg load/min:", workerObj.targetAvgLoadPerMin, "maxIncrease", maxIncrease, "startingWorkersCount", startingWorkersCount)
		trafficFunc = workerObj.WriteTraffic

	case "read-write", "Read-Write", "readwrite", "ReadWrite":
		maxIncrease = 1
		startingWorkersCount = 2
		workerObj.targetAvgLoadPerMin = 300_000

		if f.requestTimeoutMillis <= 150 {
			workerObj.targetAvgLoadPerMin = 300_000
		}
		// workerObj.opsCntr = &OperationsCounter{
		// 	getItemPercentage: 50,
		// 	putItemPercentage: 50,
		// }
		// fmt.Println("Read operations percentages. GetItem:", workerObj.opsCntr.getItemPercentage, " PutItem:", workerObj.opsCntr.putItemPercentage)
		// workerObj.opsCntr.scaleToAvgLoadperMinute(workerObj.targetAvgLoadPerMin)
		// fmt.Println("Scaled percentages to target avg load/min:", workerObj.targetAvgLoadPerMin, "GetItem:", workerObj.opsCntr.getItemPercentage, " PutItem:", workerObj.opsCntr.putItemPercentage)
		fmt.Println("Starting read-write test targetting avg load/min:", workerObj.targetAvgLoadPerMin)
		trafficFunc = workerObj.ReadWriteTraffic

	default:
		maxIncrease = 4
		startingWorkersCount = 16
		trafficFunc = workerObj.GetItemTraffic
	}

	for range startingWorkersCount {
		nCtx, nCancel := context.WithCancel(context.Background())
		cancelFuncs = append(cancelFuncs, nCancel)

		go trafficFunc(nCtx)
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
			// reqCount is incremented with every made request during the entire testing time interval
			currentReqCount := atomic.LoadUint64(&reqCount)
			// For determining the request count made during the last interval, substract the old counter value,
			// which contains the number of all requests made until the current time interval
			reqCountDiff := currentReqCount - lastReqCount
			lastReqCount = currentReqCount

			avgLoadPer10Seconds := float64(reqCountDiff)
			avgLoadPerMinute = 6 * avgLoadPer10Seconds

			log.Printf("workers=%d goroutines=%d", len(cancelFuncs), runtime.NumGoroutine())
		}

		if time.Now().Unix()-lastBiasChange > 60 {
			if loadBias > 1.0 {
				loadBias /= 1.1
			}
			if loadBias < 1.0 {
				loadBias = 1.0
			}
		}

		log.Printf("Average total request count per minute: %08f", avgLoadPerMinute)

		// Keep track of the target requests count, which can be artifically changed by loadBias factor  to alleviate throttling issues
		targetAvgLoadPerMinute := float64(workerObj.targetAvgLoadPerMin)
		avgLoadPerMinute = avgLoadPerMinute * loadBias

		log.Printf("Average total request count per minute with load bias: %08f, %08f", avgLoadPerMinute, loadBias)

		if loadBias > 1.0 {
			log.Printf("Total request count per minute | target requests count with loadBias: %08f | %08f", avgLoadPerMinute, targetAvgLoadPerMinute)
		} else {
			targetAvgLoadPerMinute = float64(workerObj.targetAvgLoadPerMin)
		}

		cancelFuncsCount := len(cancelFuncs)
		if cancelFuncsCount == 0 {
			continue
		}
		loadPerGoroutine := avgLoadPerMinute / float64(cancelFuncsCount)

		if lessThanOrEqualFloat64(avgLoadPerMinute, targetAvgLoadPerMinute) {
			log.Printf("avgLoadPerMinute:%08f | cancelFuncsCount=Num goroutines: %d | Average load per goroutine: %08f", avgLoadPerMinute, len(cancelFuncs), loadPerGoroutine)

			extraGoroutinesCount := int((targetAvgLoadPerMinute - avgLoadPerMinute) / loadPerGoroutine)

			if extraGoroutinesCount > maxIncrease {
				fmt.Println("Required increase of goroutines exceeds maxIncrease, limiting to:", extraGoroutinesCount, maxIncrease)
				extraGoroutinesCount = maxIncrease
			}

			log.Printf("Will start %d goroutine(s)", extraGoroutinesCount)
			for range extraGoroutinesCount {
				nCtx, nCancel := context.WithCancel(context.Background())
				cancelFuncs = append(cancelFuncs, nCancel)

				<-time.After(time.Millisecond)
				go trafficFunc(nCtx)
			}
		} else {
			if len(cancelFuncs) == 0 {
				panic(fmt.Sprintf("Load with zero goroutines!, avg: %f", avgLoadPerMinute))
			}

			sAvg := int((avgLoadPerMinute - targetAvgLoadPerMinute) / loadPerGoroutine)
			stoppingGoroutinesCount := sAvg

			log.Printf("Will stop %d goroutine(s)", stoppingGoroutinesCount)
			for range stoppingGoroutinesCount {
				if len(cancelFuncs) > 0 {
					cancelFuncs[0]()
					cancelFuncs = cancelFuncs[1:]
				}
			}
		}

	}
}
