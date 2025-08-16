# bpf/Makefile
# Build tc_ingress eBPF object with CO-RE + generated vmlinux.h

# ---- Tools (override if needed) ----
CLANG       ?= clang
LLVM_STRIP  ?= llvm-strip
BPFTOOL     ?= bpftool              # preferred name on PATH
BPFOOL      ?=                      # alt path override via env (kept for your CI)

# ---- Kernel / Arch detection ----
UNAME_M     := $(shell uname -m)
KREL        := $(shell uname -r)

# Override with: make TARGET_ARCH=arm64
TARGET_ARCH ?=
ifeq ($(TARGET_ARCH),)
  ifeq ($(UNAME_M),x86_64)
    TARGET_ARCH := x86
  else ifeq ($(UNAME_M),aarch64)
    TARGET_ARCH := arm64
  else ifeq ($(UNAME_M),arm64)
    TARGET_ARCH := arm64
  else
    TARGET_ARCH := x86
  endif
endif

# ---- Inputs / Outputs ----
SRC_DIR     := $(CURDIR)
INC_DIR     := $(CURDIR)/include
BPFSRCS     := $(wildcard $(SRC_DIR)/*.bpf.c)
BPFOBJS     := $(patsubst %.bpf.c,%.bpf.o,$(BPFSRCS))
VMLINUX     := vmlinux.h

# ---- Flags ----
CFLAGS := -O2 -g -target bpf -D__TARGET_ARCH_$(TARGET_ARCH) \
  -Wall -Wno-unused-value -Wno-pointer-sign -Wno-compare-distinct-pointer-types \
  -I$(SRC_DIR) -I$(INC_DIR)

# ---- Pretty/quiet build ----
V ?= 0
ifeq ($(V),0)
  Q := @
else
  Q :=
endif

.DELETE_ON_ERROR:
.PHONY: all clean print-vars

all: $(VMLINUX) $(BPFOBJS)

# -------- vmlinux.h generation (tries PATH, env BPFOOL, linux-tools path) --------
define GEN_VMLINUX
  if [ -r /sys/kernel/btf/vmlinux ]; then \
    if command -v "$(BPFTOOL)" >/dev/null 2>&1; then \
      echo ">>> Generating $(VMLINUX) with $(BPFTOOL) (PATH)"; \
      "$(BPFTOOL)" btf dump file /sys/kernel/btf/vmlinux format c > $(VMLINUX); \
    elif [ -n "$(BPFOOL)" ] && [ -x "$(BPFOOL)" ]; then \
      echo ">>> Generating $(VMLINUX) with BPFOOL=$(BPFOOL)"; \
      "$(BPFOOL)" btf dump file /sys/kernel/btf/vmlinux format c > $(VMLINUX); \
    elif [ -x /usr/lib/linux-tools-$(KREL)/bpftool ]; then \
      echo ">>> Generating $(VMLINUX) with /usr/lib/linux-tools-$(KREL)/bpftool"; \
      /usr/lib/linux-tools-$(KREL)/bpftool btf dump file /sys/kernel/btf/vmlinux format c > $(VMLINUX); \
    else \
      echo "ERROR: bpftool not found. Install linux-tools-$(KREL) or export BPFOOL=/path/to/bpftool"; \
      exit 1; \
    fi; \
  else \
    echo "ERROR: /sys/kernel/btf/vmlinux not found (no BTF). Cannot generate $(VMLINUX)."; \
    exit 1; \
  fi
endef

$(VMLINUX):
	$(Q)$(GEN_VMLINUX)

# -------- Compile each .bpf.c into .bpf.o --------
%.bpf.o: %.bpf.c $(VMLINUX)
	@echo ">>> Building $@ (ARCH=$(TARGET_ARCH))"
	$(Q)$(CLANG) $(CFLAGS) -c $< -o $@
	# Strip debug info (optional, best-effort)
	$(Q)-$(LLVM_STRIP) -g $@ >/dev/null 2>&1 || true

clean:
	@echo ">>> Cleaning eBPF objects"
	$(Q)rm -f $(BPFOBJS) $(VMLINUX)

print-vars:
	@echo "CLANG=$(CLANG)"
	@echo "LLVM_STRIP=$(LLVM_STRIP)"
	@echo "BPFTOOL=$(BPFTOOL)"
	@echo "BPFOOL=$(BPFOOL)"
	@echo "UNAME_M=$(UNAME_M)"
	@echo "KREL=$(KREL)"
	@echo "TARGET_ARCH=$(TARGET_ARCH)"
	@echo "BPFSRCS=$(BPFSRCS)"
	@echo "BPFOBJS=$(BPFOBJS)"