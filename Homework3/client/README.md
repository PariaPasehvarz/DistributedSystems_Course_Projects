# Client

An interactive CLI client for the replicated key-value store.

## How to run

```bash
go build -o client .
./client -addr http://localhost:8080
```

## Commands

| Command              | Description                                      |
|----------------------|--------------------------------------------------|
| PUT <key> <value>    | Write a value to the connected replica           |
| GET <key>            | Read a value from the connected replica          |
| HEALTH               | Check replica health                             |
| SWITCH <addr>        | Switch to a different replica                    |
| HELP                 | Show help                                        |
| EXIT                 | Quit                                             |

## Example session

```
[http://localhost:8080] > PUT x 10
  PUT  key="x"  value="10"  version=1  by=replica1  round_trip=3ms

[http://localhost:8080] > SWITCH http://localhost:8081
  Switched to http://localhost:8081

[http://localhost:8081] > GET x
  GET  key="x"  value="10"  version=1  by=replica1  round_trip=1ms
```
