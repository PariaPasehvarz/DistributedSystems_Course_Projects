package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"sync"
)

var (
	subscribers = make(map[net.Conn]bool)
	stateMutex  sync.Mutex
)

func main() {
	go startSubscriberListener()

	http.HandleFunc("/alerts", handleIncomingAlerts)

	log.Println("Pub/Sub Broker Engine initiated on VM 4...")
	log.Println("-> Ingestion webhook pipeline active on HTTP port :8084")
	if err := http.ListenAndServe(":8084", nil); err != nil {
		log.Fatalf("Critical Failure: Unable to bind HTTP ingestion engine: %v", err)
	}
}

func startSubscriberListener() {
	listener, err := net.Listen("tcp", ":8085")
	if err != nil {
		log.Fatalf("Critical Failure: Unable to bind TCP broadcast port: %v", err)
	}
	defer listener.Close()
	log.Println("-> Broadcast routing matrix active on TCP port :8085")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Network Error: Failed to map incoming subscriber: %v", err)
			continue
		}

		stateMutex.Lock()
		subscribers[conn] = true
		stateMutex.Unlock()
		log.Printf("Subscriber connected successfully from network node: %s", conn.RemoteAddr().String())
	}
}

func handleIncomingAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	payloadBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to parse incoming payload stream", http.StatusInternalServerError)
		return
	}

	log.Printf("Telemetry Alert received from Web Server! Fanout broadcasting to active listeners...")

	stateMutex.Lock()
	defer stateMutex.Unlock()
	for conn := range subscribers {
		_, err := conn.Write(append(payloadBytes, '\n'))
		if err != nil {
			log.Printf("Subscriber node %s disconnected unexpectedly. Purging stream entry.", conn.RemoteAddr().String())
			conn.Close()
			delete(subscribers, conn)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Telemetry alert broadcast successfully complete"))
}