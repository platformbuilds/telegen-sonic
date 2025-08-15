// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

SEC("classifier")
int tc_ingress(struct __sk_buff *skb) {
    // TODO: parse L2/L3/L4, sampling, counters via maps
    return BPF_OK;
}

char _license[] SEC("license") = "GPL";
