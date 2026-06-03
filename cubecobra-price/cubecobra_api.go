package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

const apiBaseURL = "https://cubecobra.com/cube/api/cubeJSON"

type Cube struct {
	Name  string `json:"name"`
	Cards struct {
		Mainboard  []Card `json:"mainboard"`
		Maybeboard []Card `json:"maybeboard"`
	} `json:"cards"`
}

type Card struct {
	Details struct {
		Name       string `json:"name"`
		ScryfallID string `json:"scryfall_id"`
	}
}

func GetCube(cubeID string) (*Cube, error) {
	cubeAPIURL, err := url.JoinPath(apiBaseURL, cubeID)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(cubeAPIURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var cube Cube
	err = json.Unmarshal(body, &cube)
	if err != nil {
		return nil, err
	}
	return &cube, nil
}
