package auth

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateBucket struct {
	start time.Time
	count int
	last  time.Time
}
type IPRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]rateBucket
	limit   int
	window  time.Duration
	calls   uint64
}

func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	return &IPRateLimiter{buckets: make(map[string]rateBucket), limit: limit, window: window}
}
func (l *IPRateLimiter) Allow(ip string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++
	if l.calls%256 == 0 {
		for key, b := range l.buckets {
			if now.Sub(b.last) > 2*l.window {
				delete(l.buckets, key)
			}
		}
		if len(l.buckets) > 8192 {
			for key := range l.buckets {
				delete(l.buckets, key)
				if len(l.buckets) <= 4096 {
					break
				}
			}
		}
	}
	b := l.buckets[ip]
	if b.start.IsZero() || now.Sub(b.start) >= l.window {
		b = rateBucket{start: now, last: now}
	}
	b.last = now
	b.count++
	l.buckets[ip] = b
	if b.count <= l.limit {
		return true, 0
	}
	return false, l.window - now.Sub(b.start)
}

func (l *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			ip = c.Request.RemoteAddr
		}
		if ip == "" {
			ip = "unknown"
		}
		ok, retry := l.Allow(ip, time.Now())
		if !ok {
			seconds := int(retry.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests; retry later"})
			return
		}
		c.Next()
	}
}
