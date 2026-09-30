package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	count     int
	expiresAt time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateLimitEntry
	limit   int
	window  time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		entries: make(map[string]rateLimitEntry),
		limit:   limit,
		window:  window,
	}
}

func (r *RateLimiter) Middleware(keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := keyFunc(c)

		if key == "" {
			c.Next()
			return
		}

		now := time.Now()

		r.mu.Lock()

		entry, exists := r.entries[key]

		if !exists || now.After(entry.expiresAt) {
			entry = rateLimitEntry{
				count:     0,
				expiresAt: now.Add(r.window),
			}
		}

		entry.count++

		if entry.count > r.limit {
			retryAfter := int(
				time.Until(entry.expiresAt).Seconds(),
			)

			if retryAfter < 1 {
				retryAfter = 1
			}

			r.entries[key] = entry
			r.mu.Unlock()

			c.Header(
				"Retry-After",
				strconv.Itoa(retryAfter),
			)

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests",
			})
			return
		}

		r.entries[key] = entry
		r.mu.Unlock()

		c.Next()
	}
}

func RateLimitKeyByIP(c *gin.Context) string {
	return c.ClientIP()
}

func RateLimitKeyByUserID(c *gin.Context) string {
	return c.GetString("userID")
}
