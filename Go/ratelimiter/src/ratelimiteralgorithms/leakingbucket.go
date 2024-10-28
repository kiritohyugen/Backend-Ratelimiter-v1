package ratelimiteralgorithms

import (
	"context"
	"log"
	"sync"
	"time"
)

type LeakingBucket struct {
	capacity    int           // capacity of bucket
	rate        int           // leak rate
	duration    time.Duration // leak interval duration
	currentSize int           // current size of bucket
	mutex       sync.Mutex    // mutex for safe concurrent access
	ticker      *time.Ticker  // ticker for periodic leaking
	stopChan    chan struct{} // channel to stop the leaking goroutine
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewLeakingBucket initializes a new LeakingBucket.
func NewLeakingBucket(capacity int, rate int, duration time.Duration) *LeakingBucket {
	ctx, cancel := context.WithCancel(context.Background())
	lb := &LeakingBucket{
		capacity:    capacity,
		rate:        rate,
		duration:    duration,
		currentSize: capacity,
		stopChan:    make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}

	lb.ticker = time.NewTicker(duration / time.Duration(rate))
	go lb.startLeaking()

	return lb
}

// Allow checks if a request can be allowed and consumes a token if possible.
func (lb *LeakingBucket) Allow() (bool, time.Duration) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if lb.currentSize > 0 {
		lb.currentSize--
		log.Printf("Request allowed. Remaining tokens: %d", lb.currentSize)
		return true, 0
	}

	// If the request is denied, return how long the client should wait before retrying
	waitDuration := lb.ResetTime().Sub(time.Now())

	log.Println("Request denied (rate limited). No remaining tokens.")
	return false, waitDuration
}

// Remaining returns the number of remaining tokens in the bucket.
func (lb *LeakingBucket) Remaining() int {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	return lb.currentSize
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

// leakTokens decreases the bucket size based on the leaking rate.
func (lb *LeakingBucket) leakTokens() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	if lb.currentSize < lb.capacity {
		lb.currentSize++
		log.Printf("Token leaked. Remaining tokens: %d", lb.currentSize)
	} else {
		log.Println("Bucket full. No leaking required.")
	}
}

// Stop stops the leaking process and ensures graceful shutdown.
func (lb *LeakingBucket) Stop() {
	lb.cancel() // Cancel the context to signal shutdown
}
