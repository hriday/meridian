package atlas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, h http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	old := baseURL
	baseURL = srv.URL + "/"
	t.Cleanup(func() { baseURL = old })
}

func TestScheduleSendsKeyAndParsesID(t *testing.T) {
	var gotAuth string
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"measurements":[42]}`))
	})
	id, err := schedule(context.Background(), http.DefaultClient, "testkey", "1.1.1.1", 10)
	require.NoError(t, err)
	require.Equal(t, 42, id)
	require.Equal(t, "Key testkey", gotAuth)
}

func TestScheduleInsufficientCredits(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"error":{"detail":"You do not have enough credits","status":400}}`))
	})
	_, err := schedule(context.Background(), http.DefaultClient, "k", "1.1.1.1", 10)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "credit")
}

func TestScheduleInvalidKey(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"error":{"detail":"The provided API key does not exist","status":403}}`))
	})
	_, err := schedule(context.Background(), http.DefaultClient, "bad", "1.1.1.1", 10)
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "key")
}

func TestResultsParsesPing(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"prb_id":1,"from":"9.9.9.9","min":1.1,"avg":1.5,"max":2.0,"sent":3,"rcvd":3}]`))
	})
	ps, err := results(context.Background(), http.DefaultClient, "k", 1)
	require.NoError(t, err)
	require.Len(t, ps, 1)
	require.Equal(t, 1, ps[0].ID)
	require.Equal(t, "9.9.9.9", ps[0].From)
	require.InDelta(t, 1.5, ps[0].AvgMs, 1e-9)
	require.Equal(t, 3, ps[0].Received)
}

func TestPingPollsUntilResults(t *testing.T) {
	var polls int
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"measurements":[7]}`))
		case strings.Contains(r.URL.Path, "/results/"):
			polls++
			if polls == 1 {
				_, _ = w.Write([]byte(`[]`)) // not reported yet
			} else {
				_, _ = w.Write([]byte(`[{"prb_id":1,"from":"9.9.9.9","min":1,"avg":2,"max":3,"sent":3,"rcvd":3}]`))
			}
		}
	})
	rep, err := Ping(context.Background(), http.DefaultClient, "k", "1.1.1.1", 5, 5*time.Millisecond, 2*time.Second)
	require.NoError(t, err)
	require.Equal(t, 7, rep.MeasurementID)
	require.Len(t, rep.Probes, 1)
	require.Equal(t, 1, rep.Responding)
	require.InDelta(t, 2, rep.AvgMs, 1e-9)
	require.GreaterOrEqual(t, polls, 2)
}

func TestPingTimeoutEmpty(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"measurements":[7]}`))
			return
		}
		_, _ = w.Write([]byte(`[]`)) // never reports
	})
	rep, err := Ping(context.Background(), http.DefaultClient, "k", "1.1.1.1", 5, 5*time.Millisecond, 40*time.Millisecond)
	require.NoError(t, err)
	require.Empty(t, rep.Probes)
	require.Equal(t, 0, rep.Responding)
}
