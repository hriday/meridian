// Package wordlist provides a catalog of password wordlists (pinned URL +
// SHA256) and a checksum-verified downloader. It fetches only when the user
// asks and never leaves an unverified file on disk.
package wordlist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const userAgent = "meridian/0.1 (+https://github.com/hriday/meridian)"

// Entry is a downloadable wordlist pinned to an immutable URL and its SHA256.
type Entry struct {
	Name   string `json:"name"`
	Desc   string `json:"desc"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// seclists pins raw SecLists files to one commit so URL content (and thus the
// checksum) can never drift.
const seclistsCommit = "a1a06ab57c3f4efc450d80738a72c11597aac81a"

func seclistsURL(path string) string {
	return "https://raw.githubusercontent.com/danielmiessler/SecLists/" + seclistsCommit + "/" + path
}

// Catalog returns the built-in wordlists, each with a verified SHA256.
// (rockyou.txt is ~133 MB; add it manually to the wordlists directory.)
func Catalog() []Entry {
	return []Entry{
		{
			Name:   "darkweb2017-top1000.txt",
			Desc:   "SecLists — 1,000 most common leaked passwords",
			URL:    seclistsURL("Passwords/Common-Credentials/darkweb2017_top-1000.txt"),
			SHA256: "552965785108905417702299cef0141415063c2bafb91f3ac25e929997cb6de4",
			Size:   8160,
		},
		{
			Name:   "darkweb2017-top10000.txt",
			Desc:   "SecLists — 10,000 most common leaked passwords",
			URL:    seclistsURL("Passwords/Common-Credentials/darkweb2017_top-10000.txt"),
			SHA256: "479f095b0dfbae07546245441f64a45b7137352617b4caf25b81ea551e67c9af",
			Size:   82603,
		},
		{
			Name:   "10k-most-common.txt",
			Desc:   "SecLists — 10,000 most common passwords",
			URL:    seclistsURL("Passwords/Common-Credentials/10k-most-common.txt"),
			SHA256: "68782d6a4a19a4768d5f15dd66bd534e7a33055cc755411e33f16d18c50fdcce",
			Size:   73026,
		},
	}
}

// Installed lists the wordlist filenames present in dir.
func Installed(dir string) ([]string, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, d := range des {
		if !d.IsDir() {
			names = append(names, d.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// Download fetches e into dir, verifying its SHA256 before keeping it. On a
// checksum mismatch (or any error) the partial file is removed and an error is
// returned — an unverified wordlist is never left on disk.
func Download(ctx context.Context, e Entry, dir string, client *http.Client, onProgress func(done, total int64)) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", e.Name, resp.StatusCode)
	}

	tmp, err := os.CreateTemp(dir, e.Name+".part-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	cleanup := func() { tmp.Close(); os.Remove(tmpName) }

	h := sha256.New()
	total := resp.ContentLength
	var done int64
	buf := make([]byte, 64*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := tmp.Write(buf[:n]); werr != nil {
				cleanup()
				return "", werr
			}
			h.Write(buf[:n])
			done += int64(n)
			if onProgress != nil {
				onProgress(done, total)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			cleanup()
			return "", rerr
		}
	}
	tmp.Close()

	if sum := hex.EncodeToString(h.Sum(nil)); e.SHA256 != "" && !strings.EqualFold(sum, e.SHA256) {
		os.Remove(tmpName)
		return "", fmt.Errorf("checksum mismatch for %s (got %s) — file discarded", e.Name, sum)
	}

	dest := filepath.Join(dir, e.Name)
	if err := os.Rename(tmpName, dest); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	return dest, nil
}
