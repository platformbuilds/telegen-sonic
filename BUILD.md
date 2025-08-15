# Build & Run

## Prereqs
- Go 1.23+
- clang/llvm (to build eBPF)
- `tc` (iproute2)
- Linux kernel with eBPF + BTF (SONiC 5.10+ is fine)
- bpffs mounted: `sudo mount -t bpf bpf /sys/fs/bpf`

## Build
```bash
make bpf
make build
```

Artifacts:
- `bin/agent` (REST server)
- `bin/telegen-sonic` (CLI)
- `bpf/tc_ingress.bpf.o` (eBPF program)

## Run (host)
```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=collector:4317
./bin/agent
# New shell
./bin/telegen-sonic start Ethernet16 120
```

## Docker
```bash
docker build -t ghcr.io/yourorg/telegen-sonic:latest -f deploy/Dockerfile .
docker run --name telegen-sonic --restart unless-stopped   --cap-add=NET_ADMIN --cap-add=BPF   -v /sys:/sys -v /proc:/proc -v /sys/fs/bpf:/sys/fs/bpf   -p 127.0.0.1:8080:8080   -e OTEL_EXPORTER_OTLP_ENDPOINT=collector:4317   ghcr.io/yourorg/telegen-sonic:latest
```

## Notes
- The CI workflow builds BPF, runs `go build` and tests, and builds the container image.
- The mirror provider is a placeholder—wire it to SONiC SPAN/ERSPAN on your platform.
