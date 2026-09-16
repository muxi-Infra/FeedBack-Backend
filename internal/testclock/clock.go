// Package testclock supplies deterministic scheduling for configuration tests.
package testclock

import (
	"github.com/muxi-Infra/FeedBack-Backend/pkg/configclock"
	"sync"
	"time"
)

type Clock struct {
	mu      sync.Mutex
	now     time.Time
	timers  map[*timer]bool
	Created chan struct{}
}
type timer struct {
	clock  *Clock
	at     time.Time
	period time.Duration
	ch     chan time.Time
}
type ticker struct{ *timer }

func New() *Clock {
	return &Clock{now: time.Unix(1800000000, 0), timers: make(map[*timer]bool), Created: make(chan struct{}, 100)}
}
func (c *Clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *Clock) add(d, period time.Duration) *timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &timer{clock: c, at: c.now.Add(d), period: period, ch: make(chan time.Time, 1)}
	c.timers[t] = true
	select {
	case c.Created <- struct{}{}:
	default:
	}
	return t
}
func (c *Clock) NewTimer(d time.Duration) configclock.Timer   { return c.add(d, 0) }
func (c *Clock) NewTicker(d time.Duration) configclock.Ticker { return ticker{c.add(d, d)} }
func (t *timer) C() <-chan time.Time                          { return t.ch }
func (t *timer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	_, ok := t.clock.timers[t]
	delete(t.clock.timers, t)
	return ok
}
func (t ticker) Stop() { t.timer.Stop() }
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	for t := range c.timers {
		if !t.at.After(c.now) {
			select {
			case t.ch <- c.now:
			default:
			}
			if t.period == 0 {
				delete(c.timers, t)
			} else {
				t.at = c.now.Add(t.period)
			}
		}
	}
}
func (c *Clock) Active() int { c.mu.Lock(); defer c.mu.Unlock(); return len(c.timers) }
