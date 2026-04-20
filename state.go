package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

func savePID(pid int) {
	path := cachePath("mpv.pid")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
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

func isMpvRunning() bool {
	data, err := os.ReadFile(cachePath("mpv.pid"))
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// signal 0 checks if process exists without killing it
	return p.Signal(syscall.Signal(0)) == nil
}

func killSavedMpv() {
	data, err := os.ReadFile(cachePath("mpv.pid"))
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return
	}
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
	_ = os.Remove(cachePath("mpv.pid"))
}
