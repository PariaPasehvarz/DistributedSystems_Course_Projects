# Authentication Backend Service (VM2)

## Overview
This directory contains the independent, standalone Authentication Service running on **VM2**. To fulfill strict architectural decoupling principles and prevent credential exposure, the primary web server (VM1) has no direct access to user file records or login validation databases. Instead, all credential verifications must pass securely through this decoupled layer via formal network socket communication links.

---

## Architectural Specifications

* **RPC Interface Exposure:** Exposes a secure, network-accessible object listener interface using the standard `JSON-RPC` protocol over port 8082. It exposes dedicated procedures such as `Login(args, reply)` to evaluate incoming authorization attempts from the web server.
* **Isolated Data Persistence Store:** Reads account information out of a protected local storage mapping file (`users.json`). This ensures that user credentials remain completely isolated on VM2 and are never transmitted to, or cached persistently on, the web server node.

---

## Project Directory Layout
```text
auth-vm/
├── main.go               # Network server listener engine and RPC operational methods
└── users.json            # Protected JSON storage containing valid testing accounts and credentials

```

---

## How to Run

### Prerequisites

1. Ensure the Go SDK is installed and configured on the VM2 operating system environment.
2. Confirm that port 8082 is open and not blocked by local firewall configurations to allow incoming TCP connection requests from VM1.

### Execution Steps

1. Open a terminal session on VM2.
2. Navigate directly to your workspace repository:
```bash
cd ~/auth-vm

```


3. Boot up the service execution process using the Go toolchain:
```bash
go run main.go

```


4. Verify from the terminal stdout logs that the service is running and listening for RPC calls:
```text
2026/06/01 17:43:14 Secure JSON-RPC Auth Server active on port 8082 (File-Backed)...

```


5. Leave this terminal window open and running. The service must remain active continuously so that the web server on VM1 can complete remote authorization handshakes when users attempt to access the application.

```

```