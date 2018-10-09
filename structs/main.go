package main

import (
	"fmt"
)

type person struct {
	firstname string
	lastname  string
	age       int
	contactInfo
}

type contactInfo struct {
	email   string
	zipcode int
}

func main() {
	// do the thing
	alex := person{
		firstname: "alex",
		lastname:  "anderson",
		age:       16,
		contactInfo: contactInfo{
			email:   "alexanderson@gmail.com",
			zipcode: 94000,
		},
	}
	alex.print()
	alex.updateName("penis")
	alex.print()
}

func (p person) print() {
	fmt.Printf("%+v\n", p)
}

func (p *person) updateName(name string) {
	(*p).firstname = name
}
