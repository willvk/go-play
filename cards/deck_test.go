package main

import (
	"os"
	"testing"
)

func TestNewDeck(t *testing.T) {
	deck := newDeck()
	if len(deck) != 52 {
		t.Errorf("Cards in deck not equal to 52 instead is [%d]", len(deck))
	}
	if deck[0] != "Two of Hearts" {
		t.Errorf("First card in deck not the expected Two of Hearts, instead was [%s]", deck[0])
	}
	if deck[len(deck)-1] != "Ace of Diamonds" {
		t.Errorf("Last card in deck not the expected Ace of Diamonds, instead was [%s]", deck[len(deck)-1])
	}
}

func TestDeckIO(t *testing.T) {

	os.Remove("_decktesting.dck")
	deck := newDeck()
	deck.saveToFile("_decktesting.dck")
	loadedDeck := newDeckFromFile("_decktesting.dck")

	if len(loadedDeck) != 52 {
		t.Errorf("Cards in deck not equal to 52 instead is [%d]", len(loadedDeck))
	}
	if loadedDeck[0] != "Two of Hearts" {
		t.Errorf("First card in deck not the expected Two of Hearts, instead was [%s]", loadedDeck[0])
	}
	if loadedDeck[len(loadedDeck)-1] != "Ace of Diamonds" {
		t.Errorf("Last card in deck not the expected Ace of Diamonds, instead was [%s]", loadedDeck[len(loadedDeck)-1])
	}

	os.Remove("_decktesting.dck")
}
