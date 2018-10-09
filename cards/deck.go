package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"
)

// Create a new type of deck which is a slice of strings
type deck []string

func (d deck) print() {
	fmt.Printf("========Decksize is: %d\n", len(d))
	for i, card := range d {
		fmt.Println(i, card)
	}
}

func newDeck() deck {
	cards := deck{}
	cardSuits := []string{"Hearts", "Spades", "Clubs", "Diamonds"}
	cardValues := []string{"Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten", "Jack", "Queen", "King", "Ace"}

	for _, suit := range cardSuits {
		for _, value := range cardValues {
			cards = append(cards, value+" of "+suit)
		}
	}

	return cards
}

func deal(mydeck deck, handsize int) (deck, deck) {
	return mydeck[:handsize], mydeck[handsize:]
}

func (d deck) toString() string {
	return strings.Join([]string(d), ",")
}

func (d deck) saveToFile(filename string) error {
	return ioutil.WriteFile(filename, []byte(d.toString()), 0666)
}

func newDeckFromFile(filename string) deck {
	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("shit broke")
		os.Exit(1)
	}
	return deck(strings.Split(string(bs), ","))
}

func (d deck) shuffle() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := range d {
		newPosition := r.Intn(len(d) - 1)
		d[i], d[newPosition] = d[newPosition], d[i]
	}
}
