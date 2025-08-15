BPF_OBJ=tc_ingress.bpf.o

all: $(BPF_OBJ)

%.bpf.o: %.bpf.c
	clang -O2 -g -target bpf -c $< -o $@
