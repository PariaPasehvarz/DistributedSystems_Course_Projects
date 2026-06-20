# HW3 — Manual Run Instructions

Ports: replica1=8080, replica2=8081, replica3=8082

Each replica reads `configs/replicaN.json`:
```json
{
  "replica_id": "replica1",
  "port": 8080,
  "consistency_mode": "eventual",
  "network_delay_ms": 0,
  "peers": [...]
}
```
Edit this file (`consistency_mode`: `eventual` or `strong`, `network_delay_ms`: int) before each scenario, then restart the replicas for the change to take effect.

Bring everything up:
```bash
docker compose up --build -d
```

Check health:
```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8081/health
curl -s http://localhost:8082/health
```

## Starting the client

Instead of `curl`, you can use `client/main.go` — an interactive CLI connected to one replica:
```bash
go run client/main.go -addr http://localhost:8080
```
Once connected:
```
PUT <key> <value>     write
GET <key>             read
HEALTH                check health
SWITCH <addr>          point this same client at a different replica
EXIT                   quit
```
If a scenario needs to talk to two replicas at once, either open two terminals (one client each, pointed at different `-addr`), or use `SWITCH` to redirect a single client between commands.

---

## Scenario 1 — Temporary Inconsistency (Eventual)

1. Set all three configs to `"consistency_mode": "eventual"`, `"network_delay_ms": 0`, then:
   ```bash
   docker compose restart
   ```
2. PUT on replica1:
   - curl:
     ```bash
     curl -s -X POST http://localhost:8080/put -H 'Content-Type: application/json' -d '{"key":"x","value":"10"}'
     ```
   - client: connect with `-addr http://localhost:8080`, then run `PUT x 10`
3. Immediately GET from replica2:
   - curl:
     ```bash
     curl -s 'http://localhost:8081/get?key=x'
     ```
   - client: connect (or `SWITCH http://localhost:8081`), then run `GET x`

   
4. if the new value is not seen, repeat the GET on replica2 until the value becomes `10`. Note how long it took (convergence time) and how many stale reads you got before that.

---

## Scenario 2 — Replica Failure

**Part A — Eventual, replica3 down**

1. Set all configs to `eventual`, `network_delay_ms: 0`, then `docker compose restart`.
2. Stop replica3:
   ```bash
   docker stop replica3
   ```
3. PUT on replica1:
   - curl:
     ```bash
     curl -s -X POST http://localhost:8080/put -H 'Content-Type: application/json' -d '{"key":"y","value":"42"}'
     ```
   - client: `-addr http://localhost:8080`, then `PUT y 42`

   it should succeed (HTTP 200 / `PUT`) even though replica3 is down.
4. Start replica3 back up:
   ```bash
   docker start replica3
   ```
5. Once healthy, GET from replica3:
   - curl:
     ```bash
     curl -s 'http://localhost:8082/get?key=y'
     ```
   - client: `SWITCH http://localhost:8082`, then `GET y`


**Part B — Strong, 1 replica down (quorum still reachable)**

1. Set all configs to `"consistency_mode": "strong"`, then `docker compose restart`.
2. Stop replica3:
   ```bash
   docker stop replica3
   ```
3. PUT on replica1:
   - curl:
     ```bash
     curl -s -X POST http://localhost:8080/put -H 'Content-Type: application/json' -d '{"key":"y","value":"99"}'
     ```
   - client: `-addr http://localhost:8080`, then `PUT y 99`

   it should succeed (replica1 + replica2 = 2/3 = quorum).

**Part C — Strong, 2 replicas down (quorum impossible)**

1. Also stop replica2:
   ```bash
   docker stop replica2
   ```
2. PUT on replica1:
   - curl:
     ```bash
     curl -s -X POST http://localhost:8080/put -H 'Content-Type: application/json' -d '{"key":"y","value":"100"}'
     ```
   - client: (same connection as above) `PUT y 100`

   it should fail (HTTP 503 / `PUT failed` — only 1/3 reachable, no quorum).
3. Restore everything:
   ```bash
   docker start replica2
   docker start replica3
   ```

**Compare:** eventual mode accepted the write with a replica down; strong mode accepted it with 1 down (quorum met) but rejected it with 2 down (quorum lost).

---

## Scenario 3 — Concurrent Conflict (Eventual)

1. Set all configs to `eventual`, `network_delay_ms: 0`, then `docker compose restart`.
2. Fire these two PUTs at (as close to) the same time as possible, to different replicas, same key, different values — open two terminals and run both within the same second:
   - curl:
     ```bash
     curl -s -X POST http://localhost:8080/put -H 'Content-Type: application/json' -d '{"key":"z","value":"from_r1"}'
     ```
     ```bash
     curl -s -X POST http://localhost:8081/put -H 'Content-Type: application/json' -d '{"key":"z","value":"from_r2"}'
     ```
   - client: terminal 1 → `go run client/main.go -addr http://localhost:8080` then `PUT z from_r1`; terminal 2 → `go run client/main.go -addr http://localhost:8081` then `PUT z from_r2`
3. Wait ~2 seconds for replication to settle.
4. GET `z` from all three replicas:
   - curl:
     ```bash
     curl -s 'http://localhost:8080/get?key=z'
     curl -s 'http://localhost:8081/get?key=z'
     curl -s 'http://localhost:8082/get?key=z'
     ```
   - client: one client, `SWITCH http://localhost:8080` → `GET z`, `SWITCH http://localhost:8081` → `GET z`, `SWITCH http://localhost:8082` → `GET z`
5. they should all agree on one final value. With the implemented conflict policy (Last-Write-Wins by replica id), `from_r2` should win since `"replica2" > "replica1"`.

---

## Scenario 4 — Network Delay

Repeat the following for `delay = 0`, then `500`, then `2000` (ms):

1. Set `network_delay_ms` to the current delay value only in replica1's config (the others stay at 0), keep `consistency_mode: eventual`, then:
   ```bash
   docker compose restart
   ```
2. PUT a unique key on replica1 (e.g. `w_delay_0`, `w_delay_500`, `w_delay_2000`):
   - curl:
     ```bash
     curl -s -X POST http://localhost:8080/put -H 'Content-Type: application/json' -d '{"key":"w_delay_0","value":"A"}'
     ```
   - client: `-addr http://localhost:8080`, then `PUT w_delay_0 A`

   the round-trip time printed (PUT latency).
3. Poll replica2 every fast until the value appears:
   - curl:
     ```bash
     curl -s 'http://localhost:8081/get?key=w_delay_0'
     ```
   - client: `SWITCH http://localhost:8081`, then repeat `GET w_delay_0`

   Note the total time until it converges.
repeat for delay values

**Compare across the three runs:** PUT latency should stay roughly flat regardless of delay (the client doesn't wait on replication). Convergence time should scale up roughly in line with the injected delay.

---

## After all scenarios

Reset to default config (eventual, no delay on any replica) and restart:
```bash
docker compose restart
```

## Metrics to record per scenario (for the report table)

- PUT latency (ms)
- GET latency (ms)
- Convergence time (ms)