# Pub/Sub Telemetry Broker and Subscriber Node (VM4)

## Overview
This directory contains the decoupled, event-driven monitoring system running on **VM4** that is designed to handle memory utilization alerts asynchronously. To separate concerns and optimize resource tracking, the web server (VM1) offloads system alert logs to this decoupled message broker. The broker ingests events from the web layer and instantly broadcasts them down to real-time command-line subscriber instances.

---

## Architectural Specifications

This node splits processing duties between two complementary backend applications:

### 1. The Event Broker (broker.go)
Acts as the central network routing hub for cluster monitoring data. It opens two distinct concurrent communication lines:
* **HTTP Ingestion Interface (Port 8084):** Exposes a REST endpoint to accept inbound level-triggered alarm JSON structures dispatched by the VM1 background watchdog loop when memory crosses 300 MB.
* **TCP Event Broadcaster (Port 8085):** Maintains persistent streaming raw socket connection links to listening terminal instances, duplicating and delivering incoming ingestion streams down to consumers immediately.

### 2. The Monitoring Consumer (subscriber.go)
A lightweight observer utility used to track cluster events in real time. To eliminate hardcoding and allow migration across changing virtual machine subnets, it utilizes a command-line flag system to receive the broker's active network location dynamically at launch.

---

## Project Directory Layout
```text
pubsub-vm/
├── broker.go             # Message distribution engine handling HTTP webhooks and TCP streaming
├── subscriber.go         # Terminal-based client logging telemetry frames dynamically

```

---

## How to Run

### Prerequisites

1. Ensure the Go SDK environment is installed and configured on the VM4 operating system.
2. Confirm that ports 8084 and 8085 are accessible and not blocked by local firewall policies so that VM1 and external terminal users can establish communication lines.

### Execution Steps

#### Step A: Bootstrapping the Central Broker Matrix

1. Open a terminal session on VM4.
2. Move into your workspace repository path:
```bash
cd ~/pubsub-vm

```


3. Start up the broker distribution node:
```bash
go run broker.go

```


4. Confirm from stdout logs that the broker has successfully opened both internal communication interfaces:
```text
2026/06/01 17:45:07 Pub/Sub Broker Engine initiated on VM 4...
2026/06/01 17:45:07 -> Ingestion webhook pipeline active on HTTP port :8084
2026/06/01 17:45:07 -> Broadcast routing matrix active on TCP port :8085
2026/06/01 17:45:15 Subscriber connected successfully from network node: 192.168.176.132:53312

```



#### Step B: Deploying the Real-Time Event Subscriber

1. Open a separate, new terminal shell session (either on VM4 itself, or on another monitoring machine inside the cluster).
2. Move into the same workspace directory:
```bash
cd ~/pubsub-vm

```


3. Launch the subscriber process by supplying the exact, active network IPv4 address of the broker machine via the runtime flag:
```bash
go run subscriber.go -broker-ip=<BROKER-IP>

```


4. Verify that your subscriber application reports an active connection status:
```text
2026/06/01 17:45:15 [+] Establishing subscription link with Pub/Sub Broker at 192.168.176.132:8085...
2026/06/01 17:45:15 [+] Subscription link active! Listening for real-time telemetry alert frames...

```



Keep both terminal windows active. When VM1 encounters a high memory condition exceeding 300 MB, the broker log will display the incoming HTTP packet, and the subscriber interface will immediately output the formatted telemetry alert stream.

```

```