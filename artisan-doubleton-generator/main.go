package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	packagelog "log"
	"net/http"
	"os"
	"time"
)

func log(log string, args ...any) {
	if os.Getenv("DEBUG") == "1" {
		packagelog.Default().Printf(log, args...)
	}
}

func fatal(log string, args ...any) {
	if os.Getenv("DEBUG") == "1" {
		packagelog.Default().Fatalf(log, args...)
	}
}

type ScryfallResponse struct {
	Data     []Card `json:"data"`
	HasMore  bool   `json:"has_more"`
	NextPage string `json:"next_page"`
}

type Card struct {
	ID       string `json:"id"`
	OracleID string `json:"oracle_id"`
	MTGOID   int    `json:"mtgo_id"`

	Name            string `json:"name"`
	Set             string `json:"set"`
	CollectorNumber string `json:"collector_number"`
}

func GetCardsForSet(setCode string) ([]Card, error) {
	cards := []Card{}

	apiQuery := fmt.Sprintf("set:%s+(r:common+or+r:uncommon)+-t:basic", setCode)
	apiUrl := fmt.Sprintf("https://api.scryfall.com/cards/search?q=%s&unique=cards", apiQuery)
	log("calling out to %s", apiUrl)

	resp, err := getScryfallResponse(apiUrl)
	if err != nil {
		return nil, err
	}

	cards = append(cards, resp.Data...)

	for resp.HasMore {
		time.Sleep(500 * time.Millisecond)
		resp, err = getScryfallResponse(resp.NextPage)
		if err != nil {
			return nil, err
		}

		cards = append(cards, resp.Data...)
	}

	return cards, nil
}

func getScryfallResponse(url string) (*ScryfallResponse, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "ArtisanDoubletonSetCubeGenerator/1.0")
	req.Header.Set("Accept", "application/json")

	log("request: %+v\n", req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	log("respsonse: %+v\n", resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsedResponse ScryfallResponse
	err = json.Unmarshal(body, &parsedResponse)
	if err != nil {
		return nil, err
	}

	return &parsedResponse, err
}

func buildDoubletonCubeList(setCode string, cards []Card) error {
	outputName := fmt.Sprintf("%s-artisan-doubleton-set-cube.csv", setCode)
	outputFile, err := os.Create(outputName)
	if err != nil {
		return err
	}
	csvWriter := csv.NewWriter(outputFile)
	headers := []string{"name", "Set", "Collector Number"}
	csvWriter.Write(headers)

	for _, card := range cards {
		line := []string{card.Name, card.Set, card.CollectorNumber}

		// Doubleton cube has 2 copies of each card
		csvWriter.Write(line)
		csvWriter.Write(line)
	}
	csvWriter.Flush()
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fatal("need to provide a set code to build an artisan doubleton cube list for\n")
		os.Exit(1)
	}

	cards, err := GetCardsForSet(os.Args[1])
	if err != nil {
		fatal("%s\n", err)
		os.Exit(1)
	}

	buildDoubletonCubeList(os.Args[1], cards)
}
