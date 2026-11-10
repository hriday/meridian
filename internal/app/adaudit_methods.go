package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

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
