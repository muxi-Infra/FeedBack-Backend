package retry

import (
	"time"
)

const (
	MaxAttempts = 3
	DelayMs     = 20
)

func Retry[Resp any](fn func() (Resp, error)) (Resp, error) {
	var res Resp
	var lastErr error
	delay := time.Duration(DelayMs) * time.Millisecond
	for i := 1; i <= MaxAttempts; i++ {
		res, err := fn()
		if err == nil {
			return res, nil
		}
		lastErr = err
		time.Sleep(delay)
		delay *= 2
	}
	return res, lastErr
}
