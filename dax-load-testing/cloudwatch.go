package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

func getCloudwatch(cfg *aws.Config) *cloudwatch.Client {
	return cloudwatch.NewFromConfig(*cfg)
}

type CloudWatchSender struct {
	client      *cloudwatch.Client
	buffer      chan cwtypes.MetricDatum
	cancelFuncs []context.CancelFunc
	cancel      context.CancelFunc
}

func (c *CloudWatchSender) Send(md cwtypes.MetricDatum) {
	if c.buffer == nil {
		panic("buffer is nil?")
	}

	select {
	case c.buffer <- md:
		//
	default:
		log.Print("[WARNING] UNABLE TO SEND METRIC! CHANNEL FULL?")
		log.Print(len(c.buffer))
	}
}

func (c *CloudWatchSender) Start() {
	if c.cancel != nil {
		panic("already started")
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	// start an initial set of workers
	for range 256 {
		nCtx, nCancel := context.WithCancel(context.Background())
		c.cancelFuncs = append(c.cancelFuncs, nCancel)

		go collectMetricWorker(nCtx, c.client, c.buffer)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				for i := range c.cancelFuncs {
					c.cancelFuncs[i]()
				}
				c.cancelFuncs = []context.CancelFunc{}
				return
			case <-time.After(time.Minute * time.Duration(Flags.App.CloudWatch.PushFrequencyMinutes)):
				break
			}

			currentQueueSize := len(c.buffer)
			actual := len(c.cancelFuncs)
			expected := currentQueueSize / 1000

			if actual < 500 && currentQueueSize > 1000 {
				toStart := min(500-actual, expected, 500)

				if toStart+actual > 500 {
					toStart = 500 - actual
				}

				log.Printf("Starting %d metric workers", toStart)
				for range toStart {
					nCtx, nCancel := context.WithCancel(context.Background())
					c.cancelFuncs = append(c.cancelFuncs, nCancel)

					go collectMetricWorker(nCtx, c.client, c.buffer)
				}
			} else if actual > 1 && currentQueueSize < 1000 {
				c.cancelFuncs[0]()
				c.cancelFuncs = c.cancelFuncs[1:]
			}
		}
	}()
}

func collectMetricWorkerStats(ctx context.Context, cw *cloudwatch.Client, metricChan chan cwtypes.MetricDatum) {
	mds := map[string]cwtypes.MetricDatum{}

	ticker := time.NewTicker(time.Minute * time.Duration(Flags.App.CloudWatch.PushFrequencyMinutes))

	for shouldExit := false; !shouldExit; {
		shouldSend := false
		select {
		case <-ctx.Done():
			shouldExit = true
		default:
			//
		}

		select {
		case <-ticker.C:
			shouldSend = true
		case md, ok := <-metricChan:
			if ok {
				keyParts := []string{
					aws.ToString(md.MetricName),
				}
				for _, d := range md.Dimensions {
					keyParts = append(keyParts, aws.ToString(d.Value))
				}

				key := strings.Join(keyParts, "-")

				if dtm, ok := mds[key]; ok {
					//if idx := slices.Index(dtm.Values, aws.ToFloat64(md.Value)); idx != -1 {
					//	dtm.Counts[idx] = dtm.Counts[idx] + 1
					//} else {
					//	dtm.Counts = append(dtm.Counts, 1)
					//	dtm.Values = append(dtm.Values, aws.ToFloat64(md.Value))
					//}
					//
					//if len(dtm.Values) == 150 {
					//	shouldSend = true
					//}

					dtm.StatisticValues.SampleCount = aws.Float64(
						aws.ToFloat64(dtm.StatisticValues.SampleCount) + float64(1),
					)
					dtm.StatisticValues.Sum = aws.Float64(
						aws.ToFloat64(dtm.StatisticValues.Sum) + aws.ToFloat64(md.Value),
					)
					dtm.StatisticValues.Maximum = aws.Float64(
						max(
							aws.ToFloat64(dtm.StatisticValues.Maximum),
							aws.ToFloat64(md.Value),
						),
					)
					if dtm.StatisticValues.Minimum == nil {
						dtm.StatisticValues.Minimum = aws.Float64(aws.ToFloat64(md.Value))
					} else {
						dtm.StatisticValues.Minimum = aws.Float64(
							min(
								aws.ToFloat64(dtm.StatisticValues.Minimum),
								aws.ToFloat64(md.Value),
							),
						)
					}
					mds[key] = dtm
				} else {
					dtmn := cwtypes.MetricDatum{
						MetricName:      md.MetricName,
						Dimensions:      md.Dimensions,
						StatisticValues: &cwtypes.StatisticSet{},
						Timestamp:       md.Timestamp,
						Unit:            md.Unit,
						//Value:           nil,
						//Values:          []float64{aws.ToFloat64(md.Value)},
						//Counts:          []float64{1},
					}

					mds[key] = dtmn
				}
			}
		}

		shouldSend = shouldSend || len(mds) == 500 || shouldExit

		if shouldSend && len(mds) > 0 {
			metricData := make([]cwtypes.MetricDatum, 0, len(mds))
			for k, md := range mds {
				metricData = append(metricData, md)
				log.Printf("\t[%s] %s - %#+v [%d]", k, aws.ToString(md.MetricName), md.Dimensions, len(md.Dimensions))
			}
			log.Printf("[AGGREGATED] Sending %d metrics", len(metricData))
			_, _ = cw.PutMetricData(context.Background(), &cloudwatch.PutMetricDataInput{
				Namespace:  aws.String(Flags.App.CloudWatch.Namespace),
				MetricData: metricData,
			})

			mds = map[string]cwtypes.MetricDatum{}
		}
	}
}

func collectMetricWorker(ctx context.Context, cw *cloudwatch.Client, metricChan chan cwtypes.MetricDatum) {
	metricData := make([]cwtypes.MetricDatum, 0, 500)

	for shouldExit := false; !shouldExit; {
		shouldSend := false

		select {
		case <-ctx.Done():
			shouldExit = true
			//log.Println("[CANCEL] Forcing send of:", len(metricData))
		case <-time.After(time.Second * 10):
			shouldSend = true
			//log.Println("[TIMEOUT] Forcing send of:", len(metricData))
			break
		case md, ok := <-metricChan:
			if ok {
				metricData = append(metricData, md)
			}
		}

		shouldSend = shouldSend || len(metricData) == 500 || shouldExit

		if shouldSend && len(metricData) > 0 {
			//log.Printf("[NORMAL] Sending %d metrics", len(metricData))
			_, _ = cw.PutMetricData(context.Background(), &cloudwatch.PutMetricDataInput{
				Namespace:  aws.String(Flags.App.CloudWatch.Namespace),
				MetricData: metricData,
			})
			metricData = make([]cwtypes.MetricDatum, 0, 500)
		}
	}
}

func (c *CloudWatchSender) Stop() {
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
}

func NewCloudWatchSender(client *cloudwatch.Client) *CloudWatchSender {
	return &CloudWatchSender{
		client: client,
		buffer: make(chan cwtypes.MetricDatum, 1024*1024),
	}
}
