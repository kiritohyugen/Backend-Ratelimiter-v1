// package ratelimiteralgorithms

// import (
// 	"context"
// 	"log"
// 	"sync"
// 	"time"
// )

// type LeakingBucket struct {
// 	capacity    int           // capacity of bucket
// 	rate        int           // leak rate
// 	duration    time.Duration // leak interval duration
// 	currentSize int           // current size of bucket
// 	mutex       sync.Mutex    // mutex for safe concurrent access
// 	ticker      *time.Ticker  // ticker for periodic leaking
// 	stopChan    chan struct{} // channel to stop the leaking goroutine
// 	ctx         context.Context
// 	cancel      context.CancelFunc
// }

// // NewLeakingBucket initializes a new LeakingBucket.
// func NewLeakingBucket(capacity int, rate int, duration time.Duration) *LeakingBucket {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	lb := &LeakingBucket{
// 		capacity:    capacity,
// 		rate:        rate,
// 		duration:    duration,
// 		currentSize: capacity,
// 		stopChan:    make(chan struct{}),
// 		ctx:         ctx,
// 		cancel:      cancel,
// 	}

// 	lb.ticker = time.NewTicker(duration / time.Duration(rate))
// 	go lb.startLeaking()

// 	return lb
// }

// // Allow checks if a request can be allowed and consumes a token if possible.
// func (lb *LeakingBucket) Allow() (bool, time.Duration) {
// 	lb.mutex.Lock()
// 	defer lb.mutex.Unlock()

// 	if lb.currentSize > 0 {
// 		lb.currentSize--
// 		log.Printf("Request allowed. Remaining tokens: %d", lb.currentSize)
// 		return true, 0
// 	}

// 	// If the request is denied, return how long the client should wait before retrying
// 	waitDuration := lb.ResetTime().Sub(time.Now())

// 	log.Println("Request denied (rate limited). No remaining tokens.")
// 	return false, waitDuration
// }

// // Remaining returns the number of remaining tokens in the bucket.
// func (lb *LeakingBucket) Remaining() int {
// 	lb.mutex.Lock()
// 	defer lb.mutex.Unlock()
// 	return lb.currentSize
// }

// // ResetTime returns the time when the bucket will be refilled.
// func (lb *LeakingBucket) ResetTime() time.Time {
// 	return time.Now().Add(lb.duration)
// }

// // SetRate dynamically adjusts the rate and duration of the leaking process.
// func (lb *LeakingBucket) SetRate(rate int, duration time.Duration) {
// 	lb.mutex.Lock()
// 	defer lb.mutex.Unlock()

// 	lb.rate = rate
// 	lb.duration = duration
// 	lb.ticker.Stop() // Stop the current ticker
// 	lb.ticker = time.NewTicker(duration / time.Duration(rate))
// 	log.Printf("Rate updated to %d requests per %s", rate, duration)
// }

// // startLeaking leaks tokens periodically based on the rate and duration.
// func (lb *LeakingBucket) startLeaking() {
// 	for {
// 		select {
// 		case <-lb.ctx.Done():
// 			// Graceful shutdown
// 			lb.ticker.Stop()
// 			log.Println("Leaking process stopped")
// 			return
// 		case <-lb.ticker.C:
// 			lb.leakTokens()
// 		}
// 	}
// }

// // leakTokens decreases the bucket size based on the leaking rate.
// func (lb *LeakingBucket) leakTokens() {
// 	lb.mutex.Lock()
// 	defer lb.mutex.Unlock()

// 	if lb.currentSize < lb.capacity {
// 		lb.currentSize++
// 		log.Printf("Token leaked. Remaining tokens: %d", lb.currentSize)
// 	} else {
// 		log.Println("Bucket full. No leaking required.")
// 	}
// }

// // Stop stops the leaking process and ensures graceful shutdown.
// func (lb *LeakingBucket) Stop() {
// 	lb.cancel() // Cancel the context to signal shutdown
// }

package ratelimiteralgorithms

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9" // Import the Redis package
)

type LeakingBucket struct {
	capacity    int           // Capacity of bucket
	rate        int           // Leak rate (tokens per interval)
	duration    time.Duration // Leak interval duration
	mutex       sync.Mutex    // Mutex for safe concurrent access
	ticker      *time.Ticker  // Ticker for periodic leaking
	stopChan    chan struct{} // Channel to stop the leaking goroutine
	ctx         context.Context
	cancel      context.CancelFunc
	redisClient *redis.Client // Redis client
	redisKey    string        // Redis key to store the bucket size
}

// NewLeakingBucket initializes a new LeakingBucket with Redis integration.
func NewLeakingBucket(capacity int, rate int, duration time.Duration, redisClient *redis.Client, redisKey string) *LeakingBucket {
	ctx, cancel := context.WithCancel(context.Background())
	lb := &LeakingBucket{
		capacity:    capacity,
		rate:        rate,
		duration:    duration,
		stopChan:    make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
		redisClient: redisClient,
		redisKey:    redisKey,
	}

	lb.ticker = time.NewTicker(duration / time.Duration(rate))
	go lb.startLeaking()

	return lb
}

// Allow checks if a request can be allowed and consumes a token if possible.
func (lb *LeakingBucket) Allow() (bool, time.Duration) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Get current size from Redis
	currentSize, err := lb.getCurrentSize()
	if err != nil {
		log.Println("Error fetching current bucket size from Redis:", err)
		return false, lb.duration
	}

	if currentSize > 0 {
		// Decrease the size of the bucket by 1
		lb.redisClient.Decr(lb.ctx, lb.redisKey)
		log.Printf("Request allowed. Remaining tokens: %d", currentSize-1)
		return true, 0
	}

	// If request is denied, return wait time before retrying
	waitDuration := lb.ResetTime().Sub(time.Now())
	log.Println("Request denied (rate limited). No remaining tokens.")
	return false, waitDuration
}

// getCurrentSize retrieves the current size of the bucket from Redis.
func (lb *LeakingBucket) getCurrentSize() (int, error) {
	// Get the current size from Redis
	size, err := lb.redisClient.Get(lb.ctx, lb.redisKey).Int()
	if err == redis.Nil {
		// If the key doesn't exist, initialize it
		lb.redisClient.Set(lb.ctx, lb.redisKey, lb.capacity, 0) // Initialize with full capacity
		return lb.capacity, nil
	} else if err != nil {
		return 0, err
	}
	return size, nil
}

// Remaining returns the number of remaining tokens in the bucket.
func (lb *LeakingBucket) Remaining() int {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Get the current size from Redis
	currentSize, err := lb.getCurrentSize()
	if err != nil {
		log.Println("Error fetching remaining tokens from Redis:", err)
		return 0
	}

	return currentSize
}

// ResetTime returns the time when the bucket will be refilled.
func (lb *LeakingBucket) ResetTime() time.Time {
	return time.Now().Add(lb.duration)
}

// SetRate dynamically adjusts the rate and duration of the leaking process.
func (lb *LeakingBucket) SetRate(rate int, duration time.Duration) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.rate = rate
	lb.duration = duration
	lb.ticker.Stop() // Stop the current ticker
	lb.ticker = time.NewTicker(duration / time.Duration(rate))
	log.Printf("Rate updated to %d requests per %s", rate, duration)
}

// startLeaking leaks tokens periodically based on the rate and duration.
func (lb *LeakingBucket) startLeaking() {
	for {
		select {
		case <-lb.ctx.Done():
			// Graceful shutdown
			lb.ticker.Stop()
			log.Println("Leaking process stopped")
			return
		case <-lb.ticker.C:
			lb.leakTokens()
		}
	}
}

// leakTokens decreases the bucket size in Redis based on the leaking rate.
func (lb *LeakingBucket) leakTokens() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Get current size from Redis
	currentSize, err := lb.getCurrentSize()
	if err != nil {
		log.Println("Error fetching current bucket size from Redis:", err)
		return
	}

	if currentSize < lb.capacity {
		// Increase the size of the bucket by 1 (leak a token)
		lb.redisClient.Incr(lb.ctx, lb.redisKey)
		log.Printf("Token leaked. Remaining tokens: %d", currentSize+1)
	} else {
		log.Println("Bucket full. No leaking required.")
	}
}

// Stop stops the leaking process and ensures graceful shutdown.
func (lb *LeakingBucket) Stop() {
	lb.cancel() // Cancel the context to signal shutdown
}
