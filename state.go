package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

func cachePath(name string) string {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "rlp", name)
}

func SaveCurrentStation(s Station) error {
	path := cachePath("current.json")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadCurrentStation() (Station, error) {
	data, err := os.ReadFile(cachePath("current.json"))
	if err != nil {
		return Station{}, err
	}
	var s Station
	return s, json.Unmarshal(data, &s)
}

const listCacheTTL = 24 * time.Hour

func SaveListCache(name string, items []string) error {
	path := cachePath(name + ".json")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadListCache(name string) ([]string, error) {
	path := cachePath(name + ".json")
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if time.Since(info.ModTime()) > listCacheTTL {
		return nil, os.ErrNotExist
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []string
	return items, json.Unmarshal(data, &items)
}
