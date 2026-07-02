# Asset Storage File Service (VM3)

## Overview
This directory contains the independent Asset running on **VM3**. To adhere to distributed systems decoupling best practices, all heavy image binaries and files are completely absent from the web server (VM1) local storage disk. When an authorized user requests an image on the dashboard, VM1 acts as a proxy, fetching the raw byte data dynamically across the virtual private network from this dedicated storage layer.

---

## Architectural Specifications

* **RPC Transport Link:** Registers a custom object mapping service using Go's native net/rpc framework (`Go-RPC`) over port 8082. This optimized protocol allows low-latency, high-throughput streaming of raw block byte payloads directly across the cluster fabric.
* **Storage Isolation:** Hosts visual assets and high-resolution telemetry attachments (such as the James Webb Space Telescope deep-space galaxy imagery displayed on the client dashboard) away from public-facing nodes.

---

## Project Directory Layout
```text
file-vm/
├── main.go               # Framework listener initialization, folder mapping, and byte stream handlers
├── files             
└── imagse
```

---

## How to Run

### Prerequisites

1. Ensure the Go SDK environment is installed and configured on the VM3 operating system.
2. Confirm that port 8082 is accessible and not blocked by local firewall policies so that VM1 can establish inbound TCP socket connections.
3. Verify that the `files/` or `images/` directory contains the required files before launching the server.

### Execution Steps

1. Open a terminal session on VM3.
2. Navigate directly to your workspace repository:
```bash
cd ~/file-vm

```


3. Boot up the application node using the Go toolchain:
```bash
go run main.go

```


4. Verify from the terminal stdout logs that the native RPC listener has started:
```text
2026/06/01 17:44:21 Flexible File Server running on port 8083 using Native Go-RPC...
```


5. Leave this terminal window open. This asset node must run continuously in the background to serve file block data requests whenever authenticated clients navigate the dashboard.

```

```