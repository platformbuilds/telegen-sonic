# telegen-sonic

**telegen-sonic** is a lightweight eBPF-based telemetry agent for SONiC (Linux 5.x+) that exposes an HTTP API and exports OpenTelemetry metrics.
It compiles a CO-RE (Compile Once – Run Everywhere) tc-ingress program, pins maps under `/sys/fs/bpf/telegen-sonic`, and collects per-protocol packet/byte stats.

This single container runs:
- The HTTP **API server** (job control)
- The **metrics collector** (reads BPF maps and exports OTLP)
- **Mirror/ERSPAN** provisioning (optionally creates an erspan device as the monitor target)

## Security Checks & Reports
<!-- GOVULNCHECK-START -->
### govulncheck

| Field | Value |
|------:|:------|
| Tag | **release/mark-v1-0-0** |
| Scan Time (UTC) | 2025-08-16T11:10:11Z |
| Findings | **0** |
| Full Report | [reports/govulncheck/release-mark-v1-0-0.json](reports/govulncheck/release-mark-v1-0-0.json) |

_This section is auto-updated by the e2e release workflow._
<!-- GOVULNCHECK-END -->


---

## Features

- **eBPF tc ingress** classifier (CO-RE) with per-CPU maps
- **OpenTelemetry** metrics (`bpf.packets`, `bpf.bytes`)
- **Pinned maps**: `/sys/fs/bpf/telegen-sonic/{stats_percpu,if_stats_percpu}`
- **Job control API**: start/stop status endpoints
- **Mirroring**:
  - ERSPAN v2 provisioning (preferred in production)
  - Placeholder mode for CI/dev (no privileged ops required)

---

## Requirements

Runtime (recommended):
- Linux kernel **5.x+** with BTF available at `/sys/kernel/btf/vmlinux`
- Container capabilities: `CAP_NET_ADMIN`, `CAP_BPF` (or `--privileged`)
- `tc` and `ip` available in the container image
- `/sys/fs/bpf` mounted in the container for pinning
- OTLP endpoint (default `localhost:4317`)

Build (if building from source):
- `clang`, `llvm`, `bpftool`, `libelf-dev`, `libbpf-dev`
- Go 1.23+

---

## Build

```bash
# Build eBPF objects + agent + CLI
make build

# Build multi-arch container and push to GHCR (requires buildx)
make docker
```

> The repo ships GitHub Actions to build/push images to GHCR and to run `govulncheck`.  
> A resilient workflow step discovers/installs `bpftool` on runners where the exact `linux-tools-$KREL` package is not present.

---

## Run (Container)

```bash
docker run --rm -d   --network host   --cap-add NET_ADMIN --cap-add BPF   -v /sys/fs/bpf:/sys/fs/bpf   -v /sys/kernel/btf:/sys/kernel/btf:ro   -e OTEL_EXPORTER_OTLP_ENDPOINT="collector:4317"   ghcr.io/platformbuilds/telegen-sonic:latest
```

The agent listens on **127.0.0.1:8080** inside the container and starts the metrics collector automatically.

---

## API

### Start a job
```bash
curl -sS -X POST http://127.0.0.1:8080/jobs/start   -H 'Content-Type: application/json'   -d '{
    "port": "Ethernet0",
    "direction": "ingress"
  }'
```

Typical response:
```json
{
  "job_id": "9b4e87cb-...",
  "status": "starting",
  "interface": "erspan0"
}
```

### Get job
```bash
curl -sS http://127.0.0.1:8080/jobs/<job_id>
```

### Stop job
```bash
curl -sS -X POST http://127.0.0.1:8080/jobs/<job_id>/stop
```

---

## OpenTelemetry Metrics

The agent exports:
- **`bpf.packets`** (Counter) — packets observed by tc ingress, attributes: `proto`, `ifindex` (optional)
- **`bpf.bytes`** (Histogram) — bytes observed by tc ingress, attributes: `proto`, `ifindex` (optional)

`proto` values: `ipv4`, `ipv6`, `icmp6`, `other`.

Configure the collector via environment:
```bash
-e OTEL_EXPORTER_OTLP_ENDPOINT="host:4317"
```

---

## Mirroring

By default the agent attempts **ERSPAN v2** provisioning (requires `ip` and CAP_NET_ADMIN). If ERSPAN is not configured or fails, it falls back to **placeholder mode** which returns a stable interface name (e.g., `erspan0`) but performs no privileged operations—useful for CI/dev.

### Environment Variables

| Variable                  | Required | Default     | Description                                      |
|--------------------------|----------|-------------|--------------------------------------------------|
| `TELEGEN_MIRROR_MODE`    | no       | `erspan`    | `erspan` or `placeholder`                        |
| `TELEGEN_ERSPAN_NAME`    | no       | `erspan0`   | Netdev name to create/reuse                      |
| `TELEGEN_ERSPAN_DEV`     | no       | `spec.Port` | Source device/port to mirror                     |
| `TELEGEN_ERSPAN_REMOTE`  | yes*     |             | Remote IPv4 (ERSPAN tunnel destination)          |
| `TELEGEN_ERSPAN_LOCAL`   | yes*     |             | Local IPv4 (ERSPAN tunnel source)                |
| `TELEGEN_ERSPAN_KEY`     | no       | `10`        | ERSPAN key (session id)                          |
| `TELEGEN_ERSPAN_TTL`     | no       | `64`        | Outer IP TTL                                     |
| `TELEGEN_ERSPAN_TOS`     | no       | `inherit`   | TOS/DSCP (e.g., `inherit` or numeric)            |

\* Only required when `TELEGEN_MIRROR_MODE=erspan`.

### ERSPAN Example

```bash
docker run --rm -d   --network host   --cap-add NET_ADMIN --cap-add BPF   -v /sys/fs/bpf:/sys/fs/bpf   -v /sys/kernel/btf:/sys/kernel/btf:ro   -e OTEL_EXPORTER_OTLP_ENDPOINT="collector:4317"   -e TELEGEN_MIRROR_MODE=erspan   -e TELEGEN_ERSPAN_REMOTE=192.0.2.100   -e TELEGEN_ERSPAN_LOCAL=192.0.2.10   -e TELEGEN_ERSPAN_DEV=Ethernet0   -e TELEGEN_ERSPAN_KEY=42   ghcr.io/platformbuilds/telegen-sonic:latest
```

### Placeholder Example (CI/dev)

```bash
docker run --rm -it   -e TELEGEN_MIRROR_MODE=placeholder   ghcr.io/platformbuilds/telegen-sonic:latest
```

---

## CO-RE Compatibility on SONiC

- Works across **5.x** kernels when **BTF** is available (`/sys/kernel/btf/vmlinux`).  
- If BTF is missing, either provide a matching BTF file (BTFHub) or enable BTF in the SONiC image. As a last resort, compile on-box.

Recommended container flags:
```bash
--cap-add NET_ADMIN --cap-add BPF -v /sys/fs/bpf:/sys/fs/bpf -v /sys/kernel/btf:/sys/kernel/btf:ro
```

---

## Versioning

Binaries embed:
- `main.version` — `release/mark-v{major}-{minor}-{bugfix}` on tag builds (e.g., `v1.2.3` → `release/mark-v1-2-3`), otherwise `dev`.
- `main.commit` — short Git SHA
- `main.date` — UTC build time

---

## Development

```bash
# Run unit tests
make test

# Format & lint
make fmt
make lint
```

### Targeted tests (no root required)

- `pkg/monitor/attach_test.go` — stubs `tc` and uses an env override for object path
- `pkg/monitor/mirror_test.go` — stubs `ip` and exercises erspan/placeholder flows
- `pkg/monitor/collect_test.go` — constructor & helpers (no kernel access required)
