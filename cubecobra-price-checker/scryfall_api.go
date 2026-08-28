package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

const dataFilePath = "./DefaultCards.jsonl"

type ScryfallBulkData []*ScryfallCard

var bulkData ScryfallBulkData
var bulkDataLoaded bool = false

var bulkDataIDToOracleIDMap map[string]string
var bulkDataIDToOracleIDMapLoaded bool = false

var bulkDataOracleIDToCardsMap map[string][]*ScryfallCard
var bulkDataOracleIDToCardsMapLoaded bool = false

func (s ScryfallBulkData) Len() int { return len(s) }

func (s ScryfallBulkData) FindOracleIDByID(id string) (string, error) {
	oracleID, ok := bulkDataIDToOracleIDMap[id]
	if !ok {
		return "", fmt.Errorf("could not find card with ID: %s", id)
	}
	return oracleID, nil
}

func (s ScryfallBulkData) FindCardsByOracleID(oracleID string) ([]*ScryfallCard, error) {
	cards, ok := bulkDataOracleIDToCardsMap[oracleID]
	if !ok {
		return nil, fmt.Errorf("could not find cards with oracle ID: %s", oracleID)
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
	if bulkDataLoaded {
		return nil
	}

	file, err := os.Open(dataFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	maxCapacity := 256 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	for scanner.Scan() {
		card := ScryfallCard{}
		err = json.Unmarshal(scanner.Bytes(), &card)
		if err != nil {
			return err
		}
		bulkData = append(bulkData, &card)
	}

	bulkDataLoaded = true
	return nil
}

func buildIDToOracleIDMap() error {
	if bulkDataIDToOracleIDMapLoaded {
		return nil
	}

	if len(bulkData) == 0 {
		return fmt.Errorf("load Scryfall bulk data first")
	}

	bulkDataIDToOracleIDMap = make(map[string]string)
	for _, card := range bulkData {
		bulkDataIDToOracleIDMap[card.ID] = card.OracleID
	}

	bulkDataIDToOracleIDMapLoaded = true
	return nil
}

func buildOracleIDToCardsMap() error {
	if bulkDataOracleIDToCardsMapLoaded {
		return nil
	}

	if len(bulkData) == 0 {
		return fmt.Errorf("load Scryfall bulk data first")
	}

	bulkDataOracleIDToCardsMap = make(map[string][]*ScryfallCard)
	for _, card := range bulkData {
		if bulkDataOracleIDToCardsMap[card.OracleID] == nil {
			bulkDataOracleIDToCardsMap[card.OracleID] = []*ScryfallCard{}
		}

		bulkDataOracleIDToCardsMap[card.OracleID] = append(bulkDataOracleIDToCardsMap[card.OracleID], card)
	}

	bulkDataOracleIDToCardsMapLoaded = true
	return nil
}

func getOracleID(card CubeCobraCard) (string, error) {
	return bulkData.FindOracleIDByID(card.Details.ScryfallID)
}

func getAllPrintings(oracleID string) ([]*ScryfallCard, error) {
	return bulkData.FindCardsByOracleID(oracleID)
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

func getAllValidPrintings(oracleID string) ([]*ScryfallCard, error) {
	var validPrintings []*ScryfallCard

	allPrintings, err := getAllPrintings(oracleID)
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

	err = buildIDToOracleIDMap()
	if err != nil {
		return nil, err
	}

	err = buildOracleIDToCardsMap()
	if err != nil {
		return nil, err
	}

	var scryfallCards []*ScryfallCard

	oracleID, err := getOracleID(card)
	if err != nil {
		return scryfallCards, err
	}

	return getAllValidPrintings(oracleID)
}
