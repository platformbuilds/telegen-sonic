# Deploy as SONiC container

Example:
```
docker run --name sonic-dpmon --restart unless-stopped   --cap-add=NET_ADMIN --cap-add=BPF   -v /sys:/sys -v /proc:/proc -v /sys/fs/bpf:/sys/fs/bpf   -p 127.0.0.1:8080:8080   -e OTEL_EXPORTER_OTLP_ENDPOINT=http://collector:4317   ghcr.io/platformbuilds/sonic-dpmon:latest
```
