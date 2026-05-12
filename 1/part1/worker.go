package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
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
	for {
		req_file, res_file, ok := create_pipes()
		if !ok {
			return
		}

		scanner := bufio.NewScanner(req_file)

		fmt.Println("[worker] Ready. Waiting for requests...")
		running := true
		for running {
			req, status := read_operation_req_from_pipe(scanner)
			switch status {
			case "ok":
				res := process_operation_req(req)
				if !write_operation_res_to_pipe(res_file, res) {
					fmt.Println("[worker] Interface closed unexpectedly while sending response.")
					running = false
				}
			case "disconnected":
				fmt.Println("[worker] Interface disconnected. Waiting for a new connection...")
				running = false
			case "pipe_error":
				fmt.Println("[worker] Pipe error. Waiting for a new connection...")
				running = false
			case "skip":
				// empty or malformed line — keep going
			}
		}

		req_file.Close()
		res_file.Close()

		fmt.Println("[worker] Restarting pipes...")
	}
}

func create_pipes() (*os.File, *os.File, bool) {
	os.Remove(request_pipe)
	os.Remove(response_pipe)

	if err := syscall.Mkfifo(request_pipe, 0600); err != nil {
		log.Printf("[worker] Failed to create request pipe: %v", err)
		return nil, nil, false
	}
	if err := syscall.Mkfifo(response_pipe, 0600); err != nil {
		log.Printf("[worker] Failed to create response pipe: %v", err)
		return nil, nil, false
	}

	fmt.Println("[worker] Pipes created. Waiting for interface to connect...")

	req_file, err := os.Open(request_pipe)
	if err != nil {
		log.Printf("[worker] Failed to open request pipe: %v", err)
		return nil, nil, false
	}

	res_file, err := os.OpenFile(response_pipe, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		log.Printf("[worker] Failed to open response pipe: %v", err)
		req_file.Close()
		return nil, nil, false
	}

	fmt.Println("[worker] Interface connected!")
	return req_file, res_file, true
}

func read_operation_req_from_pipe(scanner *bufio.Scanner) (Request, string) {
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Printf("[worker] Pipe read error: %v", err)
			return Request{}, "pipe_error"
		}
		return Request{}, "disconnected"
	}

	line := strings.TrimSpace(scanner.Text())
	if line == "" {
		return Request{}, "skip"
	}

	var req Request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		log.Printf("[worker] Invalid JSON: %q — %v", line, err)
		return Request{}, "skip"
	}

	log.Printf("[worker] Received: op=%s a=%d b=%d", req.Operation, req.Operand1, req.Operand2)
	return req, "ok"
}

func process_operation_req(req Request) Response {
	switch strings.ToUpper(req.Operation) {
	case "ADD":
		return Response{Status: "ok", Result: req.Operand1 + req.Operand2}
	case "SUB":
		return Response{Status: "ok", Result: req.Operand1 - req.Operand2}
	case "MUL":
		return Response{Status: "ok", Result: req.Operand1 * req.Operand2}
	case "DIV":
		if req.Operand2 == 0 {
			return Response{Status: "err", Error: "division_by_zero"}
		}
		return Response{Status: "ok", Result: req.Operand1 / req.Operand2}
	case "":
		return Response{Status: "err", Error: "empty_operation"}
	default:
		return Response{Status: "err", Error: "unknown_operation: " + req.Operation}
	}
}

func write_operation_res_to_pipe(res_file *os.File, res Response) bool {
	data, err := json.Marshal(res)
	if err != nil {
		log.Printf("[worker] Failed to marshal response: %v", err)
		return true 
	}

	data = append(data, '\n')

	if _, err := res_file.Write(data); err != nil {
		if isEPIPE(err) {
			log.Printf("[worker] Broken pipe — interface has disconnected.")
		} else {
			log.Printf("[worker] Write error: %v", err)
		}
		return false
	}

	log.Printf("[worker] Sent: %s", strings.TrimSpace(string(data)))
	return true
}

func isEPIPE(err error) bool {
	return strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "pipe")
}