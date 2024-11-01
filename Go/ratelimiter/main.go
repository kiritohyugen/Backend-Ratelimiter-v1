package main

import (
	"fmt"
	"log"
	"net/http"
	_ "strconv"
	"time"

	rl "github.com/kiritohyugen/ratelimiter/src/ratelimiteralgorithms"
	rli "github.com/kiritohyugen/ratelimiter/src/rlinterface"

	"context"

	"github.com/redis/go-redis/v9"
)

// Define a package-level variable for the rate limiter
var rlimiter rli.RateLimiter

var redisclient *redis.Client

// // Initialize the rate limiter once at the start of the application
// func initRateLimiter(ratelimiter string, rate, capacity int, duration time.Duration) {
// 	switch ratelimiter {
// 	case "tokenbucket":
// 		rlimiter = rl.NewTokenBucket(rate, capacity, duration)
// 	case "leakingbucket":
// 		rlimiter = rl.NewLeakingBucket(rate, capacity, duration)
// 	default:
// 		rlimiter = rl.NewTokenBucket(rate, capacity, duration)
// 	}
// }

// Initialize the rate limiter with Redis storage
func initRateLimiter(ratelimiter string, rate, capacity int, duration time.Duration) {
	// Log the start of rate limiter initialization
	log.Printf("Initializing rate limiter: %s with rate: %d, capacity: %d, duration: %v", ratelimiter, rate, capacity, duration)

	// Prepare the Redis key for the rate limiter
	key := fmt.Sprintf("ratelimiter:%s", ratelimiter)
	log.Printf("Redis key for rate limiter: %s", key)

	// Try to get the current token count from Redis
	ctx := context.Background()
	tokens, err := redisclient.Get(ctx, key).Int()
	if err == redis.Nil {
		// If key doesn't exist in Redis, initialize with capacity
		tokens = capacity
		log.Printf("Key not found in Redis, initializing with capacity: %d", capacity)

		// Set the initial tokens in Redis
		err = redisclient.Set(ctx, key, tokens, 0).Err()
		if err != nil {
			log.Fatalf("Failed to set tokens in Redis: %v", err)
		}

		// Verify that tokens were set in Redis
		newTokens, err := redisclient.Get(ctx, key).Int()
		if err != nil || newTokens != tokens {
			log.Fatalf("Error verifying tokens in Redis, expected: %d, got: %d, error: %v", tokens, newTokens, err)
		} else {
			log.Printf("Tokens successfully set in Redis with value: %d", newTokens)
		}
	} else if err != nil {
		// Log any error when retrieving tokens from Redis
		log.Fatalf("Error retrieving tokens from Redis: %v", err)
	} else {
		// Log the number of tokens retrieved from Redis
		log.Printf("Retrieved %d tokens from Redis for key: %s", tokens, key)
	}

	// Initialize the appropriate rate limiter based on the provided type
	switch ratelimiter {
	case "tokenbucket":
		log.Println("Initializing Token Bucket rate limiter")
		rlimiter = rl.NewTokenBucket(rate, capacity, duration, tokens, redisclient, ratelimiter)
	case "leakingbucket":
		log.Println("Initializing Leaking Bucket rate limiter")
		rlimiter = rl.NewLeakingBucket(rate, capacity, duration, redisclient, ratelimiter) // Adjust for leaking bucket logic
	default:
		log.Printf("Unknown rate limiter type: %s, defaulting to Token Bucket", ratelimiter)
		rlimiter = rl.NewTokenBucket(rate, capacity, duration, tokens, redisclient, ratelimiter)
	}

	// Log successful initialization
	log.Println("Rate limiter initialized successfully.")
}

// func rateLimit(ratelimiter string, rate, capacity int, duration time.Duration) {
// 	var rlimiter rli.RateLimiter
// 	switch ratelimiter {
// 	case "tokenbucket":
// 		rlimiter = rl.NewTokenBucket(rate, capacity, duration*time.Second)
// 	case "leakingbucket":
// 		rlimiter = rl.NewLeakingBucket(rate, capacity, duration*time.Second)
// 	default:
// 		rlimiter = rl.NewTokenBucket(rate, capacity, duration*time.Second)

// 	}
// 	if ok, time := rlimiter.Allow(); ok {
// 		fmt.Println("Request is allowed", "times is : ", time)
// 	} else {
// 		fmt.Println("Request denied (rate limited)", "times is : ", time)
// 	}
// }

func rateLimit() {
	// Check if the request is allowed by the rate limiter
	ok, waitTime := rlimiter.Allow()
	if ok {
		fmt.Println("Request is allowed", "Wait time until next request: ", waitTime)
	} else {
		fmt.Println("Request denied (rate limited)", "Wait time until next request: ", waitTime)
	}
}

func handlerequest(w http.ResponseWriter, r *http.Request) {

	// Log the start of the request handling
	log.Printf("Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

	// Log query parameters
	queryParams := r.URL.Query()
	log.Println("Query Parameters: ", queryParams)

	rlalgo := r.URL.Query().Get("algo")
	log.Printf("Algorithm chosen for rate limiting: %s", rlalgo)

	// Initialize the rate limiter if it hasn't been done yet
	if rlimiter == nil {
		log.Println("Rate limiter is nil, initializing with default values.")
		initRateLimiter(rlalgo, 5, 5, 60*time.Second)
	} else {
		log.Println("Rate limiter already initialized.")
	}

	// Call rate limiting logic and log it
	log.Println("Calling rate limit logic.")
	rateLimit()

	// Log the response status
	log.Println("Request handling completed.")
}

func main() {

	redisclient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // No password set
		DB:       0,  // Use default DB
		Protocol: 2,  // Connection protocol
	})

	ctx := context.Background()

	rerr := redisclient.Set(ctx, "foo", "bar", 0).Err()
	if rerr != nil {
		panic(rerr)
	}

	val, rrerr := redisclient.Get(ctx, "foo").Result()
	if rrerr != nil {
		panic(rrerr)
	}
	fmt.Println("foo", val)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/endpoint", handlerequest)

	log.Println("Starting server on port :4000")
	err := http.ListenAndServe(":4000", mux)
	log.Fatal(err)

}
