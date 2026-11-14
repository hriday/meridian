package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hriday/meridian/internal/dnsprop"
)

func TestPrintDNSPropConsensus(t *testing.T) {
	var buf bytes.Buffer
	printDNSProp(&buf, dnsprop.Report{
		Name: "example.com", Type: "A", Consensus: true, Propagated: 2, Total: 2,
		Groups: []dnsprop.Group{{Values: []string{"203.0.113.5"}, Count: 2, Resolvers: []string{"Google", "Cloudflare"}}},
	})
	out := buf.String()
	for _, want := range []string{"example.com", "CONSENSUS", "2/2", "203.0.113.5", "Google"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestPrintDNSPropDiverged(t *testing.T) {
	var buf bytes.Buffer
	printDNSProp(&buf, dnsprop.Report{
		Name: "example.com", Type: "A", Consensus: false, Propagated: 2, Total: 3,
		Groups: []dnsprop.Group{
			{Values: []string{"203.0.113.9"}, Count: 2, Resolvers: []string{"Google", "Cloudflare"}},
			{Values: []string{"203.0.113.1"}, Count: 1, Resolvers: []string{"Quad9"}},
		},
		Answers: []dnsprop.Answer{{Resolver: dnsprop.Resolver{Name: "Dead"}, Err: "timeout", Group: -1}},
	})
	out := buf.String()
	for _, want := range []string{"DIVERGED", "majority", "differs", "203.0.113.1", "no answer from: Dead"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestRunDNSPropMissingArg(t *testing.T) {
	var o, e bytes.Buffer
	if code := runDNSProp(nil, &o, &e); code != 2 {
		t.Errorf("dnspropagation with no args: got exit %d, want 2", code)
	}
}
