// package ratelimiteralgorithms

// import (
// 	// Adjust this path according to your module name
// 	"sync"
// 	"time"
// )

// // TokenBucket struct implements the RateLimiter interface.
// type TokenBucket struct {
// 	rate      int           // Refill rate (tokens per second)
// 	capacity  int           // Maximum capacity of the bucket
// 	tokens    int           // Current available tokens
// 	lastCheck time.Time     // Last time tokens were refilled
// 	duration  time.Duration // Duration to refill bucket
// 	mutex     sync.Mutex    // Mutex for safe concurrent access
// }

// // NewTokenBucket creates a new TokenBucket instance.
// func NewTokenBucket(rate, capacity int, duration time.Duration) *TokenBucket {
// 	return &TokenBucket{
// 		rate:      rate,
// 		capacity:  capacity,
// 		tokens:    capacity,
// 		lastCheck: time.Now(),
// 		duration:  duration,
// 	}
// }

// // Implementing the RateLimiter interface methods.
// func (tb *TokenBucket) Allow() (bool, time.Duration) {
// 	tb.mutex.Lock()
// 	defer tb.mutex.Unlock()

// 	// Refill tokens based on elapsed time
// 	elapsed := time.Since(tb.lastCheck)
// 	refillTokens := int(elapsed/tb.duration) * tb.rate
// 	if refillTokens > 0 {
// 		tb.tokens += refillTokens
// 		if tb.tokens > tb.capacity {
// 			tb.tokens = tb.capacity
// 		}
// 		tb.lastCheck = time.Now()
// 	}

// 	if tb.tokens > 0 {
// 		tb.tokens-- // Consume a token
// 		return true, 0
// 	}
// 	return false, tb.duration - time.Since(tb.lastCheck) // Return wait time until next refill
// }

// func (tb *TokenBucket) Remaining() int {
// 	tb.mutex.Lock()
// 	defer tb.mutex.Unlock()
// 	return tb.tokens
// }

// func (tb *TokenBucket) ResetTime() time.Time {
// 	tb.mutex.Lock()
// 	defer tb.mutex.Unlock()
// 	tb.lastCheck = time.Now()
// 	return tb.lastCheck
// }
// func (tb *TokenBucket) SetRate(rate int, duration time.Duration) {
// 	tb.mutex.Lock()
// 	defer tb.mutex.Unlock()
// 	tb.rate = rate
// 	tb.duration = duration
// 	tb.capacity = rate      // Assuming capacity equals rate initially
// 	tb.tokens = tb.capacity // Reset tokens to the new capacity
// }

package ratelimiteralgorithms

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBucket struct {
	rate        int           // Refill rate (tokens per second)
	capacity    int           // Maximum capacity of the bucket
	tokens      int           // Current available tokens
	lastCheck   time.Time     // Last time tokens were refilled
	duration    time.Duration // Duration to refill bucket
	mutex       sync.Mutex    // Mutex for safe concurrent access
	redisClient *redis.Client
	key         string // Redis key for this bucket
}

// NewTokenBucket creates a new TokenBucket instance.
func NewTokenBucket(rate, capacity int, duration time.Duration, tokens int, redisClient *redis.Client, redisKey string) *TokenBucket {
	return &TokenBucket{
		rate:        rate,
		capacity:    capacity,
		tokens:      tokens,
		lastCheck:   time.Now(),
		duration:    duration,
		redisClient: redisClient,
		key:         redisKey,
	}
}

// Allow checks if a request can proceed and updates the token count in Redis.
func (tb *TokenBucket) Allow() (bool, time.Duration) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Refill tokens based on elapsed time
	elapsed := time.Since(tb.lastCheck)
	refillTokens := int(elapsed/tb.duration) * tb.rate
	if refillTokens > 0 {
		tb.tokens += refillTokens
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastCheck = time.Now()
		// Update tokens in Redis
		tb.redisClient.Set(context.Background(), tb.key, tb.tokens, 0)
	}

	if tb.tokens > 0 {
		tb.tokens--                                                    // Consume a token
		tb.redisClient.Set(context.Background(), tb.key, tb.tokens, 0) // Update Redis
		return true, 0
	}
	return false, tb.duration - time.Since(tb.lastCheck) // Return wait time until next refill
}

func (tb *TokenBucket) Remaining() int {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	return tb.tokens
}

func (tb *TokenBucket) ResetTime() time.Time {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	tb.lastCheck = time.Now()
	return tb.lastCheck
}

func (tb *TokenBucket) SetRate(rate int, duration time.Duration) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	tb.rate = rate
	tb.duration = duration
	tb.capacity = rate      // Assuming capacity equals rate initially
	tb.tokens = tb.capacity // Reset tokens to the new capacity
}
