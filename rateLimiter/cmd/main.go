package main

func main() {
	cache := requests
	customers := []int{1, 1, 1, 1, 1, 2, 2, 2, 2, 2, 3, 1, 1, 1, 3}

	for _, customer := range customers {
		rateLimiter(customer, cache)
	}
}
