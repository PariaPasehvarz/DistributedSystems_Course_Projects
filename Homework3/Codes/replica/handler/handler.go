package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"replica/config"
	"replica/replicator"
	"replica/store"
)

type Handler struct {
	store       *store.Store
	replicator  *replicator.Replicator
	cfg         *config.Config
}

func New(kvStore *store.Store, cfg *config.Config) *Handler {
	rep := replicator.New(cfg.Peers, cfg.NetworkDelayMs)
	return &Handler{
		store:      kvStore,
		replicator: rep,
		cfg:        cfg,
	}
}

type putRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type apiResponse struct {
	Success   bool        `json:"success"`
	ReplicaID string      `json:"replica_id"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	LatencyMs float64     `json:"latency_ms"`
}

func (h *Handler) HandlePut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	var req putRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Key == "" {
		writeError(w, "key is required", http.StatusBadRequest)
		return
	}

	entry := h.store.Put(req.Key, req.Value)
	log.Printf("[%s] PUT key=%q value=%q version=%d", h.cfg.ReplicaID, req.Key, req.Value, entry.Version)

	switch h.cfg.ConsistencyMode {
	case config.ModeEventual:
		h.replicator.SendAsync(entry)
		writeSuccess(w, entry, time.Since(start))

	case config.ModeStrong:
		if err := h.replicator.SendQuorum(entry); err != nil {
			log.Printf("[%s] PUT quorum failed for key=%q: %v", h.cfg.ReplicaID, req.Key, err)
			writeError(w, "quorum not reached: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		writeSuccess(w, entry, time.Since(start))
	}
}

func (h *Handler) HandleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, "query param 'key' is required", http.StatusBadRequest)
		return
	}

	entry := h.store.Get(key)
	if entry == nil {
		writeError(w, "key not found", http.StatusNotFound)
		return
	}

	log.Printf("[%s] GET key=%q value=%q version=%d", h.cfg.ReplicaID, key, entry.Value, entry.Version)
	writeSuccess(w, entry, time.Since(start))
}

func (h *Handler) HandleReplicate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var incoming store.Entry
	if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
		writeError(w, "invalid JSON body: "+err.Error(), http.StatusBadRequest)
		return
	}

	result := h.store.ApplyReplication(&incoming)

	if result.Skipped {
		log.Printf("[%s] REPLICATE skipped key=%q: %s", h.cfg.ReplicaID, incoming.Key, result.SkipMsg)
	} else if result.Conflict {
		log.Printf("[%s] REPLICATE conflict key=%q resolved via LWW → value=%q (v%d by %s)",
			h.cfg.ReplicaID, incoming.Key, result.Entry.Value, result.Entry.Version, result.Entry.UpdatedBy)
	} else {
		log.Printf("[%s] REPLICATE applied key=%q value=%q version=%d",
			h.cfg.ReplicaID, incoming.Key, incoming.Value, incoming.Version)
	}

	writeSuccess(w, result.Entry, 0)
}


func writeSuccess(w http.ResponseWriter, data interface{}, latency time.Duration) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiResponse{
		Success:   true,
		Data:      data,
		LatencyMs: float64(latency.Microseconds()) / 1000.0,
	})
}

func writeError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiResponse{
		Success: false,
		Error:   msg,
	})
}
