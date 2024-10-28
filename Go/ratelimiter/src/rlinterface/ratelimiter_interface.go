package rlinterface

import "time"

// RateLimiter is a general interface that supports various rate-limiting algorithms.
type RateLimiter interface {
	// Allow checks if a request is allowed under the current rate limit.
	// Returns true if the request is allowed, false otherwise.
	// Optionally returns the time the client should wait before retrying.
	Allow() (bool, time.Duration)

	// Remaining returns the number of tokens/requests left in the current time window.
	// This is useful for monitoring the rate-limiting state.
	Remaining() int

	// ResetTime returns the time when the rate limit will reset and allow new requests.
	// This method can help clients know when they can retry.
	ResetTime() time.Time

	// SetRate dynamically adjusts the rate limit for the rate limiter.
	// Accepts the rate (number of tokens/requests) and the duration (time window).
	SetRate(rate int, duration time.Duration)
}
