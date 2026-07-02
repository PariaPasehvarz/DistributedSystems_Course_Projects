# Replica

Each replica is an independent HTTP server that maintains its own local key-value store.

## How to run (standalone, without Docker)

```bash
go build -o replica .
./replica -config ../configs/replica1.json
```

## Endpoints

| Method | Path                   | Description                          |
|--------|------------------------|--------------------------------------|
| POST   | /put                   | Client write (JSON body: key, value) |
| GET    | /get?key=<key>         | Client read                          |
| POST   | /internal/replicate    | Peer-to-peer replication (internal)  |
| GET    | /health                | Health check                         |

## Config fields

| Field              | Type   | Description                                         |
|--------------------|--------|-----------------------------------------------------|
| replica_id         | string | Unique name for this replica                        |
| port               | int    | Port to listen on                                   |
| consistency_mode   | string | "eventual" or "strong"                              |
| peers              | array  | List of peer replicas (id + address)                |
| network_delay_ms   | int    | Artificial replication delay in ms (0 = none)       |
