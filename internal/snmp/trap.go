package snmp

import (
	"net/netip"
	"strconv"
	"strings"
	"time"

	g "github.com/gosnmp/gosnmp"
)

// TrapVar is one varbind of a received trap.
type TrapVar struct {
	OID   string `json:"oid"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// TrapMsg is one received SNMP trap. The packet's community is never copied here.
type TrapMsg struct {
	T        time.Time `json:"t"`
	Source   string    `json:"source"`
	Version  string    `json:"version"`
	TrapOID  string    `json:"trapOid"`
	TrapName string    `json:"trapName"`
	Varbinds []TrapVar `json:"varbinds"`
}

const oidSnmpTrapOID = "1.3.6.1.6.3.1.1.4.1.0" // snmpTrapOID.0

// trapNames maps the well-known generic trap OIDs to names.
var trapNames = map[string]string{
	"1.3.6.1.6.3.1.1.5.1": "coldStart",
	"1.3.6.1.6.3.1.1.5.2": "warmStart",
	"1.3.6.1.6.3.1.1.5.3": "linkDown",
	"1.3.6.1.6.3.1.1.5.4": "linkUp",
	"1.3.6.1.6.3.1.1.5.5": "authenticationFailure",
}

// genericTrapNames maps the v1 generic-trap numbers to names.
var genericTrapNames = [6]string{"coldStart", "warmStart", "linkDown", "linkUp", "authenticationFailure", "egpNeighborLoss"}

// ParseTrap decodes a v1/v2c trap datagram. It is defensive against malformed,
// attacker-reachable input (a decode error or panic yields ok=false) and never
// copies the packet's community onto the message. ok is false if it isn't a
// decodable SNMP packet.
func ParseTrap(datagram []byte, src netip.Addr) (m TrapMsg, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			m, ok = TrapMsg{}, false
		}
	}()
	if len(datagram) == 0 {
		return TrapMsg{}, false
	}
	pkt, err := (&g.GoSNMP{}).SnmpDecodePacket(datagram)
	if err != nil || pkt == nil {
		return TrapMsg{}, false
	}

	m = TrapMsg{T: time.Now(), Source: src.String()}
	switch pkt.Version {
	case g.Version1:
		m.Version = "v1"
	case g.Version2c:
		m.Version = "v2c"
	case g.Version3:
		m.Version = "v3"
	}

	for _, v := range pkt.Variables {
		raw := pduValue(v)
		m.Varbinds = append(m.Varbinds, TrapVar{OID: v.Name, Type: typeName(v.Type), Value: formatMIBValue(raw)})
		if trimOID(v.Name) == oidSnmpTrapOID { // v2c: the trap OID is this varbind's value
			if s, sok := raw.(string); sok {
				m.TrapOID = trimOID(s)
			}
		}
	}

	if m.TrapOID == "" && pkt.PDUType == g.Trap { // v1 trap
		m.TrapOID = v1TrapOID(pkt)
	}
	m.TrapName = trapNames[m.TrapOID]
	return m, true
}

func trimOID(s string) string { return strings.TrimPrefix(s, ".") }

// v1TrapOID builds the standard trap OID from a v1 trap's generic/specific
// numbers (enterpriseSpecific → enterprise.0.specific).
func v1TrapOID(pkt *g.SnmpPacket) string {
	switch gt := pkt.GenericTrap; {
	case gt >= 0 && gt < len(genericTrapNames):
		return "1.3.6.1.6.3.1.1.5." + strconv.Itoa(gt+1)
	case gt == 6:
		return trimOID(pkt.Enterprise) + ".0." + strconv.Itoa(pkt.SpecificTrap)
	default:
		return ""
	}
}
