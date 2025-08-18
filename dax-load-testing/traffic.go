package main

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/aws/aws-dax-go-v2/dax"
	"github.com/aws/aws-sdk-go-v2/aws"
)

func (dl *DataLoader) trafficRampUp(cfg aws.Config, aggressive bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}

	go func() {
		<-time.After(time.Minute * time.Duration(Flags.App.TestDurationMinutes))
		log.Print("[trafficRampUp] Cancelling due to timeout")
		cancel()
	}()

	var daxClientBuilder func(cfg *aws.Config, aggressive bool) (*dax.Dax, error)
	if Flags.App.IsTLSClient {
		daxClientBuilder = getSecureDAXClientForTLS
	} else {
		daxClientBuilder = getDaxClient
	}

	daxClients := []*dax.Dax{}
	for range rangeClosed(1, Flags.App.NumberOfClients) {
		daxClient, err := daxClientBuilder(&cfg, aggressive)
		if err != nil {
			panic(err)
		}
		daxClients = append(daxClients, daxClient)
	}

	// sleep after all daxClients are started
	time.Sleep(time.Second)

	startedGoroutines := 0
	expectedGoroutines := Flags.App.NumberOfThreadsPerClient * Flags.App.NumberOfClients
	for range Flags.App.NumberOfThreadsPerClient {
		for clientId := range daxClients {
			wg.Add(1)
			go func() {
				dl.submitTrafficTask(ctx, daxClients[clientId], aggressive)
				wg.Done()
			}()
			startedGoroutines += 1

			log.Printf(
				"[trafficRampUp] Started %d of %d goroutines for client %d, will now sleep for %d milliseconds before starting the next one.",
				startedGoroutines,
				expectedGoroutines,
				clientId,
				time.Duration(Flags.App.ThreadSpawnIntervalMS),
			)

			time.Sleep(time.Millisecond * time.Duration(Flags.App.ThreadSpawnIntervalMS))
		}
	}

	log.Printf(
		"[trafficRampUp] Started %d of %d goroutines.",
		startedGoroutines,
		expectedGoroutines,
	)

	wg.Wait()

	for range 10 {
		for range 10 {
			runtime.GC()
		}

		time.Sleep(time.Second)
	}

	log.Print("closing all clients")
	for _, client := range daxClients {
		_ = client.Close()
	}
	log.Print("[trafficRampUp] Closed all daxClients")
}

func (dl *DataLoader) submitTrafficTask(ctx context.Context, daxClient *dax.Dax, aggressive bool) {
	rpsPerThread := Flags.App.InitialRPSPerThread
	sleepInterval := int(1_000.0 / Flags.App.InitialRPSPerThread)
	//log.Printf("[submitTrafficTask] sleepInterval: %d", sleepInterval)
	r := NewRandom[int](100, 0)

	for {
		select {
		case <-ctx.Done():
			log.Print("[submitTrafficTask] Context done")
			return
		default:
			//
		}

		if rpsPerThread < Flags.App.FinalRPSPerThread {
			rpsPerThread += Flags.App.RPSRampingFactor
			sleepInterval = int(1_000.0 / rpsPerThread)
		}

		if err := dl.executeTrafficCycle(ctx, daxClient, sleepInterval, r.Next(), aggressive); err != nil {
			log.Printf("[submitTrafficTask] Error in %s cycle: %v", ternary(Flags.App.WriteTest, "writeTest traffic", "traffic"), err)
			time.Sleep(time.Second)
		}
	}
}

func (dl *DataLoader) executeTrafficCycle(ctx context.Context, daxClient *dax.Dax, sleepInterval int, rnd int, aggressive bool) error {
	var worker workerFn
	if Flags.App.WriteTest {
		if rnd < 75 {
			worker = (*DataLoader).putItem
		} else if rnd < 90 {
			worker = (*DataLoader).updateItem
		} else {
			worker = (*DataLoader).batchWriteItem
		}
	} else {
		if rnd < 75 {
			worker = (*DataLoader).getItem
		} else if rnd < 90 {
			worker = (*DataLoader).query
		} else {
			worker = (*DataLoader).batchGetItem
		}
	}

	if worker == nil {
		return nil
	}

	return worker(dl, ctx, daxClient, time.Millisecond*ternary[time.Duration](aggressive, 150, 60_000))
}
