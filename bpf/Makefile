BPFOBJS := tc_ingress.bpf.o
CLANG  ?= clang

UNAME_M := $(shell uname -m)
KREL    := $(shell uname -r)

# Map runner arch to __TARGET_ARCH_*
BPF_ARCH := x86
ifeq ($(UNAME_M),x86_64)
  BPF_ARCH := x86
else ifeq ($(UNAME_M),aarch64)
  BPF_ARCH := arm64
else ifeq ($(UNAME_M),arm64)
  BPF_ARCH := arm64
endif

CFLAGS := -O2 -g -target bpf -D__TARGET_ARCH_$(BPF_ARCH) \
  -Wall -Wno-unused-value -Wno-pointer-sign -Wno-compare-distinct-pointer-types \
  -I$(CURDIR) -I$(CURDIR)/include

.PHONY: all clean
all: vmlinux.h $(BPFOBJS)

# Allow overriding bpftool path via env: `make BPFOOL=/custom/path/bpftool`
BPFOOL ?=

define GEN_VMLINUX
@if [ -r /sys/kernel/btf/vmlinux ]; then \
  if command -v bpftool >/dev/null 2>&1; then \
    bpftool btf dump file /sys/kernel/btf/vmlinux format c > $@; \
  elif [ -n "$(BPFOOL)" ] && [ -x "$(BPFOOL)" ]; then \
    "$(BPFOOL)" btf dump file /sys/kernel/btf/vmlinux format c > $@; \
  elif [ -x /usr/lib/linux-tools-$(KREL)/bpftool ]; then \
    /usr/lib/linux-tools-$(KREL)/bpftool btf dump file /sys/kernel/btf/vmlinux format c > $@; \
  else \
    echo "bpftool not found. Set BPFOOL=/path/to/bpftool or install linux-tools-$(KREL)."; \
    exit 1; \
  fi; \
else \
  echo "No BTF at /sys/kernel/btf/vmlinux; cannot generate vmlinux.h"; \
  exit 1; \
fi
endef

vmlinux.h:
	$(GEN_VMLINUX)

%.bpf.o: %.bpf.c vmlinux.h
	$(CLANG) $(CFLAGS) -c $< -o $@

clean:
	rm -f $(BPFOBJS) vmlinux.h
