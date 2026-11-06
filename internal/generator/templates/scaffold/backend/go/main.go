package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"message": "Hello from dev-box API (Go)"}); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		if _, err := fmt.Fprint(w, "# HELP devbox_up Whether the API process is running.\n# TYPE devbox_up gauge\ndevbox_up 1\n"); err != nil {
			http.Error(w, "failed to encode metrics", http.StatusInternalServerError)
		}
	})
	// A server without ReadHeaderTimeout lets a slow client hold a connection open indefinitely.
	server := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("Server starting on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
