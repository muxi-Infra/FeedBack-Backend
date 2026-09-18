package configclock

import (
	"context"
	"math/rand/v2"
	"time"
)

// Clock controls refresh scheduling and authorization timestamps in tests.
type Clock interface {
	Now() time.Time
	NewTimer(time.Duration) Timer
	NewTicker(time.Duration) Ticker
}
type Timer interface {
	C() <-chan time.Time
	Stop() bool
}
type Ticker interface {
	C() <-chan time.Time
	Stop()
}
type Real struct{}
type timer struct{ *time.Timer }
type ticker struct{ *time.Ticker }

func (Real) Now() time.Time                   { return time.Now() }
func (Real) NewTimer(d time.Duration) Timer   { return timer{time.NewTimer(d)} }
func (Real) NewTicker(d time.Duration) Ticker { return ticker{time.NewTicker(d)} }
func (t timer) C() <-chan time.Time           { return t.Timer.C }
func (t ticker) C() <-chan time.Time          { return t.Ticker.C }
func New() Clock                              { return Real{} }

func Wait(ctx context.Context, c Clock, d time.Duration) bool {
	t := c.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C():
		return ctx.Err() == nil
	}
}

func Backoff(min, max time.Duration, attempt int) time.Duration {
	d := min
	for i := 0; i < attempt && d < max; i++ {
		if d > max/2 {
			d = max
		} else {
			d *= 2
		}
	}
	if d > max {
		d = max
	}
	return d/2 + time.Duration(rand.Int64N(int64(d-d/2)+1))
}
