package auth

import (
	"sync"
	"time"
)

// Limiter counts failed logins per client IP and per username inside a window.
type Limiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func NewLimiter(max int, window time.Duration) *Limiter {
	if max <= 0 {
		max = 8
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	return &Limiter{hits: make(map[string][]time.Time), max: max, window: window}
}

func (l *Limiter) Blocked(ip, username string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if ip != "" && l.count("ip:"+ip, now) >= l.max {
		return true
	}
	if username != "" && l.count("user:"+username, now) >= l.max {
		return true
	}
	return false
}

func (l *Limiter) Fail(ip, username string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if ip != "" {
		l.add("ip:"+ip, now)
	}
	if username != "" {
		l.add("user:"+username, now)
	}
}

func (l *Limiter) Success(ip, username string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, "ip:"+ip)
	delete(l.hits, "user:"+username)
}

func (l *Limiter) count(key string, now time.Time) int {
	kept := l.prune(l.hits[key], now)
	if len(kept) == 0 {
		delete(l.hits, key)
		return 0
	}
	l.hits[key] = kept
	return len(kept)
}

func (l *Limiter) add(key string, now time.Time) {
	kept := l.prune(l.hits[key], now)
	kept = append(kept, now)
	l.hits[key] = kept
}

func (l *Limiter) prune(xs []time.Time, now time.Time) []time.Time {
	cut := now.Add(-l.window)
	kept := make([]time.Time, 0, len(xs))
	for _, t := range xs {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	return kept
}
