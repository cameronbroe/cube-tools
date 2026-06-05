package main

import (
	"fmt"
	"os"
	"slices"
	"strconv"
)

func cheapestTreatmentPrice(card *ScryfallCard) (float64, error) {
	var treatmentPrices []float64
	if card.Prices.Normal != "" {
		asFloat, err := strconv.ParseFloat(card.Prices.Normal, 32)
		if err != nil {
			return 0, err
		}
		treatmentPrices = append(treatmentPrices, asFloat)
	}

	if card.Prices.Foil != "" {
		asFloat, err := strconv.ParseFloat(card.Prices.Foil, 32)
		if err != nil {
			return 0, err
		}
		treatmentPrices = append(treatmentPrices, asFloat)
	}

	if len(treatmentPrices) <= 0 {
		return 0, fmt.Errorf("no prices found for card (%s => %s)", card.Name, card.ID)
	}

	return slices.Min(treatmentPrices), nil
}

func cheapestPrinting(printings []*ScryfallCard) (*ScryfallCard, error) {
	var currentCheapest *ScryfallCard

	for _, printing := range printings {
		if currentCheapest == nil {
			currentCheapest = printing
		} else {
			currentCheapestPrice, err := cheapestTreatmentPrice(currentCheapest)
			if err != nil {
				return nil, err
			}

			printingCheapestPrice, err := cheapestTreatmentPrice(printing)
			if err != nil {
				return nil, err
			}

			if printingCheapestPrice < currentCheapestPrice {
				currentCheapest = printing
			}
		}
	}

	return currentCheapest, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: cubecobra-price [cube-url]")
		os.Exit(1)
	}

	cubeToCheck := os.Args[1]
	cube, err := GetCube(cubeToCheck)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for i, card := range cube.Cards.Mainboard {
		scryfallCards, err := GetScryfallDetails(card)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		cheapestPrinting, err := cheapestPrinting(scryfallCards)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if cheapestPrinting == nil {
			fmt.Printf("couldn't find cheapest printing for %s => %s\n", card.Details.Name, card.Details.ScryfallID)
			os.Exit(1)
		}

		fmt.Printf("%d: %+v\n", i, *cheapestPrinting)
	}
}
