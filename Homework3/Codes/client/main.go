package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"bytes"
	"time"
)

type apiResponse struct {
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
	LatencyMs float64         `json:"latency_ms"`
}

type entry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Version   int64  `json:"version"`
	UpdatedBy string `json:"updated_by"`
	UpdatedAt string `json:"updated_at"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(addr string) *Client {
	return &Client{
		baseURL: addr,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Put(key, value string) {
	body, _ := json.Marshal(map[string]string{"key": key, "value": value})
	start := time.Now()
	resp, err := c.httpClient.Post(c.baseURL+"/put", "application/json", bytes.NewReader(body))
	elapsed := time.Since(start)
	if err != nil {
		fmt.Printf(" PUT error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	c.printResponse("PUT", resp, elapsed)
}

func (c *Client) Get(key string) {
	start := time.Now()
	resp, err := c.httpClient.Get(c.baseURL + "/get?key=" + url.QueryEscape(key))
	elapsed := time.Since(start)
	if err != nil {
		fmt.Printf(" GET error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	c.printResponse("GET", resp, elapsed)
}

func (c *Client) Health() {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		fmt.Printf(" HEALTH error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	fmt.Printf(" replica=%s  status=%s  time=%s\n", body["replica"], body["status"], body["time"])
}

func (c *Client) printResponse(op string, resp *http.Response, elapsed time.Duration) {
	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		fmt.Printf(" %s: could not decode response: %v\n", op, err)
		return
	}

	if !apiResp.Success {
		fmt.Printf(" %s failed (HTTP %d): %s\n", op, resp.StatusCode, apiResp.Error)
		return
	}

	var e entry
	if apiResp.Data != nil {
		json.Unmarshal(apiResp.Data, &e)
	}

	fmt.Printf(" %s  key=%q  value=%q  version=%d  by=%s  round_trip=%s\n",
		op, e.Key, e.Value, e.Version, e.UpdatedBy, elapsed.Round(time.Millisecond))
}


func printUsage() {
	fmt.Println(`
Commands:
  PUT <key> <value>   Write a value (sent to the connected replica)
  GET <key>           Read a value (from the connected replica)
  HEALTH              Check replica health
  SWITCH <addr>       Switch to a different replica  (e.g. SWITCH http://localhost:8082)
  HELP                Show this help
  EXIT                Quit
`)
}

func main() {
	addr := flag.String("addr", "http://localhost:8080", "Replica address (e.g. http://localhost:8080)")
	flag.Parse()

	client := NewClient(*addr)
	fmt.Printf("Connected to replica at %s\n", *addr)
	printUsage()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("[%s] > ", *addr)
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "PUT":
			if len(parts) < 3 {
				fmt.Println("  Usage: PUT <key> <value>")
				continue
			}
			client.Put(parts[1], strings.Join(parts[2:], " "))

		case "GET":
			if len(parts) < 2 {
				fmt.Println("  Usage: GET <key>")
				continue
			}
			client.Get(parts[1])

		case "HEALTH":
			client.Health()

		case "SWITCH":
			if len(parts) < 2 {
				fmt.Println("  Usage: SWITCH <addr>")
				continue
			}
			*addr = parts[1]
			client = NewClient(*addr)
			fmt.Printf("  Switched to %s\n", *addr)

		case "HELP":
			printUsage()

		case "EXIT", "QUIT":
			fmt.Println("Bye!")
			return

		default:
			fmt.Printf("  Unknown command %q. Type HELP for usage.\n", cmd)
		}
	}
}
