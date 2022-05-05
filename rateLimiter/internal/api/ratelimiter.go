package api

import "time"

type requests []request

type request struct {
	first      time.Time
	customerId int
	state      bool
	count      int
}

type (
	// RateLimiter limits the rate of requests
	RateLimiter struct {
		rateLimit time.Time
	}
)

// NewRateLimiter creates an instance of API
func NewRateLimiter(rateLimit time.Time) *RateLimiter {
	return &RateLimiter{
		rateLimit: rateLimit,
	}
}

func (requestCache *requests) rateLimiter(customerId int, cache requests) bool {
	/*
		search map/structure for customerId
		if (new) -> add to datastructure
		if (exists) -> increment count(invocation) this request makes it over threshold -> change state to rejects done
		if (exists & in reject state) -> reject done
	*/
	for i, request := range cache {
		flag := false
		if request.customerId == customerId {
			cache[i].count++
			flag = true
		}
		if cache[i].count > rateLimit {
			return false
		}
		if !flag {
			append(cache, request{})
		}
	}
	return true

}
