# --- build eBPF obj ---
FROM ubuntu:22.04 as bpf
RUN apt-get update && apt-get install -y clang llvm make gcc && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY bpf/ ./
RUN make

# --- build Go binaries ---
FROM golang:1.23 as gobuild
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN make build-agent && make build-cli

# --- final ---
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /
COPY --from=gobuild /src/bin/agent /agent
COPY --from=gobuild /src/bin/sonic-dpmon /sonic-dpmon
COPY --from=bpf /src/tc_ingress.bpf.o /bpf/tc_ingress.bpf.o
USER 65532:65532
ENTRYPOINT ["/agent"]
