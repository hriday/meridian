package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hriday/meridian/internal/tftp"
)

// runTFTP starts a TFTP server (for grabbing configs devices push to it),
// saving each received file to --out and printing it, until Ctrl-C.
func runTFTP(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tftp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("port", 69, "UDP port to bind")
	out := fs.String("out", ".", "directory to save received files")
	asJSON := fs.Bool("json", false, "stream one JSON object per received file (ndjson)")
	pos, ok := parsePositionals(fs, args)
	if !ok {
		return 2
	}
	if len(pos) != 0 {
		fmt.Fprintf(stderr, "tftp takes no arguments (use --port / --out)\n%s", usage)
		return 2
	}

	srv := tftp.New()
	if err := srv.Start(*port); err != nil {
		fmt.Fprintln(stderr, bindHint(err, *port))
		return 1
	}
	defer srv.Stop()
	if _, bound, _ := srv.List(); !*asJSON {
		fmt.Fprintf(stderr, "listening on udp/%d, saving to %s — Ctrl-C to stop\n", bound, *out)
	}

	ctx, stop := signalCtx()
	defer stop()
	enc := json.NewEncoder(stdout)
	ticker := time.NewTicker(collectorPoll)
	defer ticker.Stop()
	seen := map[int]bool{}
	for {
		select {
		case <-ctx.Done():
			return 0
		case <-ticker.C:
			_, _, files := srv.List()
			for _, f := range files {
				if seen[f.ID] {
					continue
				}
				seen[f.ID] = true
				content, _ := srv.Content(f.ID)
				path, err := saveTFTPFile(*out, f.Name, content)
				if err != nil {
					fmt.Fprintf(stderr, "couldn't save %s: %v\n", f.Name, err)
					continue
				}
				if *asJSON {
					_ = enc.Encode(struct {
						tftp.File
						Path string `json:"path"`
					}{f, path})
				} else {
					fmt.Fprintf(stdout, "%s  %-28s %8d bytes  from %-16s → %s\n",
						f.T.Format("15:04:05"), f.Name, f.Size, f.Source, path)
				}
			}
		}
	}
}

// saveTFTPFile writes content to dir, using a sanitized base name.
func saveTFTPFile(dir, name string, content []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	safe := filepath.Base(strings.TrimSpace(name))
	if safe == "" || safe == "." || safe == string(filepath.Separator) {
		safe = "tftp-file"
	}
	path := filepath.Join(dir, safe)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
