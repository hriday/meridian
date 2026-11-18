// Package ipp probes a networked printer over the Internet Printing Protocol
// (IPP, RFC 8010/8011) — a keyless Get-Printer-Attributes request — and reports
// the printer's state, what's wrong with it (out of paper, toner low, jam…),
// its supply levels and basics. It also checks the common print ports. It is
// read-only: it never sends a print job.
//
// buildRequest / decode implement just enough of the IPP binary format to ask
// one question and read the answer; they're pure, so they're unit-tested
// against hand-built, spec-compliant messages.
package ipp

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// IPP tags (RFC 8010).
const (
	tagOperation   = 0x01
	tagEndOfAttrs  = 0x03
	tagPrinter     = 0x04
	valInteger     = 0x21
	valEnum        = 0x23
	valText        = 0x41
	valName        = 0x42
	valKeyword     = 0x44
	valURI         = 0x45
	valCharset     = 0x47
	valLang        = 0x48
	opGetPrinterAt = 0x000B
)

// Doer performs an HTTP request (satisfied by *http.Client).
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type Supply struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
	Type  string `json:"type,omitempty"`
	Level int    `json:"level"`         // printer-reported level
	Max   int    `json:"max,omitempty"` // marker-high-level, if given (else assume 100)
}

type PortState struct {
	Port int    `json:"port"`
	Name string `json:"name"`
	Open bool   `json:"open"`
}

type Result struct {
	Host         string      `json:"host"`
	URI          string      `json:"uri"`
	Reachable    bool        `json:"reachable"` // IPP answered
	Name         string      `json:"name,omitempty"`
	MakeModel    string      `json:"makeModel,omitempty"`
	Location     string      `json:"location,omitempty"`
	Info         string      `json:"info,omitempty"`
	State        string      `json:"state"`        // idle | processing | stopped | unknown
	StateReasons []string    `json:"stateReasons"` // media-empty, toner-low, …
	Supplies     []Supply    `json:"supplies,omitempty"`
	QueuedJobs   int         `json:"queuedJobs"`
	Formats      []string    `json:"formats,omitempty"`
	IPPVersions  []string    `json:"ippVersions,omitempty"`
	Ports        []PortState `json:"ports"`
	Note         string      `json:"note,omitempty"`
}

var ippPaths = []string{"/ipp/print", "/ipp/printer", "/", "/ipp"}

// Probe queries the printer at host over IPP and checks the print ports.
func Probe(ctx context.Context, client Doer, host string) (Result, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return Result{}, fmt.Errorf("enter a printer IP or hostname")
	}
	r := Result{Host: host, State: "unknown"}
	r.Ports = checkPorts(ctx, host)

	for _, path := range ippPaths {
		uri := "ipp://" + host + ":631" + path
		body := buildRequest(uri)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+host+":631"+path, bytes.NewReader(body))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/ipp")
		req.Header.Set("User-Agent", "meridian (+https://github.com/hriday/meridian)")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || len(data) < 8 {
			continue
		}
		attrs, status, derr := decode(data)
		if derr != nil || status >= 0x0100 || len(attrs) == 0 {
			continue
		}
		r.URI = uri
		r.Reachable = true
		mapAttrs(&r, attrs)
		return r, nil
	}

	if !anyPortOpen(r.Ports) {
		return r, fmt.Errorf("no response on the print ports (631/9100/515) — is %s a printer, and reachable?", host)
	}
	r.Note = "A print port is open, but IPP (631) didn't answer Get-Printer-Attributes. The printer may speak only raw 9100 / LPD, or need a specific queue path."
	return r, nil
}

// --- IPP request ----------------------------------------------------------

func buildRequest(uri string) []byte {
	var b bytes.Buffer
	b.Write([]byte{0x02, 0x00}) // version 2.0
	_ = binary.Write(&b, binary.BigEndian, uint16(opGetPrinterAt))
	_ = binary.Write(&b, binary.BigEndian, uint32(1)) // request-id
	b.WriteByte(tagOperation)
	writeAttr(&b, valCharset, "attributes-charset", "utf-8")
	writeAttr(&b, valLang, "attributes-natural-language", "en")
	writeAttr(&b, valURI, "printer-uri", uri)
	req := []string{
		"printer-name", "printer-make-and-model", "printer-state", "printer-state-reasons",
		"printer-location", "printer-info", "marker-names", "marker-levels", "marker-colors",
		"marker-types", "marker-high-levels", "queued-job-count", "document-format-supported",
		"ipp-versions-supported",
	}
	for i, a := range req {
		name := "requested-attributes"
		if i > 0 {
			name = "" // additional value of the same attribute
		}
		writeAttr(&b, valKeyword, name, a)
	}
	b.WriteByte(tagEndOfAttrs)
	return b.Bytes()
}

func writeAttr(b *bytes.Buffer, tag byte, name, val string) {
	b.WriteByte(tag)
	_ = binary.Write(b, binary.BigEndian, uint16(len(name)))
	b.WriteString(name)
	_ = binary.Write(b, binary.BigEndian, uint16(len(val)))
	b.WriteString(val)
}

// --- IPP response ---------------------------------------------------------

type attrVal struct {
	tag byte
	str string
	num int
}

// decode reads the printer-group attributes from an IPP response and returns the
// status code.
func decode(data []byte) (map[string][]attrVal, uint16, error) {
	if len(data) < 8 {
		return nil, 0, fmt.Errorf("short IPP response")
	}
	status := binary.BigEndian.Uint16(data[2:4])
	attrs := map[string][]attrVal{}
	group := byte(0)
	lastName := ""
	i := 8
	for i < len(data) {
		tag := data[i]
		i++
		if tag <= 0x05 { // delimiter / end-of-attributes
			group = tag
			if tag == tagEndOfAttrs {
				break
			}
			continue
		}
		if i+2 > len(data) {
			break
		}
		nlen := int(binary.BigEndian.Uint16(data[i : i+2]))
		i += 2
		if i+nlen > len(data) {
			break
		}
		name := string(data[i : i+nlen])
		i += nlen
		if i+2 > len(data) {
			break
		}
		vlen := int(binary.BigEndian.Uint16(data[i : i+2]))
		i += 2
		if i+vlen > len(data) {
			break
		}
		val := data[i : i+vlen]
		i += vlen
		if nlen == 0 {
			name = lastName // additional value of a multi-valued attribute
		} else {
			lastName = name
		}
		if group != tagPrinter {
			continue
		}
		av := attrVal{tag: tag}
		switch tag {
		case valInteger, valEnum:
			if vlen == 4 {
				av.num = int(int32(binary.BigEndian.Uint32(val)))
			}
		default:
			av.str = string(val)
		}
		attrs[name] = append(attrs[name], av)
	}
	return attrs, status, nil
}

func mapAttrs(r *Result, a map[string][]attrVal) {
	r.Name = firstStr(a["printer-name"])
	r.MakeModel = firstStr(a["printer-make-and-model"])
	r.Location = firstStr(a["printer-location"])
	r.Info = firstStr(a["printer-info"])
	r.QueuedJobs = firstNum(a["queued-job-count"])
	r.State = stateName(firstNum(a["printer-state"]))
	r.StateReasons = allStr(a["printer-state-reasons"])
	// "none" is noise.
	r.StateReasons = dropNone(r.StateReasons)
	r.Formats = allStr(a["document-format-supported"])
	r.IPPVersions = allStr(a["ipp-versions-supported"])

	names := allStr(a["marker-names"])
	levels := allNum(a["marker-levels"])
	colors := allStr(a["marker-colors"])
	types := allStr(a["marker-types"])
	highs := allNum(a["marker-high-levels"])
	for i := range names {
		r.Supplies = append(r.Supplies, Supply{
			Name:  names[i],
			Level: at(levels, i),
			Color: supplyColor(names[i], at(colors, i)),
			Type:  at(types, i),
			Max:   at(highs, i),
		})
	}
}

// --- port check -----------------------------------------------------------

func checkPorts(ctx context.Context, host string) []PortState {
	ports := []struct {
		p int
		n string
	}{{631, "IPP"}, {9100, "raw / JetDirect"}, {515, "LPD"}}
	out := make([]PortState, 0, len(ports))
	for _, pr := range ports {
		out = append(out, PortState{Port: pr.p, Name: pr.n, Open: dialOpen(ctx, host, pr.p)})
	}
	return out
}

func dialOpen(ctx context.Context, host string, port int) bool {
	d := net.Dialer{Timeout: 2 * time.Second}
	c, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

func anyPortOpen(ps []PortState) bool {
	for _, p := range ps {
		if p.Open {
			return true
		}
	}
	return false
}

// --- helpers --------------------------------------------------------------

func stateName(n int) string {
	switch n {
	case 3:
		return "idle"
	case 4:
		return "processing"
	case 5:
		return "stopped"
	default:
		return "unknown"
	}
}

func firstStr(v []attrVal) string {
	if len(v) > 0 {
		return v[0].str
	}
	return ""
}
func firstNum(v []attrVal) int {
	if len(v) > 0 {
		return v[0].num
	}
	return 0
}
func allStr(v []attrVal) []string {
	out := make([]string, 0, len(v))
	for _, x := range v {
		out = append(out, x.str)
	}
	return out
}
func allNum(v []attrVal) []int {
	out := make([]int, 0, len(v))
	for _, x := range v {
		out = append(out, x.num)
	}
	return out
}

// supplyColor picks the swatch colour for a supply. The supply NAME is
// authoritative: many printers report nonsense in marker-colors (a real Samsung
// C460 reports black toner as #FFFFFF, cyan as #FF5110, …), but the name the
// user reads — "Black Toner", "Cyan Toner" — is reliable. We fall back to the
// printer's reported colour only for supplies whose name has no colour word
// (drums, imaging units, waste bottles), and never invent one.
func supplyColor(name, reported string) string {
	lname := strings.ToLower(name)
	for _, c := range supplyColorKeywords {
		if strings.Contains(lname, c.key) {
			return c.hex
		}
	}
	if isHexColor(reported) {
		return strings.ToUpper(reported)
	}
	return ""
}

// Canonical swatches for standard toner/ink colour words. First match wins, so
// order only matters where one word contains another (none here).
var supplyColorKeywords = []struct{ key, hex string }{
	{"black", "#000000"},
	{"cyan", "#00AEEF"},
	{"magenta", "#EC008C"},
	{"yellow", "#FFF200"},
	{"grey", "#9E9E9E"},
	{"gray", "#9E9E9E"},
	{"red", "#E53935"},
	{"green", "#43A047"},
	{"blue", "#1E88E5"},
	{"orange", "#FB8C00"},
	{"white", "#FFFFFF"},
}

// isHexColor reports whether s is a "#RRGGBB" string, without pulling in regexp.
func isHexColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for i := 1; i < 7; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func at[T any](s []T, i int) T {
	var zero T
	if i < len(s) {
		return s[i]
	}
	return zero
}
func dropNone(s []string) []string {
	out := make([]string, 0, len(s))
	for _, x := range s {
		if x != "none" && strings.TrimSpace(x) != "" {
			out = append(out, x)
		}
	}
	return out
}
