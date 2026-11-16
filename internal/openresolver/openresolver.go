// Package openresolver tests whether a DNS server will recurse for an arbitrary
// client — an "open resolver." Pointed at a device that shouldn't be a public
// resolver (a home router, a server, a printer), a positive result is an
// accidental open resolver: abusable for DNS amplification DDoS and a sign of
// misconfiguration. It works by asking the target to resolve a name it isn't
// authoritative for and seeing whether it does the recursion and answers.
package openresolver

import (
	"context"
	"errors"
	"strconv"
	"strings"

	mdns "github.com/miekg/dns"

	"github.com/hriday/meridian/internal/dnsx"
)

// probeName is a widely-resolvable name no ordinary server is authoritative for,
// so a recursive answer proves the target recursed on our behalf.
const probeName = "example.com"

// Report is the outcome of one open-resolver probe.
type Report struct {
	Resolver           string `json:"resolver"`
	Probe              string `json:"probe"`
	Open               bool   `json:"open"`
	RecursionAvailable bool   `json:"recursionAvailable"`
	Answered           bool   `json:"answered"`
	Rcode              string `json:"rcode"`
	ResponseBytes      int    `json:"responseBytes"`
	Summary            string `json:"summary"`
}

// queryFn is the raw-message exchange, behind a var so tests can inject a server.
var queryFn = dnsx.QueryMsg

// Test probes resolver (host or ip, :53 assumed) and reports whether it is an
// open recursive resolver. A transport error is returned (the target couldn't be
// reached), never guessed as "closed".
func Test(ctx context.Context, resolver string) (Report, error) {
	if strings.TrimSpace(resolver) == "" {
		return Report{}, errors.New("no resolver to test")
	}
	msg, err := queryFn(ctx, probeName, "A", resolver)
	if err != nil {
		return Report{}, errors.New("couldn't reach resolver: " + err.Error())
	}
	return classify(resolver, msg), nil
}

// classify grades a probe response: an open resolver recurses for us, so it
// answers NOERROR with the recursion-available bit set and at least one A record.
func classify(resolver string, msg *mdns.Msg) Report {
	r := Report{Resolver: resolver, Probe: probeName}
	if msg == nil {
		r.Summary = "No response from " + resolver + "."
		return r
	}
	r.Rcode = mdns.RcodeToString[msg.Rcode]
	r.RecursionAvailable = msg.RecursionAvailable
	r.ResponseBytes = msg.Len()
	for _, rr := range msg.Answer {
		if rr.Header().Rrtype == mdns.TypeA {
			r.Answered = true
			break
		}
	}
	r.Open = msg.Rcode == mdns.RcodeSuccess && msg.RecursionAvailable && r.Answered
	switch {
	case r.Open:
		r.Summary = "OPEN — " + resolver + " recurses for arbitrary clients (resolved " + probeName +
			" in a " + strconv.Itoa(r.ResponseBytes) + "-byte reply). If this isn't meant to be a public resolver, it's abusable for DNS amplification."
	case msg.RecursionAvailable && !r.Answered:
		r.Summary = "Not open — recursion is offered but the probe wasn't resolved (" + r.Rcode + ")."
	default:
		r.Summary = "Not open — " + resolver + " did not recurse for us (" + r.Rcode + ")."
	}
	return r
}
