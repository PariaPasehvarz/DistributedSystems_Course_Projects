package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"replica/config"
	"replica/handler"
	"replica/store"
)

func main() {
	configPath := flag.String("config", "", "Path to replica config file (required)")
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: replica -config <path-to-config.json>")
		os.Exit(1)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}


	cfgJSON, _ := json.MarshalIndent(cfg, "", "  ")
	log.Printf("Starting replica with config:\n%s\n", cfgJSON)


	kvStore := store.New(cfg.ReplicaID)


	h := handler.New(kvStore, cfg)

	mux := http.NewServeMux()


	mux.HandleFunc("/put", h.HandlePut)
	mux.HandleFunc("/get", h.HandleGet)


	mux.HandleFunc("/internal/replicate", h.HandleReplicate)


	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"replica": cfg.ReplicaID,
			"status":  "ok",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("[%s] Listening on %s (mode: %s)", cfg.ReplicaID, addr, cfg.ConsistencyMode)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
