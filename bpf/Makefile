BPF_CLANG ?= clang
BPF_CFLAGS ?= -O2 -g -target bpf -D__TARGET_ARCH_x86
BPF_OBJ=tc_ingress.bpf.o

all: $(BPF_OBJ)

%.bpf.o: %.bpf.c
	$(BPF_CLANG) $(BPF_CFLAGS) -c $< -o $@
