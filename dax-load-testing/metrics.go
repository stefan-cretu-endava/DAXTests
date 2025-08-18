package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type Counter[T number] struct {
	cwSender    *CloudWatchSender
	val         T
	name        string
	description string
	dimensions  []cwtypes.Dimension
}

func (c *Counter[T]) increment() {
	c.val += T(1)

	c.cwSender.Send(cwtypes.MetricDatum{
		MetricName: aws.String(c.name),
		Dimensions: c.dimensions,
		Timestamp:  aws.Time(time.Now()),
		Unit:       cwtypes.StandardUnitCount,
		Value:      aws.Float64(float64(c.val)),
	})
}

type Timer struct {
	cwSender    *CloudWatchSender
	name        string
	description string
	dimensions  []cwtypes.Dimension
}

func (t *Timer) record(val int64, unit cwtypes.StandardUnit) {
	t.cwSender.Send(cwtypes.MetricDatum{
		MetricName: aws.String(t.name),
		Dimensions: t.dimensions,
		Timestamp:  aws.Time(time.Now()),
		Unit:       unit,
		Value:      aws.Float64(float64(val)),
	})
}

func (t *Timer) stop(start time.Time) {
	t.record(time.Since(start).Microseconds(), cwtypes.StandardUnitMicroseconds)
}

type Metrics struct {
	cwSender *CloudWatchSender
	//private final MeterRegistry meterRegistry;
	//private final ConcurrentMap<String, Timer> timerCache = new ConcurrentHashMap<>();
	//private final ConcurrentMap<String, Counter> counterCache = new ConcurrentHashMap<>();
	//private final ConcurrentMap<String, Counter> throughput = new ConcurrentHashMap<>();
	timerCache   map[string]*Timer
	counterCache map[string]*Counter[int]
	throughput   map[string]*Counter[int]

	mtxTimerCache   sync.Mutex
	mtxCounterCache sync.Mutex
	mtxThroughput   sync.Mutex
}

func (m *Metrics) incrementThroughput(method string) {
	m.mtxThroughput.Lock()
	defer m.mtxThroughput.Unlock()

	if m.throughput == nil {
		m.throughput = map[string]*Counter[int]{}
	}

	if m.throughput[method] == nil {
		m.throughput[method] = &Counter[int]{
			name:        "app.request.throughput",
			description: "Request throughput",
			cwSender:    m.cwSender,
			dimensions: []cwtypes.Dimension{
				{
					Name:  aws.String("method"),
					Value: aws.String(method),
				},
			},
		}
	}

	m.throughput[method].increment()
}

func (m *Metrics) getOrCreateCounter(method, status string) *Counter[int] {
	m.mtxCounterCache.Lock()
	defer m.mtxCounterCache.Unlock()

	k := fmt.Sprintf("%s-%s", method, status)

	if m.counterCache == nil {
		m.counterCache = map[string]*Counter[int]{}
	}

	if m.counterCache[k] == nil {
		m.counterCache[k] = &Counter[int]{
			name:        "app.request.count",
			description: "Request count per method",
			cwSender:    m.cwSender,
			dimensions: []cwtypes.Dimension{
				{
					Name:  aws.String("method"),
					Value: aws.String(method),
				},
				{
					Name:  aws.String("status"),
					Value: aws.String(status),
				},
			},
		}
	}

	return m.counterCache[k]
}

func (m *Metrics) getOrCreateTimer(method, status string) *Timer {
	m.mtxTimerCache.Lock()
	defer m.mtxTimerCache.Unlock()

	if m.timerCache == nil {
		m.timerCache = map[string]*Timer{}
	}

	k := fmt.Sprintf("%s-%s", method, status)
	if m.timerCache[k] == nil {
		m.timerCache[k] = &Timer{
			name:        "app.request.latency",
			description: "Request latency per method",
			cwSender:    m.cwSender,
			dimensions: []cwtypes.Dimension{
				{
					Name:  aws.String("method"),
					Value: aws.String(method),
				},
				{
					Name:  aws.String("status"),
					Value: aws.String(status),
				},
			},
		}
	}

	return m.timerCache[k]
}

func (m *Metrics) recordLatency(method string, fn func() error) {
	start := time.Now()
	res := fn()
	timer := m.getOrCreateTimer(method, ternary(res == nil, "success", "error"))
	timer.stop(start)
}

func (m *Metrics) incrementStatusCounter(method string, statusCode int) {
	status := m.getStatusGroup(statusCode)
	counter := m.getOrCreateCounter(method, status)
	counter.increment()
}

func (m *Metrics) getStatusGroup(status int) string {
	out := fmt.Sprintf("%d", status)
	if len(out) == 3 {
		switch out[0] {
		case '2', '4', '5':
			return fmt.Sprintf("%sxx", out[0:1])
		}
	}

	return "other"
}
