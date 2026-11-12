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

func TestPickAsset(t *testing.T) {
	assets := []asset{
		{Name: "Meridian-v1.2.0-linux-x86_64.AppImage", URL: "l"},
		{Name: "Meridian-v1.2.0-macos-universal.dmg", URL: "m"},
		{Name: "Meridian-v1.2.0-windows-amd64.exe", URL: "w"},
	}
	for _, c := range []struct{ goos, url, label string }{
		{"darwin", "m", "macOS"},
		{"windows", "w", "Windows"},
		{"linux", "l", "Linux"},
	} {
		url, name, label := pickAsset(assets, c.goos)
		if url != c.url || label != c.label || name == "" {
			t.Errorf("pickAsset(%s) = (%q,%q,%q), want url=%q label=%q", c.goos, url, name, label, c.url, c.label)
		}
	}
	// No matching asset → empty URL but still a label.
	if url, _, label := pickAsset(nil, "darwin"); url != "" || label != "macOS" {
		t.Errorf("empty assets: got url=%q label=%q", url, label)
	}
}

func TestCheckLatestParsesDownloadURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v0.2.0","html_url":"u","body":"Notes","assets":[
			{"name":"Meridian-v0.2.0-macos-universal.dmg","browser_download_url":"https://x/m.dmg"},
			{"name":"Meridian-v0.2.0-windows-amd64.exe","browser_download_url":"https://x/w.exe"}]}`))
	}))
	defer srv.Close()
	res, err := checkLatest(context.Background(), "v0.1.0", srv.URL)
	if err != nil {
		t.Fatalf("checkLatest: %v", err)
	}
	// The chosen asset depends on the test host's OS; just assert it matched one
	// of the two offered and set a label.
	if res.OSLabel == "" {
		t.Error("want an OS label")
	}
	if (res.OSLabel == "macOS" && res.DownloadURL != "https://x/m.dmg") ||
		(res.OSLabel == "Windows" && res.DownloadURL != "https://x/w.exe") {
		t.Errorf("download url %q doesn't match OS %q", res.DownloadURL, res.OSLabel)
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
