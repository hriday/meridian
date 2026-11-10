package pwaudit

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CrackerInfo describes an available external cracker.
type CrackerInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Path      string `json:"path"`
	Available bool   `json:"available"`
}

// Progress is a cracking-progress snapshot parsed from the cracker's status output.
type Progress struct {
	Recovered int     `json:"recovered"`
	Total     int     `json:"total"`
	Percent   float64 `json:"percent"`
}

// Cracker runs the actual password cracking. The engine depends only on this
// interface, so tests inject a fake and never touch a real cracker.
type Cracker interface {
	Detect(ctx context.Context) CrackerInfo
	// Crack returns a map of NT hash (lowercase) -> recovered plaintext.
	Crack(ctx context.Context, ntHashes []string, wordlist string, rules []string, onProgress func(Progress)) (map[string]string, error)
}

// HashcatCracker drives hashcat (NTLM mode 1000) through injectable seams.
// Run is for quick calls (--version, --show); Stream runs the crack and
// delivers each status line live as hashcat emits it.
type HashcatCracker struct {
	LookPath func(string) (string, error)
	Run      func(ctx context.Context, name string, args ...string) ([]byte, error)
	Stream   func(ctx context.Context, name string, args []string, onLine func(string)) error
}

// NewHashcatCracker returns a HashcatCracker wired to the real system.
func NewHashcatCracker() *HashcatCracker {
	return &HashcatCracker{
		LookPath: exec.LookPath,
		Run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).CombinedOutput()
		},
		Stream: func(ctx context.Context, name string, args []string, onLine func(string)) error {
			cmd := exec.CommandContext(ctx, name, args...)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				return err
			}
			cmd.Stderr = cmd.Stdout
			if err := cmd.Start(); err != nil {
				return err
			}
			sc := bufio.NewScanner(stdout)
			sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for sc.Scan() {
				onLine(sc.Text())
			}
			_ = cmd.Wait() // hashcat exits non-zero on partial cracks; --show is the source of truth
			return nil
		},
	}
}

func (c *HashcatCracker) Detect(ctx context.Context) CrackerInfo {
	path, err := c.LookPath("hashcat")
	if err != nil {
		return CrackerInfo{Name: "hashcat", Available: false}
	}
	out, _ := c.Run(ctx, path, "--version")
	return CrackerInfo{
		Name:      "hashcat",
		Version:   strings.TrimSpace(string(out)),
		Path:      path,
		Available: true,
	}
}

func (c *HashcatCracker) Crack(ctx context.Context, ntHashes []string, wordlist string, rules []string, onProgress func(Progress)) (map[string]string, error) {
	path, err := c.LookPath("hashcat")
	if err != nil {
		return nil, fmt.Errorf("hashcat not found on PATH: %w", err)
	}

	hashFile, err := os.CreateTemp("", "meridian-hashes-*.txt")
	if err != nil {
		return nil, err
	}
	defer os.Remove(hashFile.Name())
	for _, h := range ntHashes {
		fmt.Fprintln(hashFile, h)
	}
	hashFile.Close()

	pot, err := os.CreateTemp("", "meridian-pot-*.txt")
	if err != nil {
		return nil, err
	}
	pot.Close()
	defer os.Remove(pot.Name())

	args := []string{"-m", "1000", "-a", "0", hashFile.Name(), wordlist,
		"--potfile-path", pot.Name(), "--quiet", "--status", "--status-json"}
	for _, r := range rules {
		args = append(args, "--rules-file", r)
	}
	// Stream the crack: report each status line live as hashcat emits it.
	if err := c.Stream(ctx, path, args, func(line string) {
		if onProgress == nil {
			return
		}
		if p, ok := parseStatusLine(line); ok {
			onProgress(p)
		}
	}); err != nil {
		return nil, err
	}

	showOut, err := c.Run(ctx, path, "-m", "1000", "--show", "--potfile-path", pot.Name(), "--quiet", hashFile.Name())
	if err != nil {
		// --show returns non-zero (exit 1) when nothing is cracked; that's not an error.
		if len(bytes.TrimSpace(showOut)) == 0 {
			return map[string]string{}, nil
		}
	}
	return parseShow(showOut), nil
}

// parseShow reads hashcat --show lines "nthash:plaintext" into a map.
func parseShow(out []byte) map[string]string {
	cracked := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.IndexByte(line, ':'); i > 0 {
			cracked[strings.ToLower(line[:i])] = line[i+1:]
		}
	}
	return cracked
}

// parseStatusLine parses one hashcat --status-json line into a Progress snapshot.
func parseStatusLine(line string) (Progress, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "{") {
		return Progress{}, false
	}
	var s struct {
		Recovered []int `json:"recovered_hashes"`
		Progress  []int `json:"progress"`
	}
	if json.Unmarshal([]byte(line), &s) != nil {
		return Progress{}, false
	}
	p := Progress{}
	if len(s.Recovered) == 2 {
		p.Recovered, p.Total = s.Recovered[0], s.Recovered[1]
	}
	if len(s.Progress) == 2 && s.Progress[1] > 0 {
		p.Percent = float64(s.Progress[0]) / float64(s.Progress[1]) * 100
	}
	return p, true
}
