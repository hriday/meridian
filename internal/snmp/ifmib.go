package snmp

import (
	"strconv"
	"strings"
)

// OID bases (no trailing index). IF-MIB, IF-MIB extensions, SNMPv2-MIB,
// HOST-RESOURCES-MIB.
const (
	oidIfDescr       = "1.3.6.1.2.1.2.2.1.2"
	oidIfType        = "1.3.6.1.2.1.2.2.1.3"
	oidIfSpeed       = "1.3.6.1.2.1.2.2.1.5"
	oidIfOperStatus  = "1.3.6.1.2.1.2.2.1.8"
	oidIfInOctets    = "1.3.6.1.2.1.2.2.1.10"
	oidIfInErrors    = "1.3.6.1.2.1.2.2.1.14"
	oidIfOutOctets   = "1.3.6.1.2.1.2.2.1.16"
	oidIfOutErrors   = "1.3.6.1.2.1.2.2.1.20"
	oidIfAlias       = "1.3.6.1.2.1.31.1.1.1.18"
	oidIfHighSpeed   = "1.3.6.1.2.1.31.1.1.1.15"
	oidIfHCInOctets  = "1.3.6.1.2.1.31.1.1.1.6"
	oidIfHCOutOctets = "1.3.6.1.2.1.31.1.1.1.10"

	oidSysDescr  = "1.3.6.1.2.1.1.1"
	oidSysUpTime = "1.3.6.1.2.1.1.3"
	oidSysName   = "1.3.6.1.2.1.1.5"

	oidHrProcessorLoad     = "1.3.6.1.2.1.25.3.3.1.2"
	oidHrStorageType       = "1.3.6.1.2.1.25.2.3.1.2"
	oidHrStorageAllocUnits = "1.3.6.1.2.1.25.2.3.1.4"
	oidHrStorageSize       = "1.3.6.1.2.1.25.2.3.1.5"
	oidHrStorageUsed       = "1.3.6.1.2.1.25.2.3.1.6"
	oidHrStorageRAM        = "1.3.6.1.2.1.25.2.1.2"
)

// Interface is one row of the device's interface table.
type Interface struct {
	Index      int    `json:"index"`
	Descr      string `json:"descr"`
	Alias      string `json:"alias"`
	SpeedBps   uint64 `json:"speedBps"`
	OperStatus string `json:"operStatus"`
}

// Counters is one interface's current octet/error counters and oper status.
type Counters struct {
	InOctets, OutOctets uint64
	InErrors, OutErrors uint64
	OperStatus          string
	Bits                int // 64 (HC) or 32
}

// SysInfo is the device identity.
type SysInfo struct {
	Descr, Name string
	UptimeTicks uint32
}

// Resources is best-effort CPU/memory (HOST-RESOURCES-MIB).
type Resources struct {
	CPUPct, MemPct float64
	HasCPU, HasMem bool
}

func lastIndex(oid string) int {
	if i := strings.LastIndexByte(oid, '.'); i >= 0 {
		n, _ := strconv.Atoi(oid[i+1:])
		return n
	}
	return 0
}

func walkUint(c Client, root string) map[int]uint64 {
	out := map[int]uint64{}
	vars, err := c.Walk(root)
	if err != nil {
		return out
	}
	for _, p := range vars {
		out[lastIndex(p.OID)] = asUint64(p.Value)
	}
	return out
}

func walkInt(c Client, root string) map[int]int {
	out := map[int]int{}
	vars, err := c.Walk(root)
	if err != nil {
		return out
	}
	for _, p := range vars {
		out[lastIndex(p.OID)] = asInt(p.Value)
	}
	return out
}

func walkStr(c Client, root string) map[int]string {
	out := map[int]string{}
	vars, err := c.Walk(root)
	if err != nil {
		return out
	}
	for _, p := range vars {
		out[lastIndex(p.OID)] = asString(p.Value)
	}
	return out
}

func mapOper(n int) string {
	switch n {
	case 1:
		return "up"
	case 2:
		return "down"
	case 3:
		return "testing"
	case 5:
		return "dormant"
	case 6:
		return "not present"
	case 7:
		return "lower down"
	default:
		return "unknown"
	}
}

// ListInterfaces walks the interface table into one row per ifIndex, speed
// preferring ifHighSpeed (Mbps) over ifSpeed (bps).
func ListInterfaces(c Client) ([]Interface, error) {
	descr, err := c.Walk(oidIfDescr)
	if err != nil {
		return nil, err
	}
	var order []int
	m := map[int]*Interface{}
	for _, p := range descr {
		idx := lastIndex(p.OID)
		if _, ok := m[idx]; !ok {
			order = append(order, idx)
		}
		m[idx] = &Interface{Index: idx, Descr: asString(p.Value)}
	}
	alias := walkStr(c, oidIfAlias)
	high := walkUint(c, oidIfHighSpeed)
	low := walkUint(c, oidIfSpeed)
	oper := walkInt(c, oidIfOperStatus)
	for idx, iface := range m {
		iface.Alias = alias[idx]
		if h := high[idx]; h > 0 {
			iface.SpeedBps = h * 1_000_000
		} else {
			iface.SpeedBps = low[idx]
		}
		iface.OperStatus = mapOper(oper[idx])
	}
	out := make([]Interface, 0, len(order))
	for _, idx := range order {
		out = append(out, *m[idx])
	}
	return out, nil
}

// InterfaceCounters reads one interface's counters, preferring 64-bit HC octets
// and falling back to 32-bit when they're absent.
func InterfaceCounters(c Client, idx int) (Counters, error) {
	sfx := "." + strconv.Itoa(idx)
	ct := Counters{Bits: 64}
	hc, err := c.Get([]string{oidIfHCInOctets + sfx, oidIfHCOutOctets + sfx})
	if err != nil {
		return ct, err
	}
	if len(hc) >= 2 && hc[0].Value != nil && hc[1].Value != nil {
		ct.InOctets = asUint64(hc[0].Value)
		ct.OutOctets = asUint64(hc[1].Value)
	} else {
		ct.Bits = 32
		lo, err := c.Get([]string{oidIfInOctets + sfx, oidIfOutOctets + sfx})
		if err != nil {
			return ct, err
		}
		if len(lo) >= 2 {
			ct.InOctets = asUint64(lo[0].Value)
			ct.OutOctets = asUint64(lo[1].Value)
		}
	}
	meta, err := c.Get([]string{oidIfInErrors + sfx, oidIfOutErrors + sfx, oidIfOperStatus + sfx})
	if err != nil {
		return ct, err
	}
	if len(meta) >= 3 {
		ct.InErrors = asUint64(meta[0].Value)
		ct.OutErrors = asUint64(meta[1].Value)
		ct.OperStatus = mapOper(asInt(meta[2].Value))
	}
	return ct, nil
}

// System reads the device identity.
func System(c Client) (SysInfo, error) {
	g, err := c.Get([]string{oidSysDescr + ".0", oidSysName + ".0", oidSysUpTime + ".0"})
	if err != nil {
		return SysInfo{}, err
	}
	var s SysInfo
	if len(g) >= 3 {
		s.Descr = asString(g[0].Value)
		s.Name = asString(g[1].Value)
		s.UptimeTicks = uint32(asUint64(g[2].Value))
	}
	return s, nil
}

// ResourceUsage returns best-effort CPU (mean of hrProcessorLoad) and memory
// (the hrStorageRam row), with Has* false when the device doesn't expose them.
func ResourceUsage(c Client) (Resources, error) {
	var r Resources
	if cores := walkInt(c, oidHrProcessorLoad); len(cores) > 0 {
		sum := 0
		for _, v := range cores {
			sum += v
		}
		r.CPUPct = float64(sum) / float64(len(cores))
		r.HasCPU = true
	}
	for idx, t := range walkStr(c, oidHrStorageType) {
		if strings.TrimPrefix(t, ".") == oidHrStorageRAM {
			g, err := c.Get([]string{
				oidHrStorageAllocUnits + "." + strconv.Itoa(idx),
				oidHrStorageSize + "." + strconv.Itoa(idx),
				oidHrStorageUsed + "." + strconv.Itoa(idx),
			})
			if err == nil && len(g) >= 3 {
				size := asUint64(g[1].Value)
				used := asUint64(g[2].Value)
				if size > 0 {
					r.MemPct = float64(used) / float64(size) * 100
					if r.MemPct > 100 { // some agents report used > size
						r.MemPct = 100
					}
					r.HasMem = true
				}
			}
			break
		}
	}
	return r, nil
}
