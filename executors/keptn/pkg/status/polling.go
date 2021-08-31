package status

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)

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
