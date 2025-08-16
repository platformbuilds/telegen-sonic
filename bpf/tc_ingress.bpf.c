// SPDX-License-Identifier: GPL-2.0
// tc_ingress.bpf.c - CO-RE TC ingress program for telegen-sonic
//
// Features:
// - Per-CPU global stats by protocol: IPv4, IPv6, ICMPv6, Other
// - Per-CPU per-interface (ifindex) stats by protocol
// - VLAN-aware Ethernet parsing (802.1Q / 802.1ad)
// - Safe bounds checks for verifier
// - Attach as tc clsact/ingress
//
// Notes:
// - Requires vmlinux.h generated from /sys/kernel/btf/vmlinux (Makefile).
// - No <linux/bpf.h> or bcc-style maps.
// - Userspace must aggregate per-CPU map values for totals.

#include "vmlinux.h"

#include <bpf/bpf_endian.h>
#include <bpf/bpf_helpers.h>

// UAPI headers for protocol constants / structs
#include <linux/if_ether.h>   // ETH_P_*
#include <linux/if_vlan.h>    // struct vlan_hdr
#include <linux/in.h>         // IPPROTO_*
#include <linux/ip.h>         // struct iphdr
#include <linux/ipv6.h>       // struct ipv6hdr
#include <linux/icmpv6.h>     // ICMPv6 constants
#include <linux/pkt_cls.h>    // TC_ACT_*

// ---- Stats structures & keys ----

struct proto_stats {
    __u64 packets;
    __u64 bytes;
};

enum {
    IDX_IPV4 = 0,
    IDX_IPV6 = 1,
    IDX_ICMP6 = 2,
    IDX_OTHER = 3,
    IDX_MAX
};

struct if_proto_key {
    __u32 ifindex;
    __u32 proto;   // one of IDX_*
};

// ---- Maps ----
// Per-CPU global proto stats
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, IDX_MAX);
    __type(key, __u32);
    __type(value, struct proto_stats);
} stats_percpu SEC(".maps");

// Per-CPU per-interface proto stats (bounded size; tune as needed)
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_HASH);
    __uint(max_entries, 4096);
    __type(key, struct if_proto_key);
    __type(value, struct proto_stats);
} if_stats_percpu SEC(".maps");

// ---- Helpers ----

static __always_inline void bump_global(__u32 idx, __u32 bytes)
{
    struct proto_stats *st = bpf_map_lookup_elem(&stats_percpu, &idx);
    if (st) {
        st->packets++;
        st->bytes += bytes;
    }
}

static __always_inline void bump_if(__u32 ifindex, __u32 idx, __u32 bytes)
{
    struct if_proto_key k = {
        .ifindex = ifindex,
        .proto   = idx,
    };
    struct proto_stats zero = {};
    struct proto_stats *st = bpf_map_lookup_elem(&if_stats_percpu, &k);
    if (!st) {
        // Initialize an entry lazily
        bpf_map_update_elem(&if_stats_percpu, &k, &zero, BPF_ANY);
        st = bpf_map_lookup_elem(&if_stats_percpu, &k);
        if (!st)
            return;
    }
    st->packets++;
    st->bytes += bytes;
}

static __always_inline void bump_all(__u32 ifindex, __u32 idx, __u32 bytes)
{
    bump_global(idx, bytes);
    if (ifindex)
        bump_if(ifindex, idx, bytes);
}

static __always_inline int parse_ethproto(void *data, void *data_end, __u16 *eth_proto, void **nh)
{
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return -1;

    __u16 proto = bpf_ntohs(eth->h_proto);
    void *cursor = (void *)(eth + 1);

#pragma clang loop unroll(full)
    for (int i = 0; i < 2; i++) { // handle up to 2 VLAN tags
        if (proto == ETH_P_8021Q || proto == ETH_P_8021AD) {
            if ((char *)cursor + sizeof(struct vlan_hdr) > (char *)data_end)
                return -1;
            struct vlan_hdr *vh = cursor;
            proto = bpf_ntohs(vh->h_vlan_encapsulated_proto);
            cursor = (char *)cursor + sizeof(struct vlan_hdr);
        } else {
            break;
        }
    }

    *eth_proto = proto;
    *nh = cursor;
    return 0;
}

// ---- TC ingress program ----

SEC("tc")
int tc_ingress(struct __sk_buff *skb)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    __u32 pkt_len = (__u32)((long)data_end - (long)data);

    // Choose an ifindex for per-port stats. Prefer skb->ifindex; fallback to ingress_ifindex.
    __u32 ifidx = skb->ifindex;
    if (!ifidx)
        ifidx = skb->ingress_ifindex;

    __u16 proto = 0;
    void *nh = data;

    if (parse_ethproto(data, data_end, &proto, &nh) < 0) {
        bump_all(ifidx, IDX_OTHER, pkt_len);
        return TC_ACT_OK;
    }

    if (proto == ETH_P_IP) {
        if ((char *)nh + sizeof(struct iphdr) > (char *)data_end) {
            bump_all(ifidx, IDX_OTHER, pkt_len);
            return TC_ACT_OK;
        }
        // IPv4
        bump_all(ifidx, IDX_IPV4, pkt_len);
        return TC_ACT_OK;
    }

    if (proto == ETH_P_IPV6) {
        if ((char *)nh + sizeof(struct ipv6hdr) > (char *)data_end) {
            bump_all(ifidx, IDX_OTHER, pkt_len);
            return TC_ACT_OK;
        }
        struct ipv6hdr *ip6h = nh;
        bump_all(ifidx, IDX_IPV6, pkt_len);

        if (ip6h->nexthdr == IPPROTO_ICMPV6) {
            bump_all(ifidx, IDX_ICMP6, pkt_len);
        }
        return TC_ACT_OK;
    }

    bump_all(ifidx, IDX_OTHER, pkt_len);
    return TC_ACT_OK;
}

// Required license
char LICENSE[] SEC("license") = "GPL";
