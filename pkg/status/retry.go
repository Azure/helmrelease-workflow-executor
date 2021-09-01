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
	for {
		if err := retryable(); err != nil {
			log.Errorf("Failed to execute the retryable function with: %v", err)
			select {
			case <-ctx.Done():
				return fmt.Errorf("failed to complete retry function within the timeout")
			default:
				time.Sleep(backoffDuration)
			}
		} else {
			log.Infof("successfully completed the retry function")
			return nil
		}
	}
}
