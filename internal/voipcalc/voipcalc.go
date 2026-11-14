// Package voipcalc computes VoIP call bandwidth: given a codec and
// packetization interval, how much a single RTP stream costs once you add the
// IP/UDP/RTP (and optionally Ethernet) overhead, and how much N concurrent calls
// need. Pure offline arithmetic for capacity planning — "how many calls fit on
// this link?"
package voipcalc

import (
	"fmt"
	"math"
	"strings"
)

// Codec is a voice codec's nominal bitrate and default packet interval.
type Codec struct {
	Name           string  `json:"name"`
	BitrateKbps    float64 `json:"bitrateKbps"`
	DefaultPtimeMs int     `json:"defaultPtimeMs"`
}

// Codecs is the supported list, in UI order.
func Codecs() []Codec {
	return []Codec{
		{"G.711 (PCMU/PCMA)", 64, 20},
		{"G.722", 64, 20},
		{"G.729", 8, 20},
		{"G.726-32", 32, 20},
		{"GSM", 13, 20},
		{"iLBC", 15.2, 20},
		{"Opus", 32, 20},
		{"Speex (wideband)", 16, 20},
	}
}

// Result is the bandwidth breakdown.
type Result struct {
	Codec         string  `json:"codec"`
	PtimeMs       int     `json:"ptimeMs"`
	PayloadBytes  int     `json:"payloadBytes"`
	PacketsPerSec float64 `json:"packetsPerSec"`
	OneWayKbps    float64 `json:"oneWayKbps"`       // one RTP stream, incl IP/UDP/RTP (40B)
	OneWayEthKbps float64 `json:"oneWayEthKbps"`    // same, incl Ethernet L2 (+18B)
	PerCallKbps   float64 `json:"perCallKbps"`      // both directions
	Calls         int     `json:"calls"`
	TotalKbps     float64 `json:"totalKbps"`        // perCall * calls
	TotalEthKbps  float64 `json:"totalEthKbps"`     // with Ethernet overhead
}

const (
	ipUDPRTP = 40 // IPv4(20) + UDP(8) + RTP(12)
	ethL2    = 18 // Ethernet header(14) + FCS(4)
)

// Calc computes the bandwidth for codecName at ptimeMs (0 => the codec default)
// for the given number of concurrent calls.
func Calc(codecName string, ptimeMs, calls int) (Result, error) {
	var c *Codec
	for i := range Codecs() {
		cc := Codecs()[i]
		if strings.EqualFold(cc.Name, codecName) || strings.EqualFold(firstWord(cc.Name), codecName) {
			c = &cc
			break
		}
	}
	if c == nil {
		return Result{}, fmt.Errorf("unknown codec %q", codecName)
	}
	if ptimeMs <= 0 {
		ptimeMs = c.DefaultPtimeMs
	}
	if calls <= 0 {
		calls = 1
	}

	payload := int(math.Round(c.BitrateKbps * float64(ptimeMs) / 8)) // kbps * ms / 8 = bytes
	pps := 1000.0 / float64(ptimeMs)
	oneWay := float64(payload+ipUDPRTP) * 8 / float64(ptimeMs) // kbps
	oneWayEth := float64(payload+ipUDPRTP+ethL2) * 8 / float64(ptimeMs)

	r := Result{
		Codec: c.Name, PtimeMs: ptimeMs, PayloadBytes: payload, PacketsPerSec: round(pps, 1),
		OneWayKbps: round(oneWay, 1), OneWayEthKbps: round(oneWayEth, 1),
		PerCallKbps: round(oneWay*2, 1), Calls: calls,
		TotalKbps:    round(oneWay*2*float64(calls), 1),
		TotalEthKbps: round(oneWayEth*2*float64(calls), 1),
	}
	return r, nil
}

func firstWord(s string) string {
	if i := strings.IndexAny(s, " ("); i > 0 {
		return s[:i]
	}
	return s
}

func round(v float64, dp int) float64 {
	p := math.Pow(10, float64(dp))
	return math.Round(v*p) / p
}
