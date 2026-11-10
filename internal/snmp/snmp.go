// Package snmp is Meridian's offline-testable SNMP engine. It wraps a gosnmp
// v2c session behind a small Client interface so tools call it and tests inject
// a fake. The community string is handled by the caller; this package never
// logs it.
package snmp

import (
	"fmt"
	"time"

	g "github.com/gosnmp/gosnmp"
)

// PDU is one variable binding: an OID, its ASN.1 type name, and its decoded
// value (nil when the OID is absent — noSuchObject/noSuchInstance/endOfMibView).
type PDU struct {
	OID   string
	Type  string
	Value any
}

// Client is the subset of an SNMP session the engine needs; injected for tests.
type Client interface {
	Get(oids []string) ([]PDU, error)
	Walk(root string) ([]PDU, error)
	Close() error
}

type gclient struct{ s *g.GoSNMP }

// Dial opens an SNMPv2c session to device using community. It retries once, so
// a dropped poll of a known device doesn't read as a failure (used by the
// continuous monitors). A discovery sweep wants no retries — see dialRetries.
func Dial(device, community string, timeout time.Duration) (Client, error) {
	return dialRetries(device, community, timeout, 1)
}

// dialRetries opens an SNMPv2c session with an explicit retry count. A sweep
// passes 0 so a silent host costs exactly one timeout, not a retransmit.
func dialRetries(device, community string, timeout time.Duration, retries int) (Client, error) {
	s := &g.GoSNMP{
		Target:    device,
		Port:      161,
		Community: community,
		Version:   g.Version2c,
		Timeout:   timeout,
		Retries:   retries,
		MaxOids:   g.MaxOids,
	}
	if err := s.Connect(); err != nil {
		return nil, fmt.Errorf("snmp connect to %s: %w", device, err)
	}
	return &gclient{s}, nil
}

func (c *gclient) Get(oids []string) ([]PDU, error) {
	pkt, err := c.s.Get(oids)
	if err != nil {
		return nil, err
	}
	return conv(pkt.Variables), nil
}

func (c *gclient) Walk(root string) ([]PDU, error) {
	vars, err := c.s.BulkWalkAll(root)
	if err != nil {
		return nil, err
	}
	return conv(vars), nil
}

func (c *gclient) Close() error {
	if c.s != nil && c.s.Conn != nil {
		return c.s.Conn.Close()
	}
	return nil
}

func conv(vars []g.SnmpPDU) []PDU {
	out := make([]PDU, len(vars))
	for i, v := range vars {
		out[i] = PDU{OID: v.Name, Type: typeName(v.Type), Value: pduValue(v)}
	}
	return out
}

func pduValue(v g.SnmpPDU) any {
	switch v.Type {
	case g.NoSuchObject, g.NoSuchInstance, g.EndOfMibView:
		return nil
	case g.OctetString:
		if b, ok := v.Value.([]byte); ok {
			return string(b)
		}
	}
	return v.Value
}

func asUint64(v any) uint64 {
	switch n := v.(type) {
	case uint64:
		return n
	case uint:
		return uint64(n)
	case uint32:
		return uint64(n)
	case int:
		if n >= 0 {
			return uint64(n)
		}
	case int64:
		if n >= 0 {
			return uint64(n)
		}
	}
	return 0
}

func asInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	}
	return 0
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
