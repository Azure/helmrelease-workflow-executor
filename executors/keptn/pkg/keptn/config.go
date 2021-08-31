package keptn

import (
	"fmt"
	"time"
)

type Config struct {
	URL       string        `json:"url,omitempty"`
	Namespace string        `json:"namespace,omitempty"`
	Token     KeptnAPIToken `json:"token,omitempty"`
	Timeframe string        `json:"timeframe,omitempty"`
}

func (k *Config) Validate() error {
	if k.URL == "" {
		return fmt.Errorf("keptn API server (nginx) cannot be nil")
	}

	if k.Namespace == "" {
		return fmt.Errorf("keptn namespace must be specified")
	}

	if k.Token.SecretRef.Name == "" {
		return fmt.Errorf("keptn API token secret name must be specified")
	}

	if k.Timeframe == "" {
		return fmt.Errorf("keptn evaluation timeframe must be specified")
	}

	if _, err := time.ParseDuration(k.Timeframe); err != nil {
		return fmt.Errorf("leptn evaluation duration must be similar to the format 5s/2m/1h")
	}

	return nil
}
