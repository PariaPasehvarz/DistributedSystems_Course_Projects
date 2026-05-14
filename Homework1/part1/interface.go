package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const request_pipe = "/tmp/hw1_part1_req_write"
const response_pipe = "/tmp/hw1_part1_res_write"

type Response struct {
	Status string `json:"status"`
	Result int    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type Request struct {
	Operation string `json:"operation"`
	Operand1  int    `json:"operand1"`
	Operand2  int    `json:"operand2"`
}

func main() {
	req_file, res_file := connect_to_pipes()
	defer req_file.Close()
	defer res_file.Close()

	stdin := bufio.NewScanner(os.Stdin)
	res_scanner := bufio.NewScanner(res_file)

	fmt.Println("[interface] Connected! Enter operations as: OP A B  (e.g. ADD 5 3)")
	fmt.Println("[interface] Supported: ADD, SUB, MUL, DIV — type 'exit' to quit.")

	for {
		req, ok := get_operation_from_user(stdin)
		if !ok {
			fmt.Println("[interface] Exiting.")
			return
		}

		if !write_operation_req_to_pipe(req_file, req) {
			fmt.Println("[interface] Lost connection to worker. Exiting.")
			return
		}

		if !read_operation_res_from_pipe(res_scanner) {
			fmt.Println("[interface] Worker closed unexpectedly before responding. Exiting.")
			return
		}
	}
}

func connect_to_pipes() (*os.File, *os.File) {
	const maxRetries = 5
	const retryDelay = 2 * time.Second

	fmt.Println("[interface] Connecting to worker...")

	var req_file *os.File
	var res_file *os.File

	for attempt := 1; attempt <= maxRetries; attempt++ {
		var err error
		req_file, err = os.OpenFile(request_pipe, os.O_WRONLY, os.ModeNamedPipe)
		if err != nil {
			if attempt == maxRetries {
				log.Fatalf("[interface] Worker is not running after %d attempts. Start worker.go first.\n", maxRetries)
			}
			fmt.Printf("[interface] Worker not ready (attempt %d/%d) — retrying in %v...\n",
				attempt, maxRetries, retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		break
	}

	var err error
	res_file, err = os.Open(response_pipe)
	if err != nil {
		log.Fatalf("[interface] Connected to request pipe but failed on response pipe: %v\n"+
			"Worker may have crashed during startup.", err)
	}

	fmt.Println("[interface] Connected to worker!")
	return req_file, res_file
}

func get_operation_from_user(stdin *bufio.Scanner) (Request, bool) {
	for {
		fmt.Print("\n> ")
		if !stdin.Scan() {
			return Request{}, false
		}

		line := strings.TrimSpace(stdin.Text())
		if line == "" {
			continue
		}
		if strings.ToLower(line) == "exit" {
			return Request{}, false
		}

		parts := strings.Fields(line)
		if len(parts) != 3 {
			fmt.Println("[interface] Error: expected format is OP A B (e.g. ADD 5 3)")
			continue
		}

		op := strings.ToUpper(parts[0])

		a, err := strconv.Atoi(parts[1])
		if err != nil {
			fmt.Printf("[interface] Error: '%s' is not a valid integer\n", parts[1])
			continue
		}

		b, err := strconv.Atoi(parts[2])
		if err != nil {
			fmt.Printf("[interface] Error: '%s' is not a valid integer\n", parts[2])
			continue
		}

		switch op {
		case "ADD", "SUB", "MUL", "DIV":
		default:
			fmt.Printf("[interface] Error: unknown operation '%s'. Use ADD SUB MUL DIV\n", op)
			continue
		}

		return Request{Operation: op, Operand1: a, Operand2: b}, true
	}
}

func write_operation_req_to_pipe(req_file *os.File, req Request) bool {
	data, err := json.Marshal(req)
	if err != nil {
		log.Printf("[interface] Failed to marshal request: %v", err)
		return true 
	}

	data = append(data, '\n')

	if _, err := req_file.Write(data); err != nil {
		log.Printf("[interface] Failed to send to worker (broken pipe): %v", err)
		return false
	}
	return true
}

func read_operation_res_from_pipe(res_scanner *bufio.Scanner) bool {
	if !res_scanner.Scan() {
		if err := res_scanner.Err(); err != nil {
			log.Printf("[interface] Pipe read error: %v", err)
		} else {
			log.Printf("[interface] Worker closed the connection (EOF on response pipe).")
		}
		return false
	}

	var res Response
	if err := json.Unmarshal([]byte(res_scanner.Text()), &res); err != nil {
		fmt.Printf("[interface] Error: could not parse worker response: %v\n", err)
		return true 
	}

	if res.Status == "ok" {
		fmt.Printf("[interface] Result: %d\n", res.Result)
	} else {
		fmt.Printf("[interface] Error from worker: %s\n", res.Error)
	}
	return true
}