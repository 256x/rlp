package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const mpvSocket = "/tmp/rlp-mpv.sock"

func fetchIcyTitle() (string, error) {
	conn, err := net.DialTimeout("unix", mpvSocket, 500*time.Millisecond)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(2 * time.Second))

	fmt.Fprintf(conn, "{\"command\":[\"get_property\",\"metadata/by-key/icy-title\"]}\n")

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var resp struct {
			Data  interface{} `json:"data"`
			Error string      `json:"error"`
		}
		if json.Unmarshal(scanner.Bytes(), &resp) == nil && resp.Error != "" {
			if resp.Error == "success" {
				if s, ok := resp.Data.(string); ok {
					return s, nil
				}
			}
			return "", fmt.Errorf("mpv: %s", resp.Error)
		}
	}
	return "", fmt.Errorf("no response")
}
