# Metrics Table

| Model    | Scenario                          | PUT Latency (ms) | Convergence Time (ms) | Stale Reads |
|----------|------------------------------------|-------------------|------------------------|-------------|
| Eventual | 1 — Temporary Inconsistency        | 3.17 | 5.58 | 0 |
| Eventual | 2a — Replica Failure (1 down)       | 2.78 | n/a | n/a |
| Strong   | 2b — Replica Failure (1 down, quorum OK) | 9.56 | 0 (synchronous) | 0 |
| Strong   | 2c — Replica Failure (2 down, quorum FAIL) | 5003.49 (failed, HTTP 503) | n/a | n/a |
| Eventual | 3 — Concurrent Conflict             | r1=2.7, r2=3.35 | ~2000 (fixed wait) | n/a |
| Eventual | 4 — Network Delay (0ms injected) | 1.8 | 4.01 | 0 |
| Eventual | 4 — Network Delay (500ms injected) | 1.5 | 551.38 | 5 |
| Eventual | 4 — Network Delay (2000ms injected) | 1.35 | 2044.25 | 19 |
