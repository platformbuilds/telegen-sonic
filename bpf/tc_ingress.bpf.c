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
// - No <linux/...> headers to avoid asm/types.h issues.
// - Userspace must aggregate per-CPU map values for totals.

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

/* ---- Local constants (avoid <linux/...> headers) ---- */
#ifndef ETH_P_IP
#define ETH_P_IP        0x0800
#endif
#ifndef ETH_P_IPV6
#define ETH_P_IPV6      0x86DD
#endif
#ifndef ETH_P_8021Q
#define ETH_P_8021Q     0x8100
#endif
#ifndef ETH_P_8021AD
#define ETH_P_8021AD    0x88A8
#endif
#ifndef IPPROTO_ICMPV6
#define IPPROTO_ICMPV6  58
#endif
#ifndef ETH_HLEN
#define ETH_HLEN        14
#endif
#ifndef VLAN_HLEN
#define VLAN_HLEN       4
#endif
#ifndef TC_ACT_OK
#define TC_ACT_OK       0
#endif
#ifndef BPF_ANY
#define BPF_ANY         0
#endif

/* ---- Stats structures & keys ---- */
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
    __u32 proto;   /* one of IDX_* */
};

/* ---- Maps ---- */
/* Per-CPU global proto stats */
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, IDX_MAX);
    __type(key, __u32);
    __type(value, struct proto_stats);
} stats_percpu SEC(".maps");

/* Per-CPU per-interface proto stats */
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_HASH);
    __uint(max_entries, 4096);
    __type(key, struct if_proto_key);
    __type(value, struct proto_stats);
} if_stats_percpu SEC(".maps");

/* ---- Bump helpers ---- */
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
    struct if_proto_key k = { .ifindex = ifindex, .proto = idx };
    struct proto_stats zero = {};
    struct proto_stats *st = bpf_map_lookup_elem(&if_stats_percpu, &k);
    if (!st) {
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

/* ---- Parse Ethernet + VLAN, return L3 proto and next header pointer ---- */
static __always_inline int parse_ethproto(void *data, void *data_end, __u16 *eth_proto, void **nh)
{
    /* Ensure we have Ethernet header */
    if ((char *)data + ETH_HLEN > (char *)data_end)
        return -1;

    /* EtherType at offset 12 */
    __u16 proto = bpf_ntohs(*(__be16 *)((char *)data + 12));
    void *cursor = (char *)data + ETH_HLEN;

#pragma clang loop unroll(full)
    for (int i = 0; i < 2; i++) { /* handle up to 2 VLAN tags */
        if (proto == ETH_P_8021Q || proto == ETH_P_8021AD) {
            if ((char *)cursor + VLAN_HLEN > (char *)data_end)
                return -1;
            /* VLAN ethertype is at +2 from VLAN header start */
            proto = bpf_ntohs(*(__be16 *)((char *)cursor + 2));
            cursor = (char *)cursor + VLAN_HLEN;
        } else {
            break;
        }
    }

    *eth_proto = proto;
    *nh = cursor;
    return 0;
}

/* ---- TC ingress program ---- */
SEC("tc")
int tc_ingress(struct __sk_buff *skb)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    __u32 pkt_len = (__u32)((long)data_end - (long)data);

    /* Prefer skb->ifindex; fallback to ingress_ifindex */
    __u32 ifidx = skb->ifindex ? skb->ifindex : skb->ingress_ifindex;

    __u16 proto = 0;
    void *nh = data;

    if (parse_ethproto(data, data_end, &proto, &nh) < 0) {
        bump_all(ifidx, IDX_OTHER, pkt_len);
        return TC_ACT_OK;
    }

    if (proto == ETH_P_IP) {
        /* minimal IPv4 header is 20 bytes */
        if ((char *)nh + 20 > (char *)data_end) {
            bump_all(ifidx, IDX_OTHER, pkt_len);
            return TC_ACT_OK;
        }
        bump_all(ifidx, IDX_IPV4, pkt_len);
        return TC_ACT_OK;
    }

    if (proto == ETH_P_IPV6) {
        /* fixed IPv6 header is 40 bytes */
        if ((char *)nh + 40 > (char *)data_end) {
            bump_all(ifidx, IDX_OTHER, pkt_len);
            return TC_ACT_OK;
        }
        bump_all(ifidx, IDX_IPV6, pkt_len);

        /* nexthdr field is byte 6 in IPv6 header */
        __u8 nexthdr = *(__u8 *)((char *)nh + 6);
        if (nexthdr == IPPROTO_ICMPV6) {
            bump_all(ifidx, IDX_ICMP6, pkt_len);
        }
        return TC_ACT_OK;
    }

    bump_all(ifidx, IDX_OTHER, pkt_len);
    return TC_ACT_OK;
}

/* Required license */
char LICENSE[] SEC("license") = "GPL";
