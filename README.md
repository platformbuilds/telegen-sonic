# Telegen‑Sonic

**Telegen‑Sonic** is a production‑grade, on‑demand **eBPF data‑plane telemetry agent** for SONiC NOS.  
It mirrors selected **ASIC‑forwarded** traffic to the CPU (via SPAN/ERSPAN), attaches **eBPF/tc** programs,
derives **OpenTelemetry (OTLP) metrics**, and exposes a **REST API + CLI** so network admins can trigger
short‑lived telemetry jobs **per port, on demand** — safely and repeatably.

> **Concurrency safety:** The agent **hard‑limits to 2 concurrent jobs** to protect switch resources.

---

## Why Telegen‑Sonic?

Switch ASICs forward most packets fully in hardware, bypassing the host networking stack. Traditional
host agents can’t “see” the data‑plane without extra work. Telegen‑Sonic solves this by:

1. **On‑demand mirroring (SPAN/ERSPAN)** to feed a CPU‑visible interface.
2. **Fast eBPF/tc classification** in the kernel to aggregate packet/byte counts and protocol stats.
3. **User‑space aggregation + OTLP export** using the OpenTelemetry SDK.
4. **Simple REST/CLI UX** to start/stop ephemeral jobs for a specific port, with filters, sampling,
   and built‑in safety limits.

This keeps your telemetry **precise, low‑overhead, and vendor‑neutral**.

---

## Features

- **On‑demand jobs**: “monitor port *Ethernet16* for 120s, DSCP=46, 1% sampling”
- **eBPF/tc** (CO‑RE‑ready skeleton) for efficient L2/L3/L4 parsing
- **OpenTelemetry metrics** (OTLP/gRPC) for `network.packets` and `network.bytes`
- **Max 2 concurrent jobs** (hard cap) to avoid performance hits
- **REST API + CLI** with consistent behavior and JSON results
- **Container‑friendly** (minimal capabilities: `CAP_BPF`, `CAP_NET_ADMIN`)

---

## Architecture (at a glance)

```
+--------------+      +------------------------+      +--------------------+
|  Admin/Tool  | ---> |  REST API / CLI Layer  | ---> |  Job Supervisor    |
+--------------+      +------------------------+      |  (max 2 jobs)      |
            ^                 |                       +--------------------+
            |                 v                                |
            |       +-----------------------+                  |
            |       |  Mirror Provider      | (SPAN/ERSPAN)    |
            |       +-----------+-----------+                  |
            |                   v                              v
            |       +-----------------------+      +-------------------------+
            |       |  tc/eBPF Program      | ---> |  Collector + OTLP       |
            |       |  (ingress/egress)     |      |  (OpenTelemetry)        |
            |       +-----------------------+      +-------------------------+
            |                                                |
            +----------------------------------------------- v
                                               Observability Backend (OTel)
```

- **Mirror Provider**: programs SPAN/ERSPAN or ACL‑mirrors to feed packets to a CPU interface.
- **tc/eBPF**: classifies packets (ETH/IP/TCP/UDP) and maintains per‑CPU counters.
- **Collector**: reads `tc`‑pinned BPF maps, computes deltas, exports OTLP metrics.
- **Supervisor**: manages job lifecycle + concurrency cap (2).

---

## REST API (excerpt)

- `POST /v1/monitor/jobs` – start a job (returns `job_id`), **429** if cap reached.
- `GET /v1/monitor/jobs/{job_id}` – job status.
- `GET /v1/monitor/jobs/{job_id}/results` – summary JSON.
- `DELETE /v1/monitor/jobs/{job_id}` – stop a job early.

OpenAPI: [`api/openapi.yaml`](api/openapi.yaml)

---

## CLI

```bash
# start a 120s job on Ethernet16
telegen-sonic start Ethernet16 120

# status
telegen-sonic status <JOB_ID>

# results
telegen-sonic results <JOB_ID>

# stop
telegen-sonic stop <JOB_ID>
```

---

## Build & Run

See **[BUILD.md](BUILD.md)** for full details. TL;DR:

```bash
make bpf
make build
export OTEL_EXPORTER_OTLP_ENDPOINT=collector:4317
./bin/agent
# new shell
./bin/telegen-sonic start Ethernet16 120
```

### Docker

```bash
docker build -t ghcr.io/yourorg/telegen-sonic:latest -f deploy/Dockerfile .
docker run --name telegen-sonic --restart unless-stopped   --cap-add=NET_ADMIN --cap-add=BPF   -v /sys:/sys -v /proc:/proc -v /sys/fs/bpf:/sys/fs/bpf   -p 127.0.0.1:8080:8080   -e OTEL_EXPORTER_OTLP_ENDPOINT=collector:4317   ghcr.io/yourorg/telegen-sonic:latest
```

---

## Production Notes

- **Security**: prefer a **Unix socket** (or mTLS) for the REST API; restrict CLI to local use.
- **Resource bounds**: default sampling (1%), top‑K and map sizes kept small, **2‑job cap** enforced.
- **SONiC integration**: replace the mirror provider with platform‑specific SPAN/ERSPAN programming.
- **Metrics**: current agent exports `network.packets` and `network.bytes`; extend with latency histograms,
  error counters, and top‑K flows by adding BPF maps + reader logic.

---

## Roadmap

- SPAN/ERSPAN providers for popular SONiC platforms (Broadcom/Dell/EdgeCore)
- Flow top‑K + LRU with OTel metrics and optional logs
- ns‑precision latency histograms and microburst detection
- gNMI hooks for streaming configuration/state telemetry
- CI hardening and integration tests with eBPF runtime checks

---

## License

MIT © Platformbuilds
