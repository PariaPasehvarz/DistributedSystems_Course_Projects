package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	brokerIP := flag.String("broker-ip", "", "The network IPv4 address of the Pub/Sub Broker Machine (VM4)")
	flag.Parse()

	if *brokerIP == "" {
		log.Println("[-] Operational Error: Missing required network target configuration.")
		log.Println("-> Usage Example: go run subscriber.go -broker-ip=192.168.176.130")
		os.Exit(1)
	}

	brokerTargetAddress := fmt.Sprintf("%s:8085", *brokerIP)

	log.Printf("[+] Establishing subscription link with Pub/Sub Broker at %s...", brokerTargetAddress)

	conn, err := net.Dial("tcp", brokerTargetAddress)
	if err != nil {
		log.Fatalf("[-] Network Link Failure: Unable to establish connection to Broker node: %v", err)
	}
	defer conn.Close()

	log.Println("[+] Subscription link active! Listening for real-time telemetry alert frames...")

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("[-] Connection terminated by the Broker cluster engine: %v", err)
		}

		fmt.Printf("\n🚨 [SYSTEM WARN] TELEMETRY EVENT INGESTED:\n%s\n", line)
	}
}