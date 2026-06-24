package bdk

import (
	"errors"
	"fmt"
	"time"
)

func Retry(fn func() error, delay time.Duration, attempts int) error {
	if attempts <= 0 {
		return errors.New("attempts must greater than 0")
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}
		time.Sleep(delay)
	}
	return fmt.Errorf("all %d attempts failed, last err: %w", attempts, lastErr)
}
