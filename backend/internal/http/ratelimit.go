package http

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TODO: swap to Postgres- or Redis-backed limiter when we run more than
// one backend instance, or when a second rate-limited endpoint appears.

// IPRateLimiter is a simple per-IP sliding-window limiter. Safe for concurrent
// use. Not cluster-safe. Intended for low-traffic single-box MVP deployments.
type IPRateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	window time.Duration
	max    int
	now    func() time.Time
}

// NewIPRateLimiter returns a limiter that allows `max` requests per `window`
// per client IP. For MVP use 5 per hour on POST /api/public/waitlist.
func NewIPRateLimiter(max int, window time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		hits:   map[string][]time.Time{},
		window: window,
		max:    max,
		now:    time.Now,
	}
}

// Middleware wraps a handler so requests beyond the limit receive 429.
func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !l.allow(ip) {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", l.window.Seconds()))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "too many requests; try again later",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// allow records a hit for the given ip and returns false if that would exceed
// the limit.
func (l *IPRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cutoff := now.Add(-l.window)

	fresh := l.hits[ip][:0]
	for _, t := range l.hits[ip] {
		if t.After(cutoff) {
			fresh = append(fresh, t)
		}
	}
	if len(fresh) >= l.max {
		l.hits[ip] = fresh
		return false
	}
	fresh = append(fresh, now)
	l.hits[ip] = fresh
	return true
}

// clientIP extracts the client IP from X-Forwarded-For (first entry) if the
// header is present, falling back to RemoteAddr. Vite proxy in dev mode sets
// X-Forwarded-For correctly; production reverse proxies should also.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	addr := r.RemoteAddr
	if i := strings.LastIndexByte(addr, ':'); i >= 0 {
		return addr[:i]
	}
	return addr
}
