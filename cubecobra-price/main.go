package main

import (
	"fmt"
	"os"
)

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

	fmt.Printf("%v\n", cube)
	fmt.Printf("total cards: %d\n", len(cube.Cards.Mainboard))

	firstCard := cube.Cards.Mainboard[0]
	scryfallDetails, err := GetScryfallDetails(firstCard)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("%+v\n", scryfallDetails)
}
