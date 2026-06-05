package main

import (
	"encoding/json"
	"fmt"
	"os"
)

const dataFilePath = "./DefaultCards.json"

type ScryfallBulkData []*ScryfallCard

var bulkData ScryfallBulkData

func (s ScryfallBulkData) Len() int { return len(s) }

func (s ScryfallBulkData) FindCardById(id string) (*ScryfallCard, error) {
	for _, card := range s {
		if card.ID == id {
			return card, nil
		}
	}
	return nil, fmt.Errorf("could not find card with ID: %s", id)
}

func (s ScryfallBulkData) FindCardsByOracleID(oracleID string) ([]*ScryfallCard, error) {
	var cards []*ScryfallCard
	for _, card := range s {
		if card.OracleID == oracleID {
			cards = append(cards, card)
		}
	}
	return cards, nil
}

type ScryfallCard struct {
	ID              string `json:"id"`
	OracleID        string `json:"oracle_id"`
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

func loadScryfallCards() error {
	fileContents, err := os.ReadFile(dataFilePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(fileContents, &bulkData)
	if err != nil {
		return err
	}

	return nil
}

func getScryfallCard(card CubeCobraCard) (*ScryfallCard, error) {
	return bulkData.FindCardById(card.Details.ScryfallID)
}

func getAllPrintings(card *ScryfallCard) ([]*ScryfallCard, error) {
	return bulkData.FindCardsByOracleID(card.OracleID)
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

	allPrintings, err := getAllPrintings(card)
	if err != nil {
		return nil, err
	}

	for _, printing := range allPrintings {
		if isValidPrinting(printing) {
			validPrintings = append(validPrintings, printing)
		}
	}

	return validPrintings, nil
}

func GetScryfallDetails(card CubeCobraCard) ([]*ScryfallCard, error) {
	err := loadScryfallCards()
	if err != nil {
		return nil, err
	}

	var scryfallCards []*ScryfallCard

	baseScryfallCard, err := getScryfallCard(card)
	if err != nil {
		return scryfallCards, err
	}

	return getAllValidPrintings(baseScryfallCard)
}
