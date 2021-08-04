package actions

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Delete(ctx context.Context, cancel context.CancelFunc, clientSet client.Client, interval time.Duration) error {
	return nil
}
