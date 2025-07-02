package utils

import (
	"fmt"
	"time"
)

func Retry(attempts int, sleep time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		fmt.Printf("Retrying (%d/%d) after error: %v\n", i+1, attempts, err)
		time.Sleep(sleep)
	}
	return fmt.Errorf("all retries failed: %w", err)
}
