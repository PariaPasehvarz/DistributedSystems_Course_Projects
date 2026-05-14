package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const server_port = ":8080"

type ComputeRequest struct {
	Operation string
	A         float64
	B         float64
}

type ComputeResponse struct {
	Status    string  `json:"status"`
	Operation string  `json:"operation,omitempty"`
	A         float64 `json:"a,omitempty"`
	B         float64 `json:"b,omitempty"`
	Result    float64 `json:"result,omitempty"`
	Error     string  `json:"error,omitempty"`
}

func main() {
	http.HandleFunc("/health", health_handler)
	http.HandleFunc("/compute", compute_handler)

	fmt.Printf("[server] Starting server on %s\n", server_port)

	if err := http.ListenAndServe(server_port, nil); err != nil {
		log.Fatalf("[server] Failed to start: %v", err)
	}
}

func health_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println("[server] Health check request received")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func compute_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req, ok := parse_compute_request(w, r)
	if !ok {
		return
	}

	log.Printf(
		"[server] Request received: op=%s a=%f b=%f",
		req.Operation,
		req.A,
		req.B,
	)

	res := process_operation(req)

	w.Header().Set("Content-Type", "application/json")

	if res.Status == "err" {
		w.WriteHeader(http.StatusBadRequest)
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Printf("[server] Failed to encode response: %v", err)
	}
}

func parse_compute_request(
	w http.ResponseWriter,
	r *http.Request,
) (ComputeRequest, bool) {

	query := r.URL.Query()

	op := strings.ToUpper(query.Get("op"))
	a_str := query.Get("a")
	b_str := query.Get("b")

	if op == "" || a_str == "" || b_str == "" {
		http.Error(w, "missing parameters", http.StatusBadRequest)
		return ComputeRequest{}, false
	}

	a, err := strconv.ParseFloat(a_str, 64)
	if err != nil {
		http.Error(w, "invalid a", http.StatusBadRequest)
		return ComputeRequest{}, false
	}

	b, err := strconv.ParseFloat(b_str, 64)
	if err != nil {
		http.Error(w, "invalid b", http.StatusBadRequest)
		return ComputeRequest{}, false
	}

	return ComputeRequest{
		Operation: op,
		A:         a,
		B:         b,
	}, true
}

func process_operation(req ComputeRequest) ComputeResponse {
	switch req.Operation {

	case "ADD":
		return ComputeResponse{
			Status:    "ok",
			Operation: req.Operation,
			A:         req.A,
			B:         req.B,
			Result:    req.A + req.B,
		}

	case "SUB":
		return ComputeResponse{
			Status:    "ok",
			Operation: req.Operation,
			A:         req.A,
			B:         req.B,
			Result:    req.A - req.B,
		}

	case "MUL":
		return ComputeResponse{
			Status:    "ok",
			Operation: req.Operation,
			A:         req.A,
			B:         req.B,
			Result:    req.A * req.B,
		}

	case "DIV":
		if req.B == 0 {
			return ComputeResponse{
				Status: "err",
				Error:  "division_by_zero",
			}
		}

		return ComputeResponse{
			Status:    "ok",
			Operation: req.Operation,
			A:         req.A,
			B:         req.B,
			Result:    req.A / req.B,
		}

	default:
		return ComputeResponse{
			Status: "err",
			Error:  "unknown_operation",
		}
	}
}