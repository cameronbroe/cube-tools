package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

const scryfallBaseAPIURL = "https://api.scryfall.com/cards"

type ScryfallCard struct {
	ID              string `json:"id"`
	URL             string `json:"scryfall_uri"`
	PrintsSearchURL string `json:"prints_search_uri"`
	Name            string `json:"name"`

	Oversized bool `json:"oversized"`
	Promo     bool `json:"promo"`
	Reprint   bool `json:"reprint"`
	Variation bool `json:"variation"`
	Digital   bool `json:"digital"`

	Prices struct {
		Normal string `json:"usd"`
		Foil   string `json:"usd_foil"`
	} `json:"prices"`
}

func getScryfallCard(card CubeCobraCard) (*ScryfallCard, error) {
	scryfallCardAPIURL, err := url.JoinPath(scryfallBaseAPIURL, card.Details.ScryfallID)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, scryfallCardAPIURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "cubecobra-price-checker/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var scryfallCard ScryfallCard
	if err := json.Unmarshal(body, &scryfallCard); err != nil {
		return nil, err
	}
	return &scryfallCard, nil
}

func getScryfallPrintSearch(card *ScryfallCard) ([]ScryfallCard, error) {
	req, err := http.NewRequest(http.MethodGet, card.PrintsSearchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "cubecobra-price-checker/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type printSearchResponse struct {
		Data []ScryfallCard `json:"data"`
	}

	var parsedResponse printSearchResponse
	if err := json.Unmarshal(body, &parsedResponse); err != nil {
		return nil, err
	}
	return parsedResponse.Data, nil
}

func isValidPrinting(card *ScryfallCard) bool {
	if card.Oversized {
		return false
	}

	if card.Digital {
		return false
	}

	if card.Prices.Normal == "" && card.Prices.Foil == "" {
		return false
	}

	return true
}

func getAllValidPrintings(card *ScryfallCard) ([]*ScryfallCard, error) {
	var validPrintings []*ScryfallCard

	allPrintings, err := getScryfallPrintSearch(card)
	if err != nil {
		return nil, err
	}

	for _, printing := range allPrintings {
		if isValidPrinting(&printing) {
			validPrintings = append(validPrintings, &printing)
		}
	}

	return validPrintings, nil
}

func GetScryfallDetails(card CubeCobraCard) ([]*ScryfallCard, error) {
	var scryfallCards []*ScryfallCard

	baseScryfallCard, err := getScryfallCard(card)
	if err != nil {
		return scryfallCards, err
	}

	return getAllValidPrintings(baseScryfallCard)
}
