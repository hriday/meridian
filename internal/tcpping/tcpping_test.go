package tcpping

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeConnector returns a canned duration/err per sequence from the slices.
func fakeConnector(durs []time.Duration, errs []error) Connector {
	i := -1
	return func(ctx context.Context, addr string) (time.Duration, string, error) {
		i++
		var e error
		if i < len(errs) {
			e = errs[i]
		}
		d := time.Duration(0)
		if i < len(durs) {
			d = durs[i]
		}
		if e != nil {
			return d, "", e
		}
		return d, "93.184.216.34:443", nil
	}
}

func TestPingAllSucceed(t *testing.T) {
	conn := fakeConnector(
		[]time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond},
		nil,
	)
	r, err := Ping(context.Background(), conn, "example.com", 443, 3, 0)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if r.Sent != 3 || r.Received != 3 {
		t.Errorf("sent/received = %d/%d", r.Sent, r.Received)
	}
	if r.LossPct != 0 {
		t.Errorf("loss = %v", r.LossPct)
	}
	if r.MinMs != 10 || r.MaxMs != 30 || r.AvgMs != 20 {
		t.Errorf("min/avg/max = %v/%v/%v", r.MinMs, r.AvgMs, r.MaxMs)
	}
	// jitter = mean abs consecutive diff = (|20-10| + |30-20|)/2 = 10
	if r.JitterMs != 10 {
		t.Errorf("jitter = %v", r.JitterMs)
	}
	if r.Address != "93.184.216.34:443" {
		t.Errorf("address = %q", r.Address)
	}
	if len(r.Probes) != 3 || !r.Probes[0].OK {
		t.Errorf("probes = %+v", r.Probes)
	}
}

func TestPingAllFailGivesVerdict(t *testing.T) {
	conn := fakeConnector(nil, []error{
		errors.New("i/o timeout"), errors.New("i/o timeout"),
	})
	r, err := Ping(context.Background(), conn, "closed.example", 9999, 2, 0)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if r.Received != 0 || r.LossPct != 100 {
		t.Errorf("expected total loss, got received=%d loss=%v", r.Received, r.LossPct)
	}
	if r.Verdict == "" {
		t.Error("expected a verdict explaining no connection")
	}
}

func TestPingPartialLoss(t *testing.T) {
	conn := fakeConnector(
		[]time.Duration{15 * time.Millisecond, 0, 25 * time.Millisecond, 0},
		[]error{nil, errors.New("timeout"), nil, errors.New("timeout")},
	)
	r, err := Ping(context.Background(), conn, "flaky.example", 80, 4, 0)
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if r.Sent != 4 || r.Received != 2 {
		t.Errorf("sent/received = %d/%d", r.Sent, r.Received)
	}
	if r.LossPct != 50 {
		t.Errorf("loss = %v", r.LossPct)
	}
	if r.MinMs != 15 || r.MaxMs != 25 {
		t.Errorf("min/max over successes = %v/%v", r.MinMs, r.MaxMs)
	}
}

func TestPingRejectsBadCount(t *testing.T) {
	if _, err := Ping(context.Background(), fakeConnector(nil, nil), "h", 80, 0, 0); err == nil {
		t.Error("expected error for count 0")
	}
}
