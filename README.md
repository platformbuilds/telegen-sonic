# sonic-dpmon

Production-grade scaffold for a SONiC eBPF data-plane telemetry agent.

Features:
- On-demand per-port monitoring via REST + CLI
- OTLP export (OpenTelemetry)
- tc/CO-RE eBPF for packet classification & metrics
- Hard concurrency cap: **max 2 active jobs**
- Pluggable SPAN/ERSPAN mirror provisioner
- Minimal capabilities (CAP_BPF, CAP_NET_ADMIN), bpffs mount

See `Wiki.md` for full design.
