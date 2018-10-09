package main

import (
	"fmt"
)

func main() {
	numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	for _, n := range numbers {
		if n%2 == 0 {
			//even
			fmt.Printf("%d is even\n", n)
		} else {
			//even
			fmt.Printf("%d is odd\n", n)
		}
	}
}
