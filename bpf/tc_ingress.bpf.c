// SPDX-License-Identifier: GPL-2.0
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/ipv6.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

struct dp_counters {
    __u64 packets;
    __u64 bytes;
};

// Global per-CPU array[1] -> aggregate stats
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct dp_counters);
} stats SEC(".maps");

// Simple per-protocol counters (index: 1=TCP, 2=UDP, 3=ICMP/ICMPv6, 4=Other)
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 5);
    __type(key, __u32);
    __type(value, struct dp_counters);
} proto SEC(".maps");

static __always_inline void bump(struct bpf_map_def *unused, __u32 idx, __u32 bytes) {
    // This helper signature is not used; we directly use the map 'proto' and 'stats' above.
}

static __always_inline void add_ctr(void *map, __u32 idx, __u32 bytes) {
    struct dp_counters *c = bpf_map_lookup_elem(map, &idx);
    if (!c) return;
    __sync_fetch_and_add(&c->packets, 1);
    __sync_fetch_and_add(&c->bytes, bytes);
}

SEC("classifier")
int tc_ingress(struct __sk_buff *skb) {
    __u32 bytes = skb->len;

    // Update global stats[0]
    __u32 zero = 0;
    add_ctr(&stats, zero, bytes);

    // Parse L2
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    struct ethhdr *eth = data;
    if ((void*)(eth + 1) > data_end) return BPF_OK;

    __u16 h_proto = bpf_ntohs(eth->h_proto);

    // IPv4
    if (h_proto == ETH_P_IP) {
        struct iphdr *iph = (void *)(eth + 1);
        if ((void*)(iph + 1) > data_end) goto other;
        __u8 proto_num = iph->protocol;
        if (proto_num == IPPROTO_TCP) {
            __u32 idx = 1;
            add_ctr(&proto, idx, bytes);
        } else if (proto_num == IPPROTO_UDP) {
            __u32 idx = 2;
            add_ctr(&proto, idx, bytes);
        } else if (proto_num == IPPROTO_ICMP) {
            __u32 idx = 3;
            add_ctr(&proto, idx, bytes);
        } else {
            __u32 idx = 4;
            add_ctr(&proto, idx, bytes);
        }
        return BPF_OK;
    }
    // IPv6
    if (h_proto == ETH_P_IPV6) {
        struct ipv6hdr *ip6 = (void *)(eth + 1);
        if ((void*)(ip6 + 1) > data_end) goto other;
        __u8 proto6 = ip6->nexthdr;
        if (proto6 == IPPROTO_TCP) {
            __u32 idx = 1; add_ctr(&proto, idx, bytes);
        } else if (proto6 == IPPROTO_UDP) {
            __u32 idx = 2; add_ctr(&proto, idx, bytes);
        } else if (proto6 == IPPROTO_ICMPV6) {
            __u32 idx = 3; add_ctr(&proto, idx, bytes);
        } else {
            __u32 idx = 4; add_ctr(&proto, idx, bytes);
        }
        return BPF_OK;
    }

other:
    {
        __u32 idx = 4;
        add_ctr(&proto, idx, bytes);
    }
    return BPF_OK;
}

char _license[] SEC("license") = "GPL";
