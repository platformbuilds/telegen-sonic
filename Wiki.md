# Telegen‑Sonic eBPF Data‑Plane Telemetry Agent (Wiki)

> **Name:** `telegen-sonic`  
> **Purpose:** On‑demand, per‑port **data‑plane telemetry** for SONiC, exporting **OpenTelemetry** metrics (OTLP) and exposing a **REST API + CLI**.  
> **Concurrency limit:** **Max 2 concurrent jobs** (hard‑enforced to protect switch resources).

---

## 1) Overview

`telegen-sonic` enables network operators to start **ephemeral capture/telemetry jobs** for specific switch ports (and optional filters) **on demand**.  
Because switch ASICs forward in hardware, the agent leverages **SPAN/ERSPAN mirroring or trap/punt** to feed copies of data‑plane packets to the CPU, where **eBPF programs** (attached via `tc clsact`) compute counters, flow stats, and latency distributions. A user‑space daemon aggregates results, exports **OTLP** to an OpenTelemetry Collector, and serves **instant job summaries** via REST/CLI.

**Key features**
- On‑demand jobs: *“monitor Ethernet16 for 120s with DSCP=46”*
- **Hard concurrency cap: 2** jobs at a time (configurable but default hard limit = 2)
- eBPF/CO‑RE programs, nanosecond‑precision histograms
- OTLP export (metrics; optional basic spans/events)
- REST API + CLI wrapper
- Automatic setup/teardown of mirror sessions and `tc` attachments
- Safe by default: sampling, top‑K limits, timeouts

---

## 2) Architecture

**Control plane**
- **REST server** (default `127.0.0.1:8080`, optional Unix socket)
- **Job supervisor** (state: *starting → running → stopping → done/failed*)
- **Concurrency gate**: at most **2** `running|starting` jobs

**Data plane**
- **Mirror provider**: SPAN to CPU *or* ERSPAN→local GRE decap (`erspan0`)
- **eBPF programs**: attached at `tc clsact` to CPU‑visible interface receiving mirrored traffic
- **User‑space collector**: ringbuf reader → aggregation → OTLP exporter + in‑memory result store

**Security**
- Runs as container with minimal caps: `CAP_BPF`, `CAP_NET_ADMIN`
- mTLS for REST (or Unix socket), optional role mapping to SONiC AAA

---

## 3) Supported telemetry

**Metrics (OpenTelemetry)**
- `network.packets` (Sum, monotonic) — attrs: `{device, port, direction, vlan?, ip_protocol?}`
- `network.bytes` (Sum, monotonic) — same attrs
- `network.errors` (Sum) — `{error.type}` e.g., `l3_checksum`, `tcp_retrans`
- `flow.packets`, `flow.bytes` (Sum) — `{src, dst, l4_sport, l4_dport, protocol, vlan?}` (top‑K limited)
- `network.latency` (Histogram, **nanoseconds**) — per‑packet/flow deltas from ingress/egress timing
- `tcp.rtt` (Histogram or Gauge) — when inferred from TCP info (if enabled)

**Resource controls**
- **Sampling** (e.g., 1% = process 1 in 100 packets)
- **Top‑K** flow table per interface (LRU with aging)
- **Duration** per job (hard timeout)
- **Concurrency cap = 2**

---

## 4) REST API

Base: `http://127.0.0.1:8080/v1` (default)

### Start a job
`POST /monitor/jobs`

Request:
```json
{
  "port": "Ethernet16",
  "direction": "ingress",
  "span_method": "span",       // "span" | "erspan"
  "vlan": 200,                 // optional
  "filters": {
    "ip_proto": ["tcp", "udp"],
    "l4_sport": [80, 443],
    "l4_dport": [],
    "src_cidr": "10.10.0.0/16",
    "dst_cidr": null,
    "dscp": [46]
  },
  "sample_rate": 100,          // 1=every packet, 100=1%
  "duration_sec": 120,         // auto stop after N seconds
  "otlp_export": true,
  "result_detail": "summary"   // "summary" | "flows" | "pcaplike"
}
```

Responses:
- **201 Created**
```json
{ "job_id": "UUID", "status": "starting", "interface": "mirror0" }
```
- **429 Too Many Requests** *(concurrency cap reached)*
```json
{ "error": "concurrency_limit", "message": "At most 2 concurrent jobs are allowed. Try again later." }
```
- **400 Bad Request** (invalid port/filters), **500** (internal error)

### Job status
`GET /monitor/jobs/{job_id}`
```json
{
  "job_id": "UUID",
  "status": "running",
  "started_at": "2025-08-15T17:10:32Z",
  "port": "Ethernet16",
  "interface": "mirror0",
  "expires_at": "2025-08-15T17:12:32Z"
}
```

### Job results
`GET /monitor/jobs/{job_id}/results?format=json`
```json
{
  "window_sec": 120,
  "packets_total": 184321,
  "bytes_total": 158234122,
  "errors": { "l2_crc": 0, "l3_checksum": 2, "tcp_retrans": 33 },
  "top_flows": [
    { "5tuple": "10.10.1.12:443->10.30.4.9:53214/TCP", "pkts": 9821, "bytes": 11231232 },
    { "5tuple": "10.10.1.13:80->10.30.4.8:51102/TCP", "pkts": 8812, "bytes": 8912312 }
  ],
  "latency_histogram_ns": {
    "bounds": [10000, 50000, 100000, 500000, 1000000],
    "counts": [1221, 8112, 19022, 2221, 182]
  },
  "otel_export": { "exported": true, "endpoint": "collector:4317" }
}
```

### Stop a job
`DELETE /monitor/jobs/{job_id}`
```json
{ "job_id": "UUID", "status": "stopped" }
```

---

## 5) CLI

Wrapper around REST (`telegen-sonic`):

```bash
# start
telegen-sonic start --port Ethernet16 --dir ingress \
  --span-method span --vlan 200 --sample-rate 100 \
  --filter ip_proto=tcp,udp --filter dscp=46 --duration 120

# status
telegen-sonic status --job UUID

# results
telegen-sonic results --job UUID --format json

# stop
telegen-sonic stop --job UUID
```

On concurrency cap hit:
```
ERROR: At most 2 concurrent jobs are allowed. Try again later. (HTTP 429)
```

---

## 6) Concurrency control (design)

- **Global gate**: count of `starting|running` jobs ≤ **2**.
- **Admission**: `POST /monitor/jobs` returns **429** if limit reached.
- **Fairness**: optional FIFO queue with TTL (disabled by default).
- **Auto‑stop**: hard timeout per job; agent force‑tears down mirror + tc.
- **Back‑pressure to OTLP**: export on a fixed cadence with bounded batch size.

**Go sketch:**
```go
var (
    maxConcurrent = 2
    activeJobs int32 // atomic
)

func tryStartJob(spec JobSpec) (Job, error) {
    if atomic.LoadInt32(&activeJobs) >= int32(maxConcurrent) {
        return Job{}, ErrConcurrencyLimit
    }
    // reserve a slot
    if !reserveSlot() { return Job{}, ErrConcurrencyLimit }
    j := newJob(spec)
    go func() {
        defer releaseSlot()
        j.run() // sets up mirror, attaches tc, collects, exports, teardown
    }()
    return j, nil
}

func reserveSlot() bool {
    for {
        n := atomic.LoadInt32(&activeJobs)
        if n >= int32(maxConcurrent) { return false }
        if atomic.CompareAndSwapInt32(&activeJobs, n, n+1) { return true }
    }
}
func releaseSlot() { atomic.AddInt32(&activeJobs, -1) }
```

---

## 7) Installation on SONiC

**Container run (example):**
```bash
docker run --name telegen-sonic --restart unless-stopped \
  --privileged \ 
  -v /sys:/sys -v /proc:/proc -v /sys/fs/bpf:/sys/fs/bpf \
  -p 127.0.0.1:8080:8080 \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=http://collector:4317 \
  ghcr.io/yourorg/telegen-sonic:latest
```
> Prefer minimal caps (`--cap-add=NET_ADMIN --cap-add=BPF`) when possible and mount `bpffs`.

**Requirements**
- SONiC with Linux kernel 5.10 and BTF enabled (`/sys/kernel/btf/vmlinux` present)
- `bpffs` mounted: `mount -t bpf bpf /sys/fs/bpf`
- SPAN/ERSPAN support on your platform (for data‑plane visibility)

---

## 8) Configuration & defaults

- `max_concurrent_jobs = 2`
- `default_duration_sec = 120`
- `default_sample_rate = 100` (1%)
- `export_interval_sec = 10`
- `topk_flows = 1024` per job
- `max_jobs_queue = 0` (queue disabled by default)

Config file (optional):
```yaml
server:
  listen: "127.0.0.1:8080"
limits:
  max_concurrent_jobs: 2
  default_duration_sec: 120
  default_sample_rate: 100
  topk_flows: 1024
export:
  otlp_endpoint: "http://collector:4317"
  interval_sec: 10
security:
  auth: "mtls"   # "mtls" | "unix"
```

---

## 9) Troubleshooting

- **429 on job start:** Two jobs already active. Wait for one to finish or stop one.
- **No packets counted:** Verify SPAN/ERSPAN session to CPU and eBPF is attached to the correct interface; check sampling.
- **High CPU:** Increase sampling rate (e.g., 500 = 0.2%), reduce `topk_flows`, shorten duration.
- **No OTLP export:** Validate Collector reachability and port (4317 gRPC or 4318 HTTP).

---

## 10) FAQ

**Q: Can I change the limit of 2 jobs?**  
A: Yes via config, but **2 is the recommended hard limit** for most fixed‑CPU switches.

**Q: Can I store PCAPs?**  
A: Not by default (to avoid I/O overhead). Use `result_detail=pcaplike` only for short windows and small samples.

**Q: Does this see hardware‑switched traffic?**  
A: Yes, **only if mirrored/punted** to CPU. Otherwise hardware forwarding bypasses the host stack.

---

## 11) Roadmap

- Optional queueing w/ priorities
- gNMI/REST hooks to program mirror/trap policies directly on different SONiC platforms
- Per‑ASIC plugins for vendor‑specific latency/error counters
- OTLP logs/spans for notable events (e.g., drop bursts, microbursts)
