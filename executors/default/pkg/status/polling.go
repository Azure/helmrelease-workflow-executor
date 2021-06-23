package status

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

// Retry retries the passed retryable function, sleeping for the given backoffDuration
// If the context times out while executing the retryable function, an error is returned
func Retry(ctx context.Context, retryable func() error, backoffDuration time.Duration) error {
	pollingFunc := func(done chan<- bool) {
		if err := retryable(); err != nil {
			log.Errorf("Failed to execute the retryable function with: %v", err)
		}
		log.Infof("successfully completed the retry function")
		done <- true
	}
	return Poll(ctx, pollingFunc, backoffDuration)
}

// Poll retries the poller function, sleeping for the given backoffDuration
// If the poller times out it will return an error
func Poll(ctx context.Context, poller func(chan<- bool), backoffDuration time.Duration) error {
	done := make(chan bool)
	for {
		go poller(done)
		select {
		case <-ctx.Done():
			return fmt.Errorf("failed to complete polling within the timeout")
		case <-done:
			log.Infof("successfully completed the polling function")
			return nil
		default:
			time.Sleep(backoffDuration)
		}
	}
}
