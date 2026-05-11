package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type tokenBucket struct {
	tokens   float64
	capacity float64
	rate     float64 // tokens per second
	lastTime time.Time
	mu       sync.Mutex
}

func newTokenBucket(capacity, rate float64) *tokenBucket {
	return &tokenBucket{
		tokens:   capacity,
		capacity: capacity,
		rate:     rate,
		lastTime: time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()
	tb.lastTime = now
	tb.tokens = min(tb.capacity, tb.tokens+elapsed*tb.rate)

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

var (
	buckets   = make(map[string]*tokenBucket)
	bucketsMu sync.Mutex
)

func getBucket(ip string, capacity, rate float64) *tokenBucket {
	bucketsMu.Lock()
	defer bucketsMu.Unlock()
	if b, ok := buckets[ip]; ok {
		return b
	}
	b := newTokenBucket(capacity, rate)
	buckets[ip] = b
	return b
}

// RateLimit limits requests per IP. capacity = burst, rate = tokens/sec refill.
func RateLimit(capacity, ratePerSec float64) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		bucket := getBucket(ip, capacity, ratePerSec)
		if !bucket.allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}
