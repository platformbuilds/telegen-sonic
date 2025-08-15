BPFOBJS := tc_ingress.bpf.o
CLANG  ?= clang

# Map runner arch to __TARGET_ARCH_*
UNAME_M := $(shell uname -m)
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

vmlinux.h:
	@if [ -r /sys/kernel/btf/vmlinux ]; then \
	  bpftool btf dump file /sys/kernel/btf/vmlinux format c > $@; \
	else \
	  echo "No BTF available at /sys/kernel/btf/vmlinux; cannot generate vmlinux.h"; \
	  exit 1; \
	fi

%.bpf.o: %.bpf.c vmlinux.h
	$(CLANG) $(CFLAGS) -c $< -o $@

clean:
	rm -f $(BPFOBJS) vmlinux.h
