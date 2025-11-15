package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type resp struct {
	Service    string            `json:"service"`
	Message    string            `json:"message"`
	RequestID  string            `json:"request_id"`
	RequestURI string            `json:"request_uri"`
	Headers    map[string]string `json:"headers"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handle)

	addr := ":" + getPort()
	log.Printf("service-a listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handle(w http.ResponseWriter, r *http.Request) {
	log.Printf("service-a received %s %s rid=%s", r.Method, r.URL.Path, r.Header.Get("X-Request-ID"))
	headers := map[string]string{}
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	out := resp{
		Service:    "service-a",
		Message:    "hello from service A",
		RequestID:  r.Header.Get("X-Request-ID"),
		RequestURI: r.RequestURI,
		Headers:    headers,
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
