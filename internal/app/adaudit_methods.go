package app

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/hriday/meridian/internal/hibp"
	"github.com/hriday/meridian/internal/pwaudit"
	"github.com/hriday/meridian/internal/wordlist"
)

func defaultWordlistDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "wordlists"
	}
	return filepath.Join(dir, "meridian", "wordlists")
}

func defaultRulesDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "rules"
	}
	return filepath.Join(dir, "meridian", "rules")
}

func (a *App) emit(event string, data any) {
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, event, data)
	}
}

// PickFile opens a native file picker and returns the chosen path (empty if
// cancelled). Used to select your own wordlist or a hashcat rules file.
func (a *App) PickFile(title string) (string, error) {
	if a.ctx == nil {
		return "", errors.New("no window context for the file dialog")
	}
	return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: title})
}

// ADAuditDetect reports which cracker (hashcat) is available.
func (a *App) ADAuditDetect() pwaudit.CrackerInfo {
	return a.adCracker.Detect(a.lookupCtx())
}

// ADAuditParseText parses pasted NT hashes.
func (a *App) ADAuditParseText(text string) ([]pwaudit.Account, error) {
	return pwaudit.ParseRaw(text)
}

// ADAuditParseFile parses a secretsdump/pwdump file at path.
func (a *App) ADAuditParseFile(path string) ([]pwaudit.Account, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return pwaudit.ParseDump(f)
}

// ADAuditRun cracks the accounts and returns the report, streaming
// `adaudit:progress` events. The report is held for an optional encrypted save.
func (a *App) ADAuditRun(accounts []pwaudit.Account, wordlistPath string, rules []string) (pwaudit.AuditReport, error) {
	rep, err := pwaudit.RunAudit(a.lookupCtx(), accounts, a.adCracker, wordlistPath, rules,
		func(p pwaudit.Progress) { a.emit("adaudit:progress", p) })
	if err != nil {
		return rep, err
	}
	a.adReport = &rep
	return rep, nil
}

// ADAuditCheckPwned checks each audited account's NT hash against Have I Been
// Pwned (Pwned Passwords, NTLM mode). It uses k-anonymity — only the 5-character
// prefix of each hash leaves the machine — and works on every account, including
// ones hashcat couldn't crack, since the NT hash is enough. Returns a map from NT
// hash to the number of breaches it appears in, for the UI to show next to each
// account. This is an explicit, user-initiated network call; it never runs as
// part of the audit, so an offline audit stays offline unless you ask for it.
func (a *App) ADAuditCheckPwned() (map[string]int, error) {
	if a.adReport == nil {
		return nil, errors.New("run an audit first")
	}
	seen := map[string]bool{}
	var hashes []string
	for _, r := range a.adReport.Results {
		h := r.Account.NTHash
		if h == "" || seen[h] {
			continue
		}
		seen[h] = true
		hashes = append(hashes, h)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	return hibp.CheckNTLM(a.lookupCtx(), client, hashes)
}

// ADAuditSave writes the last run's report to path, passphrase-encrypted.
func (a *App) ADAuditSave(path, passphrase string) error {
	if a.adReport == nil {
		return errors.New("no audit to save yet")
	}
	return pwaudit.SaveReport(path, passphrase, *a.adReport)
}

// WordlistCatalog returns the built-in, checksum-pinned wordlists.
func (a *App) WordlistCatalog() []wordlist.Entry { return wordlist.Catalog() }

// WordlistInstalled lists wordlists already downloaded.
func (a *App) WordlistInstalled() ([]string, error) { return wordlist.Installed(a.wordlistDir) }

// WordlistDir is where wordlists are stored.
func (a *App) WordlistDir() string { return a.wordlistDir }

// RulesCatalog returns the built-in, checksum-pinned hashcat rule files.
func (a *App) RulesCatalog() []wordlist.Entry { return wordlist.RulesCatalog() }

// RulesInstalled lists rule files already downloaded.
func (a *App) RulesInstalled() ([]string, error) { return wordlist.Installed(a.rulesDir) }

// RulesDir is where rule files are stored.
func (a *App) RulesDir() string { return a.rulesDir }

// RulesDownload fetches a rules catalog entry by name (checksum-verified),
// streaming `rules:progress` events, and returns the saved path.
func (a *App) RulesDownload(name string) (string, error) {
	for _, e := range wordlist.RulesCatalog() {
		if e.Name == name {
			return wordlist.Download(a.lookupCtx(), e, a.rulesDir, nil,
				func(done, total int64) {
					a.emit("rules:progress", map[string]int64{"done": done, "total": total})
				})
		}
	}
	return "", fmt.Errorf("unknown rule %q", name)
}

// WordlistDownload fetches a catalog entry by name (checksum-verified),
// streaming `wordlist:progress` events, and returns the saved path.
func (a *App) WordlistDownload(name string) (string, error) {
	for _, e := range wordlist.Catalog() {
		if e.Name == name {
			return wordlist.Download(a.lookupCtx(), e, a.wordlistDir, nil,
				func(done, total int64) {
					a.emit("wordlist:progress", map[string]int64{"done": done, "total": total})
				})
		}
	}
	return "", fmt.Errorf("unknown wordlist %q", name)
}
