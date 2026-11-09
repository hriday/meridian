# Meridian IP-space Records Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Record your own IP space as a nested tree of blocks/subnets/addresses in a local, passphrase-encrypted, append-only store — with overlap guards, actor-stamped history, undo, and CSV import/export.

**Architecture:** A pure-Go `internal/ipspace` engine whose source of truth is an **append-only event log** (add/edit/delete, each stamped with actor + time); the tree state is derived by replaying the log. The log is persisted as a single **passphrase-encrypted blob** (argon2id key derivation + NaCl secretbox — no cgo). The Wails controller holds one open store and exposes bound methods; a minimal CLI reads the store. The Svelte tool renders unlock → tree → detail/actions → add form → change log.

**Tech Stack:** Go 1.23; **`golang.org/x/crypto`** (argon2, nacl/secretbox — already in the module graph, promoted to direct); `net/netip` for prefix math; stdlib `encoding/json`, `encoding/csv`; Wails v2.10.1 + Svelte/TS.

**Spec:** docs/superpowers/specs/2026-11-08-meridian-design.md (§4.4 IP-space records, decision 71 "team-aware from the start"). Scope + storage approved by owner 2026-11-09 against the clickable mockup: an **encrypted append-only log** (the change log IS the store), **encrypt at rest with a passphrase**; nested tree, overlap/nest guard on add, actor + conflict metadata on every event, undo, CSV, Subnet/Lookup pivots from a record. Sync (git/S3/server) and conflict-resolution UI are deferred ("soon").

## Global Constraints

- Go **1.23**. New direct dependency: **`golang.org/x/crypto`** (already indirect; promote and pin the resolved version, do not let it bump the `go` directive — same discipline as the miekg/dns and x/net pins).
- **No cgo.** Encryption is pure Go: argon2id (with a stored random salt) derives a 32-byte key; NaCl secretbox (random 24-byte nonce) seals the serialized log. On-disk format: `salt | nonce | ciphertext`.
- **Actor on every event**, from day one (team/conflict metadata the design requires): actor = `MERIDIAN_ACTOR` env, else the OS username, else `"unknown"`.
- A wrong passphrase, or a tampered/corrupt file, **fails to open with a clear error** — never a panic and never silently loads partial/garbage data.
- Store location: the OS app-data dir (`os.UserConfigDir()/Meridian/ipspace.enc`); the path is injectable so tests use a temp file.
- Prefix math via `net/netip`; a single address is stored as a host prefix (`/32`/`/128`).
- UI stays non-blocking; themes (Auto/Light/Dark/Terminal), Geist fonts, collapsible sidebar intact; the tool follows the plain-language / Nielsen standard ([[ui-principles-nielsen]]). No passphrase recovery — the UI states this plainly.

## Review Focus

- **Wrong passphrase on unlock** → a clear "couldn't unlock (wrong passphrase?)" error; no crash, no data loss, store stays locked. *Test in Task 3.*
- **Overlapping / duplicate add** → an exact duplicate is rejected; a block that nests inside an existing one is auto-parented; a block that would *contain* existing blocks is rejected with a clear message. The tree never ends up with a record overlapping a sibling. *Test in Task 2.*
- **Undo edge cases** → undo on empty history is a no-op (clear result); undo of an add that has children removes the subtree-root consistently (children were added by later events, so they remain only if their own add events remain). *Test in Task 2.*
- **Corrupt / truncated / tampered store file** → `Load` returns an error, no panic, no partial state. *Test in Task 3.*
- **CSV with malformed rows / bad CIDRs** → bad rows are skipped and counted in a report; valid rows import; existing data is untouched. *Test in Task 4.*

---

## Task 1: Record / event model + tree replay (`internal/ipspace`)

**Files:**
- Create: `internal/ipspace/model.go`
- Test: `internal/ipspace/model_test.go`

**Interfaces:**
- Produces: `type Status string` (`active`/`reserved`/`deprecated`); `type Record struct { ID, Prefix, Name string; Status Status; Tags []string; Notes, ParentID, CreatedBy, ChangedBy string; CreatedAt, ChangedAt time.Time }`; `type EventType string` (`add`/`edit`/`delete`); `type Event struct { ID string; Type EventType; Actor string; At time.Time; Record Record }`; `type TreeNode struct { Record Record; Children []TreeNode }`; `func replay(events []Event) map[string]Record` (add inserts + stamps CreatedBy/At; edit updates mutable fields + stamps ChangedBy/At, preserving CreatedBy/At; delete removes by ID); `func buildTree(records map[string]Record) []TreeNode` (roots = ParentID "", children nested, each list sorted by prefix numerically).
- Consumes: nothing.

- [ ] **Step 1: Write the failing test** `TestReplayAddEditDelete` — three events (add A, edit A's name, add B child of A) → `replay` yields A (new name, CreatedBy from the add, ChangedBy from the edit) and B; a delete event for B removes it.
- [ ] **Step 2: Add** `TestBuildTreeNesting` (records with ParentID form the expected parent→child tree, sorted) and `TestReplayDeleteUnknownIsNoop`.
- [ ] **Step 3: Run the tests, verify they fail.** Run: `go test ./internal/ipspace/ -run 'TestReplay|TestBuildTree' -v` — Expected: FAIL, undefined.
- [ ] **Step 4: Implement** the types, `replay`, and `buildTree` (numeric prefix sort via `netip.Prefix` compare).
- [ ] **Step 5: Run the tests, verify they pass.**
- [ ] **Step 6: Commit** (`feat(ipspace): record/event model and tree replay`).

## Task 2: Store — add/edit/delete/undo with overlap guard

**Files:**
- Create: `internal/ipspace/store.go`
- Test: `internal/ipspace/store_test.go`

**Interfaces:**
- Consumes: Task 1 types.
- Produces: `type Store struct { ... }`; `func NewStore(actor string) *Store`; `func FromEvents(actor string, events []Event) *Store`; `func (*Store) Events() []Event`; `func (*Store) Records() []Record`; `func (*Store) Tree() []TreeNode`; `func (*Store) History() []Event`; `func (*Store) Add(prefix, name string, status Status, tags []string, notes string) (Record, error)` (parses+normalizes the prefix, runs the overlap guard to find a parent or reject, appends an `add` event); `func (*Store) Edit(id, name string, status Status, tags []string, notes string) error`; `func (*Store) Delete(id string) error`; `func (*Store) Undo() error`; and the pure helper `func findParent(existing []Record, p netip.Prefix) (parentID string, err error)` (equal→duplicate error; p inside an existing→most-specific parent; p contains an existing→"contains existing blocks" error).

- [ ] **Step 1: Write the failing test** `TestAddNestsAndGuardsOverlap` — add `10.0.0.0/8`, then `10.1.0.0/16` (parent = the /8), then `10.1.0.0/16` again (duplicate → error), then `10.0.0.0/8` again (duplicate → error), then `10.0.0.0/7` which contains the /8 (→ "contains existing" error). Assert the parent wiring and that errors don't mutate state.
- [ ] **Step 2: Add** `TestAddHostAddress` (`10.1.10.5` stored as `/32`, parented under the enclosing block), `TestEditAndActorStamp` (edit updates fields + ChangedBy = actor), `TestDeleteRemoves`, `TestUndoRevertsLastAdd` (add then Undo → record gone, history shows the add is dropped), and `TestUndoEmptyIsError`.
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL, undefined.
- [ ] **Step 4: Implement** `Store` (holds events + derived `replay` cache, an actor), `findParent` (using `netip.Prefix.Overlaps`/`Bits`/`Masked`), and the operations (each appends an event with a fresh ID + actor + `time.Now().UTC()`; Undo drops the last event and re-derives).
- [ ] **Step 5: Run the tests, verify they pass.**
- [ ] **Step 6: Commit** (`feat(ipspace): store with overlap-guarded add, edit, delete, undo`).

## Task 3: Encrypted persistence (argon2 + secretbox)

**Files:**
- Create: `internal/ipspace/crypto.go`
- Test: `internal/ipspace/crypto_test.go`

**Interfaces:**
- Consumes: Task 1 `Event`.
- Produces: `func Save(path, passphrase string, events []Event) error` (serialize events to JSON, derive key via argon2id with a fresh salt, secretbox-seal, write `salt|nonce|ciphertext` atomically — temp file + rename); `func Load(path, passphrase string) ([]Event, error)` (read, split, derive key, open; a bad passphrase or any tamper → a clear error); `func Exists(path string) bool`; `func Create(path, passphrase string) error` (Save an empty event list).

- [ ] **Step 1: Write the failing test** `TestSaveLoadRoundTrip` — `Save` three events to a temp path, `Load` with the same passphrase → identical events.
- [ ] **Step 2: Add** `TestLoadWrongPassphrase` (Load with a different passphrase → error, no panic), `TestLoadTamperedFile` (flip a ciphertext byte → error), and `TestLoadMissingFile` (→ error, `Exists` false).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL, undefined.
- [ ] **Step 4: Implement** `Save`/`Load`/`Exists`/`Create` with argon2id (fixed sane params, 16-byte salt) + secretbox (24-byte nonce), atomic write via `os.CreateTemp`+`os.Rename`.
- [ ] **Step 5: Run the tests, verify they pass.**
- [ ] **Step 6: Commit** (`feat(ipspace): passphrase-encrypted append-only persistence`).

## Task 4: CSV import / export

**Files:**
- Create: `internal/ipspace/csv.go`
- Test: `internal/ipspace/csv_test.go`

**Interfaces:**
- Consumes: Task 2 `Store`.
- Produces: `type ImportReport struct { Added, Skipped int; Errors []string }`; `func (*Store) ImportCSV(r io.Reader) (ImportReport, error)` (header `prefix,name,status,tags,notes`; each valid row → `Add`; a bad CIDR or an overlap conflict is skipped and recorded, never aborts the batch); `func (*Store) ExportCSV(w io.Writer) error` (writes the same columns for every record, prefix-sorted).

- [ ] **Step 1: Write the failing test** `TestImportCSVSkipsBadRows` — a CSV with two good rows, one bad CIDR, one duplicate → `ImportReport{Added:2, Skipped:2}` with two error strings, and the two good records present.
- [ ] **Step 2: Add** `TestExportCSVRoundTrips` (Add records, Export, re-Import into a fresh store → same prefixes/names).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL, undefined.
- [ ] **Step 4: Implement** `ImportCSV`/`ExportCSV` using `encoding/csv`, tolerating a missing/extra trailing column and quoting tags as `a|b`.
- [ ] **Step 5: Run the tests, verify they pass.**
- [ ] **Step 6: Commit** (`feat(ipspace): CSV import/export`).

## Task 5: App bound methods + minimal CLI

**Files:**
- Create: `internal/app/ipspace_methods.go`
- Modify: `internal/cli/cli.go` (add `ipspace list` / `ipspace export`)
- Test: `internal/app/ipspace_methods_test.go`, `internal/cli/cli_ipspace_test.go`

**Interfaces:**
- Consumes: `internal/ipspace`.
- Produces: App methods `IPSpaceStatus() {Exists, Unlocked bool}`, `IPSpaceCreate(passphrase) error`, `IPSpaceUnlock(passphrase) error`, `IPSpaceTree() []ipspace.TreeNode`, `IPSpaceAdd(prefix,name,status,tags []string→string? ...) (ipspace.Record, error)`, `IPSpaceEdit(...)`, `IPSpaceDelete(id) error`, `IPSpaceUndo() error`, `IPSpaceHistory() []ipspace.Event`, `IPSpaceImportCSV(text) (ipspace.ImportReport,error)`, `IPSpaceExportCSV() (string,error)` — each mutating method persists via `ipspace.Save` to the app's store path (an injectable field, default `os.UserConfigDir()`). CLI `meridian-cli ipspace list` / `ipspace export` reading `--file` + `--passphrase`/`MERIDIAN_PASSPHRASE`.

- [ ] **Step 1: Write the failing test** `TestAppIPSpaceCreateAddUnlock` — point the app's store path at a temp file; `IPSpaceCreate("pw")`, `IPSpaceAdd("10.0.0.0/8","Corp",…)`, then a fresh App `IPSpaceUnlock("pw")` and `IPSpaceTree()` shows the block (persistence round-trips through the encrypted file).
- [ ] **Step 2: Add** `TestAppIPSpaceWrongPassphrase` (Unlock with a bad passphrase → error, `IPSpaceStatus().Unlocked == false`) and a CLI test `TestRunIPSpaceExport` (temp store seeded, `ipspace export --file … --passphrase …` prints CSV).
- [ ] **Step 3: Run the tests, verify they fail.** Expected: FAIL.
- [ ] **Step 4: Implement** the methods (App holds `*ipspace.Store`, path, unlocked flag; mutating ops Save after applying) and the CLI command.
- [ ] **Step 5: Run** `go test ./...`.
- [ ] **Step 6: Commit** (`feat(ipspace): app methods and CLI`).

## Task 6: Frontend — IP space tool

**Files:**
- Create: `frontend/src/lib/tools/IpSpace.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (activate `ipspace`; render it)

**Interfaces:**
- Consumes: the bound IPSpace methods and the `ipspace.TreeNode`/`Record`/`Event`/`ImportReport` models.

- [ ] **Step 1:** Build the **unlock gate** — on mount call `IPSpaceStatus()`; if no store, show "Create an encrypted IP-space store" (passphrase + confirm, with the "no recovery" note); if locked, show an unlock passphrase field; errors shown inline.
- [ ] **Step 2:** Build the **tree + detail**: the nested tree (expand/collapse, utilization, status pill), a selected-record detail panel with the **action bar** (Edit · Add child · Subnet calc ↗ · Look up ↗ · Delete — the two pivots switch to those tools' tabs), and the **change log** with **Undo**.
- [ ] **Step 3:** Build the **Add block form** with live CIDR validation + the overlap/nest hint (calls `IPSpaceAdd`, shows the returned parent), plus **Import/Export CSV** (Export offers the text; Import takes pasted CSV and shows the `ImportReport`).
- [ ] **Step 4:** In `Workbench.svelte`, set the `ipspace` tool `ready: true` and render `IpSpace`.
- [ ] **Step 5: Build** (`~/go/bin/wails build`) and launch; create a store, add a block, add an overlapping block (see the guard), undo, export CSV (manual visual check).
- [ ] **Step 6: Run** `go test ./...`.
- [ ] **Step 7: Commit** (`feat(ui): IP space records tool`).

---

## Self-review notes
- Spec coverage: model/tree = Task 1; overlap-guarded ops + undo + actor = Task 2; encrypted append-only persistence = Task 3; CSV = Task 4; app/CLI = Task 5; UI = Task 6.
- Type consistency: `Record`/`Event`/`TreeNode`/`Status` (Task 1), `Store` (Task 2), `ImportReport` (Task 4) are consumed verbatim downstream.
- Review Focus: wrong passphrase + tamper (Task 3), overlap/duplicate/contains (Task 2), undo edges (Task 2), CSV bad rows (Task 4) each have named tests.
- Proportion: the plan fixes the event model, the overlap semantics, the crypto construction, the store path seam, and the dependency; implementations are left to the engineer.
