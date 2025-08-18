package main

import "math"

type Histogram struct {
	resolution   float64
	multiplier   float64
	sum          float64
	count        int64
	maxValue     float64
	minSeen      float64
	maxSeen      float64
	buckets      []int64
	sumOfSquares float64
}

func NewHistogram(resolution, maxValue float64) *Histogram {
	h := &Histogram{
		resolution: resolution,
		multiplier: 1 + resolution,
		maxValue:   maxValue,
		minSeen:    maxValue,
		maxSeen:    0,
	}

	bucketCount := 1 + h.bucketFor(maxValue)
	h.buckets = make([]int64, bucketCount)
	return h
}

func (h *Histogram) bucketFor(value float64) int {
	return int(math.Floor(math.Log(value) / math.Log(h.multiplier)))
}

func (h *Histogram) bucketStart(index int) float64 {
	return math.Pow(h.multiplier, float64(index))
}

func (h *Histogram) Add(value float64) {
	h.sum += value
	h.sumOfSquares += value * value
	h.count++

	if value < h.minSeen {
		h.minSeen = value
	}
	if value > h.maxSeen {
		h.maxSeen = value
	}

	h.buckets[h.bucketFor(value)]++
}

func (h *Histogram) Mean() float64 {
	return h.sum / float64(h.count)
}

func (h *Histogram) Variance() float64 {
	if h.count == 0 {
		return 0
	}
	return (h.sumOfSquares*float64(h.count) - h.sum*h.sum) / float64(h.count*h.count)
}

func (h *Histogram) StdDev() float64 {
	return math.Sqrt(h.Variance())
}

func (h *Histogram) Maximum() float64 {
	return h.maxSeen
}

func (h *Histogram) Minimum() float64 {
	return h.minSeen
}

func (h *Histogram) GetCount() int64 {
	return h.count
}

func (h *Histogram) GetSum() float64 {
	return h.sum
}

func (h *Histogram) Percentile(pct float64) float64 {
	return h.ThresholdBelowCount(float64(h.count) * pct / 100.0)
}

func (h *Histogram) ThresholdBelowCount(cnt float64) float64 {
	if h.count <= 0 {
		return 0.0
	}

	if cnt <= 0 {
		return h.minSeen
	}
	if cnt >= float64(h.count) {
		return h.maxSeen
	}

	var soFar float64
	var loIdx int

	for loIdx = 0; loIdx < len(h.buckets); loIdx++ {
		soFar += float64(h.buckets[loIdx])
		if soFar >= cnt {
			break
		}
	}

	if soFar == cnt {
		hiIdx := loIdx + 1
		for ; hiIdx < len(h.buckets); hiIdx++ {
			if h.buckets[hiIdx] > 0 {
				break
			}
		}
		return (h.bucketStart(loIdx) + h.bucketStart(hiIdx)) / 2.0
	}

	lo := h.bucketStart(loIdx)
	hi := h.bucketStart(loIdx + 1)

	return hi - (hi-lo)*(soFar-cnt)/float64(h.buckets[loIdx])
}
