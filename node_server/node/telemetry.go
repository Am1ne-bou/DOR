package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

var dashURL = func() string {
	if v := os.Getenv("DASHBOARD_URL"); v != "" {
		return v
	}
	return "http://localhost:8888"
}()

type telEvent struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
	Msg  string `json:"msg,omitempty"`
	ExID string `json:"exid,omitempty"`
	Hop  int    `json:"hop,omitempty"` // ← position dans la route, 0 = non défini
}

func Telemetry(from, to, evType, msg, exid string, hop int) {
	go func() {
		b, err := json.Marshal(telEvent{From: from, To: to, Type: evType, Msg: msg, ExID: exid, Hop: hop})
		if err != nil {
			return
		}
		client := http.Client{Timeout: 300 * time.Millisecond}
		client.Post(dashURL+"/telemetry", "application/json", bytes.NewReader(b)) //nolint:errcheck
	}()
}
