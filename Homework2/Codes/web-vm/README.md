# Web Server Subsystem (VM1)

## Overview
This directory contains the primary entry point and user-facing interface of our distributed system architecture running on **VM1**. The web server acts as a centralized application gateway that orchestrates remote procedure interactions across detached infrastructure nodes, safely separates user records from public surfaces, and embeds automated system health telemetry loops.

---

## Distributed Architecture Links

* **Authentication Link (VM1 -> VM2):** Handled using a strict standard network socket connection via `JSON-RPC`. The web layer holds zero persistent data arrays regarding usernames or password definitions.
* **Asset Retrieval Link (VM1 -> VM3):** Powered by an optimized, low-latency native `Go-RPC` stream to pull deep-space imagery blocks across the cluster fabric upon validation check passes.
* **Telemetry Monitoring Engine (VM1 -> VM4):** A detached background watchdog execution thread that continuously reads active runtime metrics and issues structured event dispatches when utilization limits are breached.

---

## Core System Implementations

### 1. Dynamic Memory Telemetry & Leak Simulator
The server instantiates a background execution loop (`time.Ticker`) operating at a fixed interval of 2 seconds. 
* **State Assessment:** It programmatically samples core heap allocation layers utilizing Go's `runtime.ReadMemStats`.
* **Threshold Logic:** If total active memory allocation crosses the high-utilization safety threshold of **300 MB**, it instantiates a non-blocking asynchronous network payload request.
* **Ingestion Webhook:** It serializes a structured level-triggered warning schema to the centralized event broker instance at VM4 on port **8084**.
* **Global Variable Anchoring:** To simulate memory leaks realistically, user-triggered programmatic resource blocks are appended directly to a globally pinned matrix variable, ensuring they are intentionally protected from Go garbage collection passes.

### 2. Secure Remote Asset Streaming
Files and image records are completely omitted from the storage layout of VM1. When an authenticated user requests an asset, the file identity parameter is structured into an RPC argument, transmitted over the wire to VM3, unpacked from raw block byte clusters, and served to the browser dynamically using an inline streaming layout.

---

## HTTP Endpoint Routing Matrix

| Route Path | Allowed Verbs | Access Clearance | Operational Description |
| :--- | :--- | :--- | :--- |
| `/` | `GET` | Public | Automatically routes unauthenticated traffic to the application interface. |
| `/login` | `GET`, `POST` | Public | Serves the login view; pushes authentication payloads down to the VM2 network cluster via `JSON-RPC`. |
| `/dashboard` | `GET` | Authenticated Only | Renders the primary system cockpit panel, memory monitoring metrics, and active image elements. |
| `/view-file` | `GET` | Authenticated Only | Accepts folder paths and filename parameters, querying VM3 dynamically using `Go-RPC` to pipe image streams. |
| `/consume-memory` | `POST` | Authenticated Only | Testing endpoint designed to intentionally push mock allocation sizes into global scope to test system threshold alerts. |

---

## Project Directory Layout
```text
web-vm/
├── main.go               # Application entry point, HTTP routing engine, and telemetry worker loop
├── templates/            # Core presentation UI views
│   ├── login.html        # Secure portal login interface structure
│   └── dashboard.html    # Main control cockpit panel and deep-space galaxy layout
└── static/               # Style layers
    └── styles.css        # Layout styling rules and element definitions

```

---

## How to Run

### Prerequisites

1. Ensure the Go SDK is installed and configured on the VM1 operating system environment.
2. Verify that network destination configurations inside `main.go` point to the correct explicit IP addresses of VM2, VM3, and VM4 instead of using localhost loopbacks.

### Execution Steps

1. Open a terminal session on VM1.
2. Navigate directly to your workspace repository:
```bash
cd ~/web-vm

```


3. Boot up the application execution process using the Go toolchain:
```bash
go run main.go -auth-ip=<AUTH_IP> -file-ip=<FILE_IP> -pubsub-ip=<PUBSUB_IP>
```


4. Verify from the terminal stdout logs that the services bound successfully to the network interface:
```text
2026/06/01 17:44:52 Initializing Gateway Routing Configurations...
2026/06/01 17:44:52 -> Auth Server target (VM2): 192.168.176.130:8082
2026/06/01 17:44:52 -> File Server target (VM3): 192.168.176.131:8083
2026/06/01 17:44:52 -> PubSub Server target (VM4): 192.168.176.132:8084
2026/06/01 17:44:52 Web Server running on port 8080...

```


5. Open a web browser on your host system and navigate to `http://<VM1-IP>:8080` to access the live system dashboard interface.

```
```
