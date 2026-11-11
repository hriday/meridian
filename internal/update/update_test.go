package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		cur, lat string
		want     bool
	}{
		{"v0.1.0", "v0.2.0", true},
		{"v0.1.0", "v0.1.0", false},
		{"v0.2.0", "v0.1.0", false},
		{"v0.1.0", "v0.1.1", true},
		{"v1.0.0", "v0.9.9", false},
		{"v0.1.0", "v1.0.0", true},
		{"dev", "v0.1.0", true},  // an unversioned build sees any release as newer
		{"0.1.0", "0.2.0", true}, // tolerate a missing "v"
		{"v0.1.0", "", false},    // no latest → not newer
	}
	for _, c := range cases {
		if got := isNewer(c.cur, c.lat); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.cur, c.lat, got, c.want)
		}
	}
}

func TestCheckLatestNewer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		w.Write([]byte(`{"tag_name":"v0.2.0","html_url":"https://github.com/hriday/meridian/releases/tag/v0.2.0","body":"Notes"}`))
	}))
	defer srv.Close()

	res, err := checkLatest(context.Background(), "v0.1.0", srv.URL)
	if err != nil {
		t.Fatalf("checkLatest: %v", err)
	}
	if !res.Newer {
		t.Error("want Newer=true")
	}
	if res.Latest != "v0.2.0" || res.Current != "v0.1.0" {
		t.Errorf("current=%q latest=%q", res.Current, res.Latest)
	}
	if res.URL == "" {
		t.Error("want a release URL")
	}
}

func TestCheckLatestUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v0.1.0","html_url":"u","body":""}`))
	}))
	defer srv.Close()

	res, err := checkLatest(context.Background(), "v0.1.0", srv.URL)
	if err != nil {
		t.Fatalf("checkLatest: %v", err)
	}
	if res.Newer {
		t.Error("want Newer=false when already on the latest")
	}
}

func TestCheckLatestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := checkLatest(context.Background(), "v0.1.0", srv.URL); err == nil {
		t.Error("expected an error on a non-200 response")
	}
}
