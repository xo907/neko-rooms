package auth

import (
	"sync"
	"time"
)

// limiter blocks a key after too many failures within a window.
type limiter struct {
	mu       sync.Mutex
	max      int
	window   time.Duration
	failures map[string][]time.Time
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{
		max:      max,
		window:   window,
		failures: map[string][]time.Time{},
	}
}

func (l *limiter) prune(key string, now time.Time) []time.Time {
	list := l.failures[key]
	i := 0
	for i < len(list) && now.Sub(list[i]) > l.window {
		i++
	}
	list = list[i:]
	if len(list) == 0 {
		delete(l.failures, key)
	} else {
		l.failures[key] = list
	}
	return list
}

// Blocked reports whether the key is blocked and for how long.
func (l *limiter) Blocked(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	list := l.prune(key, now)
	if len(list) < l.max {
		return false, 0
	}
	return true, l.window - now.Sub(list[0])
}

func (l *limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.failures[key] = append(l.prune(key, now), now)

	// keep memory bounded
	if len(l.failures) > 10000 {
		for k := range l.failures {
			l.prune(k, now)
		}
	}
}

func (l *limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, key)
}
