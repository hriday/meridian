package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// batchStdin is where batch reads its target list (a seam for tests).
var batchStdin io.Reader = os.Stdin

// runBatch runs a single-target command once per newline-delimited target read
// from stdin, concurrently (bounded) but printing results in input order. With
// --json it emits one JSON object per line (NDJSON) for easy jq piping;
// otherwise it prints a "== target ==" section per result.
//
//	cat hosts.txt | meridian-cli batch cipherscan --json
func runBatch(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "batch needs a command, e.g.  cat hosts.txt | meridian-cli batch cipherscan --json")
		return 2
	}
	cmd := args[0]
	if cmd == "batch" {
		fmt.Fprintln(stderr, "can't batch the batch command")
		return 2
	}
	flags := args[1:]

	var targets []string
	sc := bufio.NewScanner(batchStdin)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		targets = append(targets, t)
	}
	if len(targets) == 0 {
		fmt.Fprintln(stderr, "no targets on stdin (pipe a newline-delimited list, e.g. a hosts file)")
		return 2
	}

	isJSON := false
	for _, f := range flags {
		if f == "--json" || f == "-json" {
			isJSON = true
		}
	}

	type result struct {
		out, err string
		code     int
	}
	results := make([]result, len(targets))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			var out, errb bytes.Buffer
			callArgs := append([]string{cmd, targets[i]}, flags...)
			code := Run(callArgs, &out, &errb)
			results[i] = result{out: out.String(), err: errb.String(), code: code}
		}(i)
	}
	wg.Wait()

	anyOK := false
	for i, t := range targets {
		r := results[i]
		if r.code == 0 {
			anyOK = true
		}
		if isJSON {
			out := strings.TrimRight(r.out, "\n")
			if out == "" {
				out = fmt.Sprintf(`{"target":%q,"ok":false}`, t)
			}
			fmt.Fprintln(stdout, out)
			continue
		}
		if r.code != 0 {
			msg := strings.TrimSpace(r.err)
			if msg == "" {
				msg = strings.TrimSpace(r.out)
			}
			fmt.Fprintf(stdout, "== %s ==  (failed)\n%s\n\n", t, msg)
			continue
		}
		fmt.Fprintf(stdout, "== %s ==\n%s\n", t, strings.TrimRight(r.out, "\n"))
	}
	if !anyOK {
		return 1
	}
	return 0
}
