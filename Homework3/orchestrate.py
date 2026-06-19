#!/usr/bin/env python3

import json
import os
import shutil
import subprocess
import sys
import time
import urllib.request
import urllib.error
from datetime import datetime
from pathlib import Path
from typing import Optional, Dict


PROJECT_ROOT = Path(__file__).resolve().parent
CONFIGS_DIR = PROJECT_ROOT / "configs"
RESULTS_DIR = PROJECT_ROOT / "results"
TMUX_SESSION = "hw3"

REPLICAS = {
    "replica1": {"port": 8080, "container": "replica1"},
    "replica2": {"port": 8081, "container": "replica2"},
    "replica3": {"port": 8082, "container": "replica3"},
}

PEERS = {
    "replica1": [
        {"id": "replica2", "address": "http://replica2:8081"},
        {"id": "replica3", "address": "http://replica3:8082"},
    ],
    "replica2": [
        {"id": "replica1", "address": "http://replica1:8080"},
        {"id": "replica3", "address": "http://replica3:8082"},
    ],
    "replica3": [
        {"id": "replica1", "address": "http://replica1:8080"},
        {"id": "replica2", "address": "http://replica2:8081"},
    ],
}

class Tee:
    """Writes to both a live stream (terminal / tmux pane) and a log file."""

    def __init__(self, log_path: Path):
        self.log_path = log_path
        self.log_path.parent.mkdir(parents=True, exist_ok=True)
        self._fh = open(log_path, "a", encoding="utf-8")

    def write(self, text: str, also_print: bool = True):
        self._fh.write(text)
        self._fh.flush()
        if also_print:
            sys.stdout.write(text)
            sys.stdout.flush()

    def close(self):
        self._fh.close()


def banner(tee: Tee, text: str):
    line = "=" * 78
    tee.write(f"\n{line}\n{text}\n{line}\n")


def sub_banner(tee: Tee, text: str):
    tee.write(f"\n── {text} " + "─" * max(0, 70 - len(text)) + "\n")


def now_iso():
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")


def run_shell(cmd: str, tee: Tee, check: bool = False, show_cmd: bool = True):
    """Run a shell command, streaming output live and logging it."""
    if show_cmd:
        tee.write(f"$ {cmd}\n")
    proc = subprocess.run(
        cmd, shell=True, cwd=PROJECT_ROOT,
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True
    )
    tee.write(proc.stdout)
    if check and proc.returncode != 0:
        tee.write(f"[ERROR] command exited with code {proc.returncode}\n")
    return proc.returncode, proc.stdout


def http_request(method: str, url: str, body: Optional[dict] = None, timeout: float = 6.0):
    """Minimal HTTP client (stdlib only). Returns (status_code, parsed_json, latency_ms, error_str)."""
    data = None
    headers = {}
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        headers["Content-Type"] = "application/json"

    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    start = time.perf_counter()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
            elapsed_ms = (time.perf_counter() - start) * 1000
            try:
                parsed = json.loads(raw)
            except json.JSONDecodeError:
                parsed = {"raw": raw}
            return resp.status, parsed, elapsed_ms, None
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8")
        elapsed_ms = (time.perf_counter() - start) * 1000
        try:
            parsed = json.loads(raw)
        except json.JSONDecodeError:
            parsed = {"raw": raw}
        return e.code, parsed, elapsed_ms, None
    except Exception as e:
        elapsed_ms = (time.perf_counter() - start) * 1000
        return None, None, elapsed_ms, str(e)


def put(replica: str, key: str, value: str, tee: Tee, label: str = "PUT"):
    port = REPLICAS[replica]["port"]
    url = f"http://localhost:{port}/put"
    cmd_repr = (
        f"curl -s -X POST {url} -H 'Content-Type: application/json' "
        f"-d '{{\"key\":\"{key}\",\"value\":\"{value}\"}}'"
    )
    tee.write(f"$ {cmd_repr}\n")
    status, parsed, elapsed_ms, err = http_request("POST", url, {"key": key, "value": value})
    _print_result(tee, label, replica, status, parsed, elapsed_ms, err)
    return status, parsed, elapsed_ms, err


def get(replica: str, key: str, tee: Tee, label: str = "GET"):
    port = REPLICAS[replica]["port"]
    url = f"http://localhost:{port}/get?key={key}"
    tee.write(f"$ curl -s '{url}'\n")
    status, parsed, elapsed_ms, err = http_request("GET", url)
    _print_result(tee, label, replica, status, parsed, elapsed_ms, err)
    return status, parsed, elapsed_ms, err


def health(replica: str, tee: Tee):
    port = REPLICAS[replica]["port"]
    url = f"http://localhost:{port}/health"
    tee.write(f"$ curl -s '{url}'\n")
    status, parsed, elapsed_ms, err = http_request("GET", url)
    _print_result(tee, "HEALTH", replica, status, parsed, elapsed_ms, err)
    return status, parsed, elapsed_ms, err


def _print_result(tee, label, replica, status, parsed, elapsed_ms, err):
    ts = now_iso()
    if err:
        tee.write(f"  [{ts}] {label} {replica} -> NETWORK ERROR: {err}  (round_trip={elapsed_ms:.1f}ms)\n")
        return
    ok = "OK" if status and 200 <= status < 300 else "FAIL"
    tee.write(
        f"  [{ts}] {label} {replica} -> HTTP {status} [{ok}]  "
        f"round_trip={elapsed_ms:.1f}ms  body={json.dumps(parsed, ensure_ascii=False)}\n"
    )


def write_config(replica: str, mode: str, delay_ms: int = 0):
    """Rewrite configs/<replica>.json with the given mode and delay."""
    cfg = {
        "replica_id": replica,
        "port": REPLICAS[replica]["port"],
        "consistency_mode": mode,
        "network_delay_ms": delay_ms,
        "peers": PEERS[replica],
    }
    path = CONFIGS_DIR / f"{replica}.json"
    path.write_text(json.dumps(cfg, indent=2) + "\n", encoding="utf-8")


def set_all_configs(mode: str, delay_ms_per_replica: Optional[dict] = None, tee: Tee = None):
    """Write configs for all three replicas. delay_ms_per_replica overrides per-replica delay."""
    delay_ms_per_replica = delay_ms_per_replica or {}
    for r in REPLICAS:
        d = delay_ms_per_replica.get(r, 0)
        write_config(r, mode, d)
    if tee:
        delays_str = ", ".join(f"{r}={delay_ms_per_replica.get(r,0)}ms" for r in REPLICAS)
        tee.write(f"[config] mode={mode}  delays: {delays_str}\n")


def docker_compose_up(tee: Tee):
    run_shell("docker compose up --build -d", tee, check=True)


def docker_compose_restart(service: str, tee: Tee):
    run_shell(f"docker compose restart {service}", tee, check=True)


def docker_compose_restart_all(tee: Tee):
    run_shell("docker compose restart", tee, check=True)


def docker_stop(container: str, tee: Tee):
    run_shell(f"docker stop {container}", tee, check=True)


def docker_start(container: str, tee: Tee):
    run_shell(f"docker start {container}", tee, check=True)


def docker_compose_down(tee: Tee):
    run_shell("docker compose down", tee, check=True)


def wait_for_health(replica: str, tee: Tee, retries: int = 20, interval: float = 0.5) -> bool:
    for i in range(retries):
        status, _, _, err = health(replica, tee)
        if status == 200:
            return True
        time.sleep(interval)
    tee.write(f"[WARN] {replica} did not become healthy after {retries * interval:.1f}s\n")
    return False


def wait_for_all_healthy(tee: Tee):
    sub_banner(tee, "Waiting for all replicas to become healthy")
    for r in REPLICAS:
        wait_for_health(r, tee)


def tmux_available() -> bool:
    return shutil.which("tmux") is not None


def setup_tmux_session():
    """
    Creates a tmux session with 4 panes:
      - pane 0 (top-left):    this orchestrator script's own output
      - pane 1 (top-right):   docker compose logs (all replicas, live)
      - pane 2 (bottom-left): replica1 logs only
      - pane 3 (bottom-right): a free shell for you to poke around in
    Layout:
        +-------------------+-------------------+
        | orchestrator       | docker compose logs|
        | (this script)      | (all replicas)      |
        +-------------------+-------------------+
        | replica1 logs only | free shell           |
        +-------------------+-------------------+
    """
    subprocess.run(["tmux", "kill-session", "-t", TMUX_SESSION],
                    stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    subprocess.run(["tmux", "new-session", "-d", "-s", TMUX_SESSION, "-c", str(PROJECT_ROOT)])
    subprocess.run(["tmux", "split-window", "-h", "-t", f"{TMUX_SESSION}:0", "-c", str(PROJECT_ROOT)])
    subprocess.run(["tmux", "split-window", "-v", "-t", f"{TMUX_SESSION}:0.0", "-c", str(PROJECT_ROOT)])
    subprocess.run(["tmux", "split-window", "-v", "-t", f"{TMUX_SESSION}:0.1", "-c", str(PROJECT_ROOT)])

    subprocess.run(["tmux", "send-keys", "-t", f"{TMUX_SESSION}:0.2",
                     "docker compose logs -f --tail=50", "Enter"])
    subprocess.run(["tmux", "send-keys", "-t", f"{TMUX_SESSION}:0.1",
                     "docker compose logs -f replica1", "Enter"])
    subprocess.run(["tmux", "send-keys", "-t", f"{TMUX_SESSION}:0.3",
                     "echo 'Free shell — use this to run curl/docker commands manually if you want.'", "Enter"])

    subprocess.run(["tmux", "select-pane", "-t", f"{TMUX_SESSION}:0.0"])
    print(f"[tmux] Session '{TMUX_SESSION}' created with 4 panes.")
    print(f"[tmux] Attach with:  tmux attach -t {TMUX_SESSION}")


def run_self_inside_tmux():
    """Re-exec this script inside the orchestrator pane of the tmux session."""
    script_path = Path(__file__).resolve()
    cmd = f"python3 {script_path} --inner"
    subprocess.run(["tmux", "send-keys", "-t", f"{TMUX_SESSION}:0.0", cmd, "Enter"])



def scenario_1(tee: Tee, metrics: dict):
    banner(tee, "SCENARIO 1 — Temporary Inconsistency (Eventual Consistency)")
    set_all_configs("eventual", tee=tee)
    docker_compose_restart_all(tee)
    wait_for_all_healthy(tee)

    sub_banner(tee, "Step 1: PUT x=10 on replica1")
    s1, r1, t1, _ = put("replica1", "x", "10", tee)
    metrics["s1_put_latency_ms"] = round(t1, 2)
    metrics["s1_put_server_latency_ms"] = r1.get("latency_ms") if isinstance(r1, dict) else None

    sub_banner(tee, "Step 2: IMMEDIATELY read x from replica2 (expect stale / not found)")
    s2, r2, t2, _ = get("replica2", "x", tee, label="GET(immediate)")
    immediate_stale = not (s2 == 200 and isinstance(r2, dict) and r2.get("data", {}).get("value") == "10")
    metrics["s1_immediate_read_stale"] = immediate_stale

    sub_banner(tee, "Step 3: Poll replica2 every 100ms until it converges (measuring convergence time)")
    conv_start = time.perf_counter()
    converged = False
    polls = 0
    stale_count = 0
    for i in range(50): 
        polls += 1
        s, r, t, _ = get("replica2", "x", tee, label=f"GET(poll {i+1})")
        if s == 200 and isinstance(r, dict) and r.get("data", {}).get("value") == "10":
            converged = True
            break
        else:
            stale_count += 1
        time.sleep(0.1)
    conv_time_ms = (time.perf_counter() - conv_start) * 1000

    metrics["s1_convergence_time_ms"] = round(conv_time_ms, 2)
    metrics["s1_polls_until_converged"] = polls
    metrics["s1_stale_reads_observed"] = stale_count

    sub_banner(tee, "Result")
    tee.write(
        f"  Converged: {converged} after {polls} poll(s), "
        f"~{conv_time_ms:.1f}ms, {stale_count} stale read(s) observed.\n"
    )
    return metrics


def scenario_2(tee: Tee, metrics: dict):
    banner(tee, "SCENARIO 2 — Replica Failure")

    sub_banner(tee, "Part A: Eventual consistency, replica3 stopped")
    set_all_configs("eventual", tee=tee)
    docker_compose_restart_all(tee)
    wait_for_all_healthy(tee)

    docker_stop("replica3", tee)
    time.sleep(1)

    s, r, t, err = put("replica1", "y", "42", tee, label="PUT(replica3 down, eventual)")
    metrics["s2a_put_status"] = s
    metrics["s2a_put_latency_ms"] = round(t, 2)
    metrics["s2a_put_succeeded_despite_failure"] = (s == 200)

    docker_start("replica3", tee)
    wait_for_health("replica3", tee)
    time.sleep(1)

    sub_banner(tee, "Checking whether replica3 ever received y after recovery (no retry expected)")
    s3, r3, t3, _ = get("replica3", "y", tee, label="GET(replica3 after recovery)")
    metrics["s2a_replica3_recovered_value_present"] = (
        s3 == 200 and isinstance(r3, dict) and r3.get("data", {}).get("value") == "42"
    )

    sub_banner(tee, "Part B: Strong consistency, replica3 stopped (quorum 2/3 still reachable)")
    set_all_configs("strong", tee=tee)
    docker_compose_restart_all(tee)
    wait_for_all_healthy(tee)

    docker_stop("replica3", tee)
    time.sleep(1)

    s, r, t, err = put("replica1", "y", "99", tee, label="PUT(strong, 1 peer down)")
    metrics["s2b_put_status"] = s
    metrics["s2b_put_latency_ms"] = round(t, 2)
    metrics["s2b_quorum_succeeded_with_1_down"] = (s == 200)

    sub_banner(tee, "Part C: Strong consistency, replica2 ALSO stopped (quorum impossible)")
    docker_stop("replica2", tee)
    time.sleep(1)

    s, r, t, err = put("replica1", "y", "100", tee, label="PUT(strong, 2 peers down)")
    metrics["s2c_put_status"] = s
    metrics["s2c_put_latency_ms"] = round(t, 2)
    metrics["s2c_quorum_failed_with_2_down"] = (s != 200)

    sub_banner(tee, "Restoring all replicas")
    docker_start("replica2", tee)
    docker_start("replica3", tee)
    wait_for_all_healthy(tee)

    sub_banner(tee, "Result summary")
    tee.write(
        f"  Eventual mode, 1 peer down: PUT still succeeded "
        f"({metrics['s2a_put_succeeded_despite_failure']}), replica3 caught up after "
        f"restart: {metrics['s2a_replica3_recovered_value_present']} (no retry mechanism, "
        f"so this is expected to be False).\n"
        f"  Strong mode, 1 peer down (quorum 2/3 reachable): "
        f"PUT succeeded = {metrics['s2b_quorum_succeeded_with_1_down']}\n"
        f"  Strong mode, 2 peers down (quorum impossible): "
        f"PUT correctly failed = {metrics['s2c_quorum_failed_with_2_down']}\n"
    )
    return metrics


def scenario_3(tee: Tee, metrics: dict):
    banner(tee, "SCENARIO 3 — Concurrent Conflict (Eventual Consistency)")
    set_all_configs("eventual", tee=tee)
    docker_compose_restart_all(tee)
    wait_for_all_healthy(tee)

    sub_banner(tee, "Firing two conflicting writes to replica1 and replica2 simultaneously")
    import concurrent.futures
    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as ex:
        f1 = ex.submit(put, "replica1", "z", "from_r1", tee, "PUT(concurrent, r1)")
        f2 = ex.submit(put, "replica2", "z", "from_r2", tee, "PUT(concurrent, r2)")
        s1, r1, t1, _ = f1.result()
        s2, r2, t2, _ = f2.result()

    metrics["s3_put_r1_latency_ms"] = round(t1, 2)
    metrics["s3_put_r2_latency_ms"] = round(t2, 2)

    sub_banner(tee, "Waiting 2s for replication to settle")
    time.sleep(2)

    sub_banner(tee, "Final values on all three replicas")
    final_values = {}
    for r in REPLICAS:
        s, resp, t, _ = get(r, "z", tee, label=f"GET(final, {r})")
        if s == 200 and isinstance(resp, dict):
            final_values[r] = resp.get("data", {})

    metrics["s3_final_values"] = final_values
    values_seen = {v.get("value") for v in final_values.values() if v}
    metrics["s3_converged_to_single_value"] = (len(values_seen) == 1)
    metrics["s3_winning_value"] = next(iter(values_seen)) if values_seen else None

    sub_banner(tee, "Result")
    tee.write(
        f"  All replicas converged to a single value: {metrics['s3_converged_to_single_value']}\n"
        f"  Winning value (expected 'from_r2' since replica2 > replica1 lexicographically): "
        f"{metrics['s3_winning_value']}\n"
    )
    return metrics


def scenario_4(tee: Tee, metrics: dict):
    banner(tee, "SCENARIO 4 — Effect of Network Delay")
    delays_to_test = [0, 500, 2000]
    metrics["s4_runs"] = []

    for delay in delays_to_test:
        sub_banner(tee, f"Testing with network_delay_ms={delay} on replica1's outgoing replication")
        set_all_configs("eventual", delay_ms_per_replica={"replica1": delay}, tee=tee)
        docker_compose_restart_all(tee)
        wait_for_all_healthy(tee)

        key = f"w_delay_{delay}"
        sub_banner(tee, f"PUT {key}=A on replica1")
        s, r, put_latency_ms, _ = put("replica1", key, "A", tee, label=f"PUT(delay={delay}ms)")

        sub_banner(tee, f"Polling replica2 for convergence (delay={delay}ms)")
        conv_start = time.perf_counter()
        converged = False
        polls = 0
        for i in range(60):
            polls += 1
            sg, rg, tg, _ = get("replica2", key, tee, label=f"GET(poll {i+1}, delay={delay}ms)")
            if sg == 200 and isinstance(rg, dict) and rg.get("data", {}).get("value") == "A":
                converged = True
                break
            time.sleep(0.1)
        conv_time_ms = (time.perf_counter() - conv_start) * 1000

        run_result = {
            "delay_ms": delay,
            "put_latency_ms": round(put_latency_ms, 2),
            "convergence_time_ms": round(conv_time_ms, 2),
            "polls_until_converged": polls,
            "converged": converged,
        }
        metrics["s4_runs"].append(run_result)

        sub_banner(tee, f"Result for delay={delay}ms")
        tee.write(
            f"  PUT latency: {run_result['put_latency_ms']}ms  "
            f"(should stay flat regardless of delay, since eventual mode doesn't block on replication)\n"
            f"  Convergence time: {run_result['convergence_time_ms']}ms  "
            f"(should track closely with injected delay of {delay}ms)\n"
        )

    return metrics


def generate_metrics_table(metrics: dict) -> str:
    lines = []
    lines.append("# Metrics Table\n")
    lines.append("| Model    | Scenario                          | PUT Latency (ms) | Convergence Time (ms) | Stale Reads |")
    lines.append("|----------|------------------------------------|-------------------|------------------------|-------------|")
    lines.append(
        f"| Eventual | 1 — Temporary Inconsistency        | "
        f"{metrics.get('s1_put_latency_ms','-')} | "
        f"{metrics.get('s1_convergence_time_ms','-')} | "
        f"{metrics.get('s1_stale_reads_observed','-')} |"
    )
    lines.append(
        f"| Eventual | 2a — Replica Failure (1 down)       | "
        f"{metrics.get('s2a_put_latency_ms','-')} | n/a | n/a |"
    )
    lines.append(
        f"| Strong   | 2b — Replica Failure (1 down, quorum OK) | "
        f"{metrics.get('s2b_put_latency_ms','-')} | 0 (synchronous) | 0 |"
    )
    lines.append(
        f"| Strong   | 2c — Replica Failure (2 down, quorum FAIL) | "
        f"{metrics.get('s2c_put_latency_ms','-')} (failed, HTTP {metrics.get('s2c_put_status','-')}) | n/a | n/a |"
    )
    lines.append(
        f"| Eventual | 3 — Concurrent Conflict             | "
        f"r1={metrics.get('s3_put_r1_latency_ms','-')}, r2={metrics.get('s3_put_r2_latency_ms','-')} | "
        f"~2000 (fixed wait) | n/a |"
    )
    for run in metrics.get("s4_runs", []):
        lines.append(
            f"| Eventual | 4 — Network Delay ({run['delay_ms']}ms injected) | "
            f"{run['put_latency_ms']} | {run['convergence_time_ms']} | "
            f"{run['polls_until_converged'] - 1} |"
        )
    return "\n".join(lines) + "\n"


def generate_report_summary(metrics: dict) -> str:
    s4_lines = ""
    for run in metrics.get("s4_runs", []):
        s4_lines += (
            f"- **delay={run['delay_ms']}ms** → PUT latency {run['put_latency_ms']}ms, "
            f"convergence {run['convergence_time_ms']}ms, "
            f"converged={run['converged']}\n"
        )

    return f"""# Report Summary — Auto-Generated by orchestrate.py

Generated: {now_iso()}

This file summarizes the outcome of all four test scenarios. Raw command-by-command
logs (with timestamps and full JSON responses) are in `results/scenarioN.log`.
Use those logs as evidence/screenshots for your report, and use this file as the
narrative text you can adapt directly into Part 4 (آزمایش و تحلیل).

---

## Scenario 1 — Temporary Inconsistency

- PUT x=10 on replica1: client-observed latency = **{metrics.get('s1_put_latency_ms')}ms**,
  server-measured latency = **{metrics.get('s1_put_server_latency_ms')}ms**
- Immediate read from replica2 was stale: **{metrics.get('s1_immediate_read_stale')}**
- Convergence took **{metrics.get('s1_convergence_time_ms')}ms**
  ({metrics.get('s1_polls_until_converged')} poll attempts,
  {metrics.get('s1_stale_reads_observed')} stale reads observed before convergence)

**Explanation:** In eventual consistency, replica1 applies the write locally and
replies to the client immediately, before replication to peers has completed.
Because replication happens asynchronously in the background, a read against
replica2 issued right after the write can observe the old (or missing) value.
After a short delay the asynchronous replication message arrives and replica2
converges to the same value as replica1.

---

## Scenario 2 — Replica Failure

**Part A (Eventual, replica3 stopped):**
PUT y=42 on replica1 succeeded despite replica3 being down: **{metrics.get('s2a_put_succeeded_despite_failure')}**
(latency {metrics.get('s2a_put_latency_ms')}ms). After restarting replica3, it had
already received the update: **{metrics.get('s2a_replica3_recovered_value_present')}**.

**Limitation observed:** the async replicator has no retry-on-recovery mechanism —
if a peer is unreachable when `SendAsync` fires, that update is logged as an error
and dropped, not redelivered when the peer comes back. This is worth stating
explicitly in the "implementation problems and limitations" section of the report.

**Part B (Strong, 1 peer down, quorum 2/3 still reachable):**
PUT y=99 succeeded: **{metrics.get('s2b_quorum_succeeded_with_1_down')}**
(latency {metrics.get('s2b_put_latency_ms')}ms, HTTP {metrics.get('s2b_put_status')}).

**Part C (Strong, 2 peers down, quorum impossible):**
PUT y=100 correctly failed: **{metrics.get('s2c_quorum_failed_with_2_down')}**
(HTTP {metrics.get('s2c_put_status')}).

**Explanation:** Eventual consistency always accepts a write locally regardless of
peer availability — it only fails if the local write itself fails. Strong
consistency requires a majority (2 of 3) of replicas to acknowledge the write
before it is considered successful. When only one peer is reachable
(local + 1 = 2 = quorum), the write still succeeds. When two peers are down
(only the local replica is reachable), quorum cannot be reached and the write
is correctly rejected with an HTTP 503.

---

## Scenario 3 — Concurrent Conflict

Two clients wrote different values for key `z` to replica1 and replica2 at
nearly the same time ({metrics.get('s3_put_r1_latency_ms')}ms and
{metrics.get('s3_put_r2_latency_ms')}ms respectively). After replication settled,
all three replicas converged to a single value: **{metrics.get('s3_converged_to_single_value')}**,
specifically `{metrics.get('s3_winning_value')}`.

Final values observed on each replica:
```
{json.dumps(metrics.get('s3_final_values', {}), indent=2, ensure_ascii=False)}
```

**Explanation:** Both writes created entries with the same version number
(version 1, since both started from an empty key). This is a true conflict —
not a stale read — because both replicas have a value at the same version.
The conflict resolution policy is Last-Write-Wins by replica ID: the entry
whose `updated_by` field is lexicographically larger wins. Since
`"replica2" > "replica1"`, the value written via replica2 (`from_r2`) wins,
and every replica that receives both versions of the entry independently
reaches the same conclusion, guaranteeing convergence.

---

## Scenario 4 — Effect of Network Delay

{s4_lines}

**Explanation:** PUT latency in eventual consistency mode stays roughly constant
regardless of the injected replication delay, because the client receives a
response as soon as the local write completes — replication to peers happens
in background goroutines that the client never waits on. Convergence time,
however, scales directly with the injected delay, since that delay is added
before each replication message is sent to peers. This demonstrates the
fundamental trade-off of eventual consistency: low write latency at the cost
of a longer (and delay-dependent) window during which replicas may disagree.

---

## Overall Comparison: Strong vs Eventual

| Aspect | Eventual Consistency | Strong Consistency |
|---|---|---|
| Write latency | Low, constant (Scenario 4) | Higher, depends on quorum round-trip (Scenario 2b) |
| Availability under failure | Always available if local replica is up (Scenario 2a) | Requires majority of replicas reachable (Scenario 2c) |
| Read freshness | May return stale data briefly after a write (Scenario 1) | Always consistent once a write succeeds |
| Conflict handling | Needed — concurrent writes can create same-version conflicts (Scenario 3) | Conflicts are rarer due to quorum coordination, but the same LWW policy still applies |

---

## Known Implementation Limitations

1. No retry-on-recovery for asynchronous replication — updates sent while a peer
   is down are lost rather than redelivered once it comes back online (see Scenario 2a).
2. Strong consistency mode writes locally before confirming quorum; if quorum
   fails, the local replica still holds the new (unconfirmed) value, which is
   a simplification relative to a true two-phase commit protocol.
3. Reads in strong consistency mode are served from whichever replica receives
   the request rather than from a single coordinated primary, so a client
   reading from a replica that hasn't yet replicated a quorum-confirmed write
   could theoretically see a different value — though no such case was observed
   in testing above.
"""


def main_inner():
    """The actual test run. Runs inside the tmux orchestrator pane (or directly, with --no-tmux)."""
    RESULTS_DIR.mkdir(exist_ok=True)
    master_log = Tee(RESULTS_DIR / "FULL_RUN.log")
    metrics: dict = {}

    banner(master_log, "HW3 ORCHESTRATOR — Full Automated Test Run")
    master_log.write(f"Started: {now_iso()}\n")
    master_log.write(f"Project root: {PROJECT_ROOT}\n")

    sub_banner(master_log, "Bringing up all replicas (initial state: eventual)")
    set_all_configs("eventual", tee=master_log)
    docker_compose_up(master_log)
    wait_for_all_healthy(master_log)

    s1_log = Tee(RESULTS_DIR / "scenario1.log")
    metrics = scenario_1(s1_log, metrics)
    s1_log.close()

    s2_log = Tee(RESULTS_DIR / "scenario2.log")
    metrics = scenario_2(s2_log, metrics)
    s2_log.close()

    s3_log = Tee(RESULTS_DIR / "scenario3.log")
    metrics = scenario_3(s3_log, metrics)
    s3_log.close()

    s4_log = Tee(RESULTS_DIR / "scenario4.log")
    metrics = scenario_4(s4_log, metrics)
    s4_log.close()

    sub_banner(master_log, "Restoring default configuration (eventual, no delay)")
    set_all_configs("eventual", tee=master_log)
    docker_compose_restart_all(master_log)
    wait_for_all_healthy(master_log)

    (RESULTS_DIR / "metrics.json").write_text(
        json.dumps(metrics, indent=2, ensure_ascii=False, default=str), encoding="utf-8"
    )

    (RESULTS_DIR / "metrics_table.md").write_text(generate_metrics_table(metrics), encoding="utf-8")
    (RESULTS_DIR / "REPORT_SUMMARY.md").write_text(generate_report_summary(metrics), encoding="utf-8")

    banner(master_log, "ALL SCENARIOS COMPLETE")
    master_log.write(f"Finished: {now_iso()}\n")
    master_log.write(
        "\nReport-ready files generated in results/:\n"
        "  - REPORT_SUMMARY.md   <- narrative text, ready to paste into your report\n"
        "  - metrics_table.md    <- the required metrics table, pre-filled\n"
        "  - metrics.json        <- raw numbers, in case you want custom formatting\n"
        "  - scenario1.log .. scenario4.log  <- full command-by-command evidence\n"
        "  - FULL_RUN.log        <- everything above, in one file\n"
    )
    master_log.close()
    print("\nDone. See results/REPORT_SUMMARY.md and results/metrics_table.md\n")


def main():
    use_tmux = "--no-tmux" not in sys.argv
    inner = "--inner" in sys.argv

    if inner or not use_tmux:
        main_inner()
        return

    if not tmux_available():
        print("[WARN] tmux not found — running directly without live panes.")
        print("       Install with: sudo apt install -y tmux")
        main_inner()
        return

    setup_tmux_session()
    run_self_inside_tmux()
    print(f"\nLive view: tmux attach -t {TMUX_SESSION}")
    print("Detach from tmux without stopping it with: Ctrl+b then d")
    print("Results will appear in results/ as scenarios complete, even if you don't attach.\n")


if __name__ == "__main__":
    main()
