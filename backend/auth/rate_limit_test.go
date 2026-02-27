package auth

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
}

func TestAllow_FirstRequest(t *testing.T) {
	rl := newTestLimiter(3, time.Minute)
	assert.True(t, rl.Allow("192.168.1.1"))
}

func TestAllow_UnderLimit(t *testing.T) {
	rl := newTestLimiter(3, time.Minute)
	assert.True(t, rl.Allow("192.168.1.1"))
	assert.True(t, rl.Allow("192.168.1.1"))
	assert.True(t, rl.Allow("192.168.1.1"))
}

func TestAllow_AtLimit(t *testing.T) {
	rl := newTestLimiter(3, time.Minute)
	assert.True(t, rl.Allow("192.168.1.1"))
	assert.True(t, rl.Allow("192.168.1.1"))
	assert.True(t, rl.Allow("192.168.1.1"))
	assert.False(t, rl.Allow("192.168.1.1"))
	assert.False(t, rl.Allow("192.168.1.1"))
}

func TestAllow_DifferentKeys(t *testing.T) {
	rl := newTestLimiter(1, time.Minute)
	assert.True(t, rl.Allow("ip-a"))
	assert.False(t, rl.Allow("ip-a"))
	assert.True(t, rl.Allow("ip-b"))
	assert.False(t, rl.Allow("ip-b"))
}

func TestAllow_ExpiredAttemptsAreRemoved(t *testing.T) {
	rl := newTestLimiter(2, 50*time.Millisecond)
	assert.True(t, rl.Allow("key"))
	assert.True(t, rl.Allow("key"))
	assert.False(t, rl.Allow("key"))

	time.Sleep(60 * time.Millisecond)

	assert.True(t, rl.Allow("key"))
}

func TestAllow_ConcurrentAccess(t *testing.T) {
	rl := newTestLimiter(100, time.Minute)
	var wg sync.WaitGroup

	allowed := 0
	var mu sync.Mutex

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow("concurrent-key") {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, 100, allowed)
}

func TestCleanup_RemovesExpiredEntries(t *testing.T) {
	rl := newTestLimiter(10, 50*time.Millisecond)
	rl.Allow("expired-key")
	rl.Allow("expired-key")

	time.Sleep(60 * time.Millisecond)

	rl.Cleanup()

	rl.mu.RLock()
	_, exists := rl.attempts["expired-key"]
	rl.mu.RUnlock()
	assert.False(t, exists)
}

func TestCleanup_KeepsActiveEntries(t *testing.T) {
	rl := newTestLimiter(10, time.Minute)
	rl.Allow("active-key")

	rl.Cleanup()

	rl.mu.RLock()
	_, exists := rl.attempts["active-key"]
	rl.mu.RUnlock()
	assert.True(t, exists)
}

func TestCleanup_MixedEntries(t *testing.T) {
	rl := newTestLimiter(10, 50*time.Millisecond)

	rl.Allow("will-expire")
	time.Sleep(60 * time.Millisecond)
	rl.Allow("still-active")

	rl.Cleanup()

	rl.mu.RLock()
	_, expiredExists := rl.attempts["will-expire"]
	_, activeExists := rl.attempts["still-active"]
	rl.mu.RUnlock()

	assert.False(t, expiredExists)
	assert.True(t, activeExists)
}

func TestRateLimitMiddleware_AllowsRequest(t *testing.T) {
	rl := newTestLimiter(5, time.Minute)

	r := gin.New()
	r.Use(RateLimitMiddleware(rl))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitMiddleware_BlocksExcessiveRequests(t *testing.T) {
	rl := newTestLimiter(2, time.Minute)

	r := gin.New()
	r.Use(RateLimitMiddleware(rl))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestGetLoginLimiter(t *testing.T) {
	limiter := GetLoginLimiter()
	assert.NotNil(t, limiter)
	assert.Equal(t, 5, limiter.max)
	assert.Equal(t, 15*time.Minute, limiter.window)
}
