package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const apiBase = "https://de1.api.radio-browser.info/json"

var Genres = []string{
	"Adult Contemporary", "Alternative Rock", "Blues", "Broadway", "Chill",
	"Classical", "Country", "Dance", "Holiday Music", "Jazz", "Latin",
	"Oldies", "Pop Hits", "Reggae", "Rock", "Soundtracks", "Talk",
	"World Music", "80s", "90s", "00s", "10s",
}

var genreTags = map[string]string{
	"Adult Contemporary": "adult contemporary",
	"Alternative Rock":   "alternative",
	"Blues":              "blues",
	"Broadway":           "broadway",
	"Chill":              "chill",
	"Classical":          "classical",
	"Country":            "country",
	"Dance":              "dance",
	"Holiday Music":      "christmas",
	"Jazz":               "jazz",
	"Latin":              "latin",
	"Oldies":             "oldies",
	"Pop Hits":           "pop",
	"Reggae":             "reggae",
	"Rock":               "rock",
	"Soundtracks":        "soundtrack",
	"Talk":               "talk",
	"World Music":        "world",
	"80s":                "80s",
	"90s":                "90s",
	"00s":                "2000s",
	"10s":                "2010s",
}

type Station struct {
	Name    string `json:"name"`
	Country string `json:"country"`
	Bitrate int    `json:"bitrate"`
	URL     string `json:"url_resolved"`
}

type countryItem struct {
	Name         string `json:"name"`
	StationCount int    `json:"stationcount"`
}

type languageItem struct {
	Name         string `json:"name"`
	StationCount int    `json:"stationcount"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

func FetchStationsByGenre(genre string) ([]Station, error) {
	tag, ok := genreTags[genre]
	if !ok {
		return nil, fmt.Errorf("unknown genre: %s", genre)
	}
	return fetchStations("bytag", tag)
}

func FetchStationsByCountry(country string) ([]Station, error) {
	return fetchStations("bycountry", country)
}

func FetchStationsByLanguage(language string) ([]Station, error) {
	return fetchStations("bylanguage", language)
}

func FetchStationsByName(query string) ([]Station, error) {
	u := fmt.Sprintf("%s/stations/search?name=%s&limit=50&order=clickcount&reverse=true&hidebroken=true",
		apiBase, url.QueryEscape(query))
	return fetchStationsURL(u)
}

func fetchStations(endpoint, value string) ([]Station, error) {
	u := fmt.Sprintf("%s/stations/%s/%s?limit=50&order=clickcount&reverse=true&hidebroken=true",
		apiBase, endpoint, url.PathEscape(value))
	return fetchStationsURL(u)
}

func fetchStationsURL(u string) ([]Station, error) {
	resp, err := httpClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stations []Station
	if err := json.NewDecoder(resp.Body).Decode(&stations); err != nil {
		return nil, err
	}
	var result []Station
	for _, s := range stations {
		if s.URL != "" {
			result = append(result, s)
		}
	}
	return result, nil
}

func FetchCountries() ([]string, error) {
	u := apiBase + "/countries?order=stationcount&reverse=true&hidebroken=true"
	resp, err := httpClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var items []countryItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(items))
	for _, c := range items {
		if c.Name != "" {
			names = append(names, c.Name)
		}
	}
	return names, nil
}

func FetchLanguages() ([]string, error) {
	u := apiBase + "/languages?order=stationcount&reverse=true&hidebroken=true"
	resp, err := httpClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var items []languageItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(items))
	for _, l := range items {
		if l.Name != "" {
			names = append(names, l.Name)
		}
	}
	return names, nil
}
