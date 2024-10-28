package main

import (
	"fmt"
	rl "ratelimiter/src/ratelimiteralgorithms"
	"time"
)

func main() {
	// Initialize rate limiter with 5 tokens per second, bucket capacity of 5, and a 16-second duration
	// r1 := rl.NewTokenBucket(5, 5, 16*time.Second)
	r1 := rl.NewLeakingBucket(5, 5, 16*time.Second)

	// Simulate 20 requests with a 2-second delay between each
	for i := 1; i <= 20; i++ {
		if ok, time := r1.Allow(); ok {
			fmt.Println("Request", i, "is allowed", "times is : ", time)
		} else {
			fmt.Println("Request", i, "denied (rate limited)", "times is : ", time)
		}
		time.Sleep(2 * time.Second) // Simulate delay between requests
	}
}

// TODO apply leaking bucket algorithmm
// TODO add endpoints and requests to make rate limiting configurable wherther to apply on endpoint,usertoken etc
// TODO try from eexternal Postman
// TODO push on github
// TODO improve it
