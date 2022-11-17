package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	go func() {
		// listen terminate signal
		// signal is received
		<-sigchan
		// then cancel the work
		cancelFunc()
	}()

	// go UnprocessedReporter(ctx,
	// 	time.Duration(5*time.Second))

processLoop:
	for {
		fmt.Println(1234)
		log.Debug("Processing data")
		// err := mutator.ProcessData(ctx)
		// if err != nil {
		// 	panic(err)
		// }
		select {
		case <-ctx.Done():
			break processLoop
		default:
			time.Sleep(time.Millisecond * time.Duration(rand.Int63n(1000)))
		}
	}
}

func UnprocessedReporter(ctx context.Context, interval time.Duration) {
	fmt.Println(5678)
	ticker := time.NewTicker(interval)
	// dbBackend := models.DbBackend{DbMap: storage.Db}
	for {
		select {
		case <-ticker.C:
			fmt.Println(9999)
			// count, err := dbBackend.StorageService().GetUnprocessedRecordCount(ctx)
			// if err != nil {
			// 	log.WithError(err).Error("Couldn't get unprocessed data count")
			// 	continue
			// }
		case <-ctx.Done():
			log.Debug("Stopping reporting unprocessed counts.")
			ticker.Stop()
			return
		}
	}
}
