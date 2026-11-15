package app

import (
	"fmt"
	"strings"

	"github.com/hriday/meridian/internal/ipspace"
)

// IPSpaceState reports whether a store exists on disk and whether it's unlocked.
type IPSpaceState struct {
	Exists   bool `json:"exists"`
	Unlocked bool `json:"unlocked"`
}

// IPSpaceStatus reports the active inventory's existence and unlocked state.
func (a *App) IPSpaceStatus() IPSpaceState {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	return IPSpaceState{Exists: a.activeID != "" && ipspace.Exists(a.ipPath), Unlocked: a.ipUnlocked}
}

// IPSpaceUnlock decrypts the active inventory's store with the passphrase.
func (a *App) IPSpaceUnlock(passphrase string) error {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	a.ensureInv()
	if a.activeID == "" {
		return fmt.Errorf("no inventory is selected")
	}
	events, err := ipspace.Load(a.ipPath, passphrase)
	if err != nil {
		return err
	}
	a.ipStore = ipspace.FromEvents(ipspace.Actor(), events)
	a.ipPass = passphrase
	a.ipUnlocked = true
	a.ipSyncCfgLoaded = false
	return nil
}

func (a *App) requireUnlocked() error {
	if a.ipStore == nil || !a.ipUnlocked {
		return fmt.Errorf("the IP-space store is locked")
	}
	return nil
}

func (a *App) ipSave() error {
	return ipspace.Save(a.ipPath, a.ipPass, a.ipStore.Events())
}

// IPSpaceTree returns the nested record tree (empty when locked).
func (a *App) IPSpaceTree() []ipspace.TreeNode {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if a.requireUnlocked() != nil {
		return nil
	}
	return a.ipStore.Tree()
}

// IPSpaceHistory returns the change log.
func (a *App) IPSpaceHistory() []ipspace.Event {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if a.requireUnlocked() != nil {
		return nil
	}
	return a.ipStore.History()
}

// IPSpaceAdd records a new block or address and persists.
func (a *App) IPSpaceAdd(prefix, name, status string, tags []string, notes string) (ipspace.Record, error) {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if err := a.requireUnlocked(); err != nil {
		return ipspace.Record{}, err
	}
	rec, err := a.ipStore.Add(prefix, name, ipspace.Status(status), tags, notes)
	if err != nil {
		return ipspace.Record{}, err
	}
	return rec, a.ipSave()
}

// IPSpaceCarve subdivides the block with id into equal child subnets of
// newPrefixLen (each with the given status), skipping any that already exist,
// and returns the records it created.
func (a *App) IPSpaceCarve(id string, newPrefixLen int, status string) ([]ipspace.Record, error) {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if err := a.requireUnlocked(); err != nil {
		return nil, err
	}
	added, err := a.ipStore.Carve(id, newPrefixLen, ipspace.Status(status))
	if err != nil {
		return nil, err
	}
	return added, a.ipSave()
}

// IPSpaceEdit updates a record and persists.
func (a *App) IPSpaceEdit(id, name, status string, tags []string, notes string) error {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if err := a.ipStore.Edit(id, name, ipspace.Status(status), tags, notes); err != nil {
		return err
	}
	return a.ipSave()
}

// IPSpaceDelete removes a record and persists.
func (a *App) IPSpaceDelete(id string) error {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if err := a.ipStore.Delete(id); err != nil {
		return err
	}
	return a.ipSave()
}

// IPSpaceUndo reverts the last change and persists.
func (a *App) IPSpaceUndo() error {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if err := a.ipStore.Undo(); err != nil {
		return err
	}
	return a.ipSave()
}

// IPSpaceImportCSV imports records from CSV text and persists.
func (a *App) IPSpaceImportCSV(text string) (ipspace.ImportReport, error) {
	return a.IPSpaceImport("csv", text)
}

// IPSpaceImport imports records from text in the given format ("csv",
// "netbox" YAML, or "terraform" HCL) and persists.
func (a *App) IPSpaceImport(format, text string) (ipspace.ImportReport, error) {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if err := a.requireUnlocked(); err != nil {
		return ipspace.ImportReport{}, err
	}
	r := strings.NewReader(text)
	var rep ipspace.ImportReport
	var err error
	switch strings.ToLower(format) {
	case "", "csv":
		rep, err = a.ipStore.ImportCSV(r)
	case "netbox", "yaml":
		rep, err = a.ipStore.ImportNetBoxYAML(r)
	case "terraform", "tf":
		rep, err = a.ipStore.ImportTerraform(r)
	default:
		return ipspace.ImportReport{}, fmt.Errorf("unknown import format %q", format)
	}
	if err != nil {
		return rep, err
	}
	return rep, a.ipSave()
}

// IPSpaceExportCSV returns the records as CSV text.
func (a *App) IPSpaceExportCSV() (string, error) { return a.IPSpaceExport("csv") }

// IPSpaceExport returns the records in the given format ("csv", "netbox" YAML,
// or "terraform" HCL).
func (a *App) IPSpaceExport(format string) (string, error) {
	a.ipMu.Lock()
	defer a.ipMu.Unlock()
	if err := a.requireUnlocked(); err != nil {
		return "", err
	}
	var b strings.Builder
	var err error
	switch strings.ToLower(format) {
	case "", "csv":
		err = a.ipStore.ExportCSV(&b)
	case "netbox", "yaml":
		err = a.ipStore.ExportNetBoxYAML(&b)
	case "terraform", "tf":
		err = a.ipStore.ExportTerraform(&b)
	default:
		return "", fmt.Errorf("unknown export format %q", format)
	}
	if err != nil {
		return "", err
	}
	return b.String(), nil
}
