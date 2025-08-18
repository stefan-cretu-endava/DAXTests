package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

type LoadGenFunc func() error

type LoadGen struct {
	fn          LoadGenFunc
	duration    time.Duration
	concurrency int
	startTime   time.Time
	endTime     time.Time
}

func NewLoadGen(fn LoadGenFunc, duration time.Duration, concurrency int) *LoadGen {
	return &LoadGen{
		fn:          fn,
		duration:    duration,
		concurrency: concurrency,
	}
}

func (lg *LoadGen) Start(ctx context.Context) error {
	lg.startTime = time.Now()
	lg.endTime = lg.startTime.Add(lg.duration)

	var wg sync.WaitGroup
	errChan := make(chan error, lg.concurrency)

	// Track errors with mutex protection
	var (
		errorsMutex sync.Mutex
		errors      []error
	)

	// Create a context with grace period for cleanup
	ctxWithGrace, cancelGrace := context.WithTimeout(context.Background(),
		lg.duration+30*time.Second) // 30 second grace period
	defer cancelGrace()

	// Launch worker goroutines
	for i := 0; i < lg.concurrency; i++ {
		wg.Add(1)
		workerID := i

		go func(id int) {
			defer wg.Done()
			// log.Printf("Worker %d started", id)
			requestCount := 0
			errorCount := 0
			maxRetries := 3
			backoffBase := 500 * time.Millisecond

			for {
				// Check primary test duration
				if time.Now().After(lg.endTime) {
					// log.Printf("Worker %d completed: processed %d requests (%d errors)",
					// 	id, requestCount, errorCount)
					return
				}

				// Use grace period context for the actual operation
				select {
				case <-ctxWithGrace.Done():
					// log.Printf("Worker %d stopped during grace period after %d requests (%d errors)",
					// 	id, requestCount, errorCount)
					return
				default:
					// Retry loop for each request
					var lastErr error
					success := false

					for attempt := 0; attempt < maxRetries; attempt++ {
						// Create context with timeout for individual request
						reqCtx, cancel := context.WithTimeout(ctxWithGrace, 5*time.Second)

						// Use the request context in the function call
						err := func() error {
							// Check if context is already cancelled before making the call
							select {
							case <-reqCtx.Done():
								return reqCtx.Err()
							default:
								return lg.fn()
							}
						}()

						cancel() // Cancel the request context

						if err == nil {
							success = true
							break
						}

						lastErr = err
						errorCount++

						// If we're in grace period, don't retry
						if time.Now().After(lg.endTime) {
							// log.Printf("Worker %d: stopping retries during grace period", id)
							break
						}

						// log.Printf("Worker %d: attempt %d/%d failed: %v",
						// id, attempt+1, maxRetries, err)

						if attempt < maxRetries-1 {
							backoffDuration := backoffBase * time.Duration(math.Pow(2, float64(attempt)))

							select {
							case <-ctxWithGrace.Done():
								return
							case <-time.After(backoffDuration):
								continue
							}
						}
					}

					if !success {
						select {
						case errChan <- fmt.Errorf("worker %d: all retries failed, last error: %v",
							id, lastErr):
							errorsMutex.Lock()
							errors = append(errors, lastErr)
							errorsMutex.Unlock()
						default:
							log.Printf("Error channel full: worker %d all retries failed: %v",
								id, lastErr)
						}
					}

					requestCount++

					// if requestCount%1000 == 0 {
					// 	log.Printf("Worker %d progress: %d requests processed (%d errors)",
					// 		id, requestCount, errorCount)
					// }
				}
			}
		}(workerID)
	}

	// Create a channel to signal completion of all goroutines
	done := make(chan struct{})
	go func() {
		wg.Wait()
		// log.Printf("All workers completed")
		close(done)
	}()

	// Start error collector
	go func() {
		for err := range errChan {
			log.Printf("Error collected: %v", err)
		}
	}()

	// Wait for completion with grace period
	select {
	case <-done:
		log.Printf("Test completed normally")
	case <-ctxWithGrace.Done():
		log.Printf("Test completed with grace period timeout")
	}

	close(errChan)

	// Final error summary
	errorsMutex.Lock()
	defer errorsMutex.Unlock()
	if len(errors) > 0 {
		log.Printf("Test completed with %d errors", len(errors))
		for i, err := range errors {
			log.Printf("Error %d: %v", i+1, err)
		}
	}

	return nil
}
