package retry

import "time"

const (
	MaxAttempts = 3
	DelayMs     = 20
)

func Retry[T any](fn func() (T, error)) (T, error) {
	var res T
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
