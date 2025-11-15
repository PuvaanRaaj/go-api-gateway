package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type resp struct {
	Service    string `json:"service"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
	RequestURI string `json:"request_uri"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handle)

	addr := ":" + getPort()
	log.Printf("service-b listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handle(w http.ResponseWriter, r *http.Request) {
	log.Printf("service-b received %s %s rid=%s", r.Method, r.URL.Path, r.Header.Get("X-Request-ID"))
	out := resp{
		Service:    "service-b",
		Message:    "hello from service B",
		RequestID:  r.Header.Get("X-Request-ID"),
		RequestURI: r.RequestURI,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func getPort() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}
