package atlas

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScheduleTraceSendsTracerouteDef(t *testing.T) {
	var gotBody, gotAuth string
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"measurements":[55]}`))
	})
	id, err := scheduleTrace(context.Background(), http.DefaultClient, "testkey", "1.1.1.1", 5)
	require.NoError(t, err)
	require.Equal(t, 55, id)
	require.Equal(t, "Key testkey", gotAuth)
	require.Contains(t, gotBody, `"type":"traceroute"`)
}

// a traceroute result: hop 1 responds, hop 2 all timeout, hop 3 is the target.
const traceReached = `[{"prb_id":101,"from":"2.2.2.2","dst_addr":"1.1.1.1","result":[` +
	`{"hop":1,"result":[{"from":"10.0.0.1","rtt":0.7},{"from":"10.0.0.1","rtt":0.5},{"from":"10.0.0.1","rtt":0.9}]},` +
	`{"hop":2,"result":[{"x":"*"},{"x":"*"},{"x":"*"}]},` +
	`{"hop":3,"result":[{"from":"1.1.1.1","rtt":10.1}]}]}]`

func TestTraceResultsParsesHops(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(traceReached))
	})
	ps, err := traceResults(context.Background(), http.DefaultClient, "k", 1)
	require.NoError(t, err)
	require.Len(t, ps, 1)
	p := ps[0]
	require.Equal(t, 101, p.ID)
	require.Equal(t, "2.2.2.2", p.From)
	require.Equal(t, "1.1.1.1", p.Dest)
	require.True(t, p.Reached)
	require.Len(t, p.Hops, 3)

	require.Equal(t, 1, p.Hops[0].Number)
	require.Equal(t, "10.0.0.1", p.Hops[0].From)
	require.False(t, p.Hops[0].Timeout)
	require.InDelta(t, 0.5, p.Hops[0].RttMs, 1e-9) // min of the three replies

	require.True(t, p.Hops[1].Timeout)
	require.Equal(t, "", p.Hops[1].From)

	require.Equal(t, "1.1.1.1", p.Hops[2].From)
	require.InDelta(t, 10.1, p.Hops[2].RttMs, 1e-9)
}

func TestTraceResultsNotReached(t *testing.T) {
	body := `[{"prb_id":5,"from":"2.2.2.2","dst_addr":"1.1.1.1","result":[` +
		`{"hop":1,"result":[{"from":"10.0.0.1","rtt":0.7}]},` +
		`{"hop":2,"result":[{"x":"*"},{"x":"*"},{"x":"*"}]}]}]`
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})
	ps, err := traceResults(context.Background(), http.DefaultClient, "k", 1)
	require.NoError(t, err)
	require.Len(t, ps, 1)
	require.False(t, ps[0].Reached)
}

func TestTracePollsUntilResults(t *testing.T) {
	var polls int
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"measurements":[7]}`))
			return
		}
		polls++
		if polls == 1 {
			_, _ = w.Write([]byte(`[]`))
		} else {
			_, _ = w.Write([]byte(traceReached))
		}
	})
	rep, err := Trace(context.Background(), http.DefaultClient, "k", "1.1.1.1", 1, 5*time.Millisecond, 2*time.Second)
	require.NoError(t, err)
	require.Equal(t, 7, rep.MeasurementID)
	require.Equal(t, 1, rep.Requested)
	require.Len(t, rep.Probes, 1)
	require.Equal(t, 1, rep.Responding)
	require.Equal(t, 1, rep.Reached)
	require.GreaterOrEqual(t, polls, 2)
}

// a 2-probe result, both reaching the target.
const traceTwo = `[` +
	`{"prb_id":101,"from":"2.2.2.2","dst_addr":"1.1.1.1","result":[{"hop":1,"result":[{"from":"1.1.1.1","rtt":1.0}]}]},` +
	`{"prb_id":102,"from":"3.3.3.3","dst_addr":"1.1.1.1","result":[{"hop":1,"result":[{"from":"1.1.1.1","rtt":2.0}]}]}]`

func TestTraceWaitsForRequested(t *testing.T) {
	var polls int
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"measurements":[7]}`))
			return
		}
		polls++
		if polls < 3 {
			_, _ = w.Write([]byte(traceReached)) // only 1 probe reported yet
		} else {
			_, _ = w.Write([]byte(traceTwo)) // both requested probes in
		}
	})
	rep, err := Trace(context.Background(), http.DefaultClient, "k", "1.1.1.1", 2, 5*time.Millisecond, 2*time.Second)
	require.NoError(t, err)
	require.Equal(t, 2, rep.Requested)
	require.Len(t, rep.Probes, 2, "should wait for the 2nd requested probe, not return on the 1st")
	require.Equal(t, 2, rep.Responding)
}

func TestTraceTimeoutEmpty(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"measurements":[7]}`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	})
	rep, err := Trace(context.Background(), http.DefaultClient, "k", "1.1.1.1", 5, 5*time.Millisecond, 40*time.Millisecond)
	require.NoError(t, err)
	require.Empty(t, rep.Probes)
	require.Equal(t, 0, rep.Responding)
}

func TestScheduleTraceCreditsError(t *testing.T) {
	// shared scheduleError path still maps the nested credits message.
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"detail":"There was a problem with your request","errors":[{"detail":"You do not have enough credit"}]}}`))
	})
	_, err := scheduleTrace(context.Background(), http.DefaultClient, "k", "1.1.1.1", 5)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "credit")
}
