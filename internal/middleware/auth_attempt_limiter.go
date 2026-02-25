package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type AuthAttemptLimiter struct {
	mu            sync.Mutex
	entries       map[string]*authAttempt
	maxFailures   int
	window        time.Duration
	blockDuration time.Duration
	lastCleanup   time.Time
	cleanupEvery  time.Duration
	staleEntryTTL time.Duration
}

type authAttempt struct {
	failures     int
	windowStart  time.Time
	blockedUntil time.Time
	lastSeen     time.Time
}

func NewAuthAttemptLimiter(maxFailures int, window, blockDuration time.Duration) *AuthAttemptLimiter {
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if window <= 0 {
		window = 5 * time.Minute
	}
	if blockDuration <= 0 {
		blockDuration = 15 * time.Minute
	}

	now := time.Now()
	return &AuthAttemptLimiter{
		entries:       make(map[string]*authAttempt),
		maxFailures:   maxFailures,
		window:        window,
		blockDuration: blockDuration,
		lastCleanup:   now,
		cleanupEvery:  5 * time.Minute,
		staleEntryTTL: 24 * time.Hour,
	}
}

func (l *AuthAttemptLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	entry, ok := l.entries[key]
	if !ok {
		l.cleanupLocked(now)
		return true
	}

	entry.lastSeen = now
	if now.Before(entry.blockedUntil) {
		l.cleanupLocked(now)
		return false
	}

	if now.Sub(entry.windowStart) > l.window {
		entry.failures = 0
		entry.windowStart = now
	}

	l.cleanupLocked(now)
	return true
}

func (l *AuthAttemptLimiter) registerFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	entry, ok := l.entries[key]
	if !ok {
		l.entries[key] = &authAttempt{
			failures:    1,
			windowStart: now,
			lastSeen:    now,
		}
		l.cleanupLocked(now)
		return
	}

	entry.lastSeen = now
	if now.Sub(entry.windowStart) > l.window {
		entry.windowStart = now
		entry.failures = 0
	}

	entry.failures++
	if entry.failures >= l.maxFailures {
		entry.blockedUntil = now.Add(l.blockDuration)
		entry.failures = 0
		entry.windowStart = now
	}

	l.cleanupLocked(now)
}

func (l *AuthAttemptLimiter) registerSuccess(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.entries, key)
	l.cleanupLocked(time.Now())
}

func (l *AuthAttemptLimiter) cleanupLocked(now time.Time) {
	if now.Sub(l.lastCleanup) < l.cleanupEvery {
		return
	}

	for key, entry := range l.entries {
		if now.Sub(entry.lastSeen) > l.staleEntryTTL && now.After(entry.blockedUntil) {
			delete(l.entries, key)
		}
	}

	l.lastCleanup = now
}

func clientIPKey(r *http.Request, prefix string) string {
	host := r.RemoteAddr
	if parsedHost, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = parsedHost
	}
	if host == "" {
		host = "unknown"
	}
	return prefix + ":" + host
}
