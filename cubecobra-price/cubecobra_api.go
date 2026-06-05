package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

const cubeCobraBaseAPIURL = "https://cubecobra.com/cube/api/cubeJSON"

type CubeCobraCube struct {
	Name  string `json:"name"`
	Cards struct {
		Mainboard  []CubeCobraCard `json:"mainboard"`
		Maybeboard []CubeCobraCard `json:"maybeboard"`
	} `json:"cards"`
}

type CubeCobraCard struct {
	Details struct {
		Name       string `json:"name"`
		ScryfallID string `json:"scryfall_id"`
	}
}

func GetCube(cubeID string) (*CubeCobraCube, error) {
	cubeAPIURL, err := url.JoinPath(cubeCobraBaseAPIURL, cubeID)
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

	var cube CubeCobraCube
	err = json.Unmarshal(body, &cube)
	if err != nil {
		return nil, err
	}
	return &cube, nil
}
