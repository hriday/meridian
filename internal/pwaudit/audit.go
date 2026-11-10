package pwaudit

import (
	"context"
	"fmt"
	"sort"
)

// PatternStat counts how many cracked passwords share a pattern label.
type PatternStat struct {
	Pattern string `json:"pattern"`
	Count   int    `json:"count"`
}

// AuditReport is the full result of an audit run. It contains recovered
// plaintext (Results/Reuse) and so is sensitive: keep it in memory or the
// encrypted store only.
type AuditReport struct {
	Total          int           `json:"total"`
	Cracked        int           `json:"cracked"`
	PercentCracked float64       `json:"percentCracked"`
	Results        []Result      `json:"results"`
	Reuse          []ReuseGroup  `json:"reuse"`
	Patterns       []PatternStat `json:"patterns"`
	CrackerUsed    string        `json:"crackerUsed"`
}

// RunAudit cracks the accounts' NT hashes with c against wordlist (+ optional
// rules) and builds the report: per-account results with strength/pattern,
// password reuse, and pattern/summary statistics.
func RunAudit(ctx context.Context, accounts []Account, c Cracker, wordlist string, rules []string, onProgress func(Progress)) (AuditReport, error) {
	info := c.Detect(ctx)
	if !info.Available {
		return AuditReport{}, fmt.Errorf("no cracker available (install hashcat and ensure it is on PATH)")
	}

	// Unique NT hashes to crack.
	seen := map[string]bool{}
	var hashes []string
	for _, a := range accounts {
		if a.NTHash != "" && !seen[a.NTHash] {
			seen[a.NTHash] = true
			hashes = append(hashes, a.NTHash)
		}
	}

	cracked, err := c.Crack(ctx, hashes, wordlist, rules, onProgress)
	if err != nil {
		return AuditReport{}, err
	}

	rep := AuditReport{Total: len(accounts), CrackerUsed: info.Name}
	patternCount := map[string]int{}
	for _, a := range accounts {
		r := Result{Account: a}
		if pt, ok := cracked[a.NTHash]; ok {
			r.Cracked = true
			r.Plaintext = pt
			r.Strength, r.Pattern = Classify(pt)
			rep.Cracked++
			patternCount[r.Pattern]++
		}
		rep.Results = append(rep.Results, r)
	}
	if rep.Total > 0 {
		rep.PercentCracked = float64(rep.Cracked) / float64(rep.Total) * 100
	}
	rep.Reuse = DetectReuse(rep.Results)

	for p, n := range patternCount {
		rep.Patterns = append(rep.Patterns, PatternStat{Pattern: p, Count: n})
	}
	sort.Slice(rep.Patterns, func(i, j int) bool {
		if rep.Patterns[i].Count != rep.Patterns[j].Count {
			return rep.Patterns[i].Count > rep.Patterns[j].Count
		}
		return rep.Patterns[i].Pattern < rep.Patterns[j].Pattern
	})
	return rep, nil
}
