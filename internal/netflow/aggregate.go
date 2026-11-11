package netflow

import (
	"net/netip"
	"sort"
	"strconv"
	"time"
)

// Conversation is an aggregated src→dst:port/proto talker over a window.
type Conversation struct {
	SrcIP   string `json:"srcIp"`
	DstIP   string `json:"dstIp"`
	DstPort int    `json:"dstPort"`
	Proto   string `json:"proto"`
	Bytes   uint64 `json:"bytes"`
	Packets uint64 `json:"packets"`
	Flows   int    `json:"flows"`
}

type convKey struct {
	src, dst netip.Addr
	dstPort  uint16
	proto    uint8
}

// TopTalkers aggregates the records within the window into conversations, ranked
// by bytes (or packets), returning the top N and how many seconds of the window
// the retained records actually cover (so a view that can't reach back the full
// window — the ring evicted older flows — can say so).
func TopTalkers(records []FlowRecord, window time.Duration, now time.Time, byPackets bool, topN int) ([]Conversation, int) {
	cutoff := now.Add(-window)
	agg := map[convKey]*Conversation{}
	oldest := now
	kept := 0
	for i := range records {
		f := &records[i]
		if f.T.Before(cutoff) {
			continue
		}
		kept++
		if f.T.Before(oldest) {
			oldest = f.T
		}
		k := convKey{f.Src, f.Dst, f.DstPort, f.Proto}
		c := agg[k]
		if c == nil {
			c = &Conversation{SrcIP: f.Src.String(), DstIP: f.Dst.String(), DstPort: int(f.DstPort), Proto: protoName(f.Proto)}
			agg[k] = c
		}
		c.Bytes += f.Bytes
		c.Packets += f.Packets
		c.Flows++
	}

	out := make([]Conversation, 0, len(agg))
	for _, c := range agg {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if byPackets {
			return out[i].Packets > out[j].Packets
		}
		return out[i].Bytes > out[j].Bytes
	})
	if topN > 0 && len(out) > topN {
		out = out[:topN]
	}

	covered := 0
	if kept > 0 {
		covered = int(now.Sub(oldest).Seconds()) + 1
	}
	return out, covered
}

func protoName(p uint8) string {
	switch p {
	case 1:
		return "ICMP"
	case 6:
		return "TCP"
	case 17:
		return "UDP"
	case 47:
		return "GRE"
	case 50:
		return "ESP"
	default:
		return "proto " + strconv.Itoa(int(p))
	}
}
