# AD password audit (security slice 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline) or superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A defensive AD password-audit tool — import your own NTLM hashes, crack them via a detected hashcat/John, and get a report of weak passwords (plaintext, pattern, strength, reuse) — plus the wordlist manager it needs. Local only, results optionally encrypted.

**Architecture:** Pure engine in `internal/pwaudit` (parsing + analysis + orchestration behind a `Cracker` seam), `internal/wordlist` (catalog + checksum-verified downloader behind an HTTP seam), `internal/securebox` (generic argon2id+secretbox seal/open for the optional encrypted save). Thin CLI + App bindings; a `AdAudit.svelte` tool. The cracker and HTTP are injected interfaces so everything is unit-tested without real hashcat or network.

**Tech Stack:** Go 1.23 stdlib (`os/exec`, `bufio`, `crypto/sha256`, `net/http`, `encoding/hex`), `golang.org/x/crypto` (argon2, nacl/secretbox — already a dep); Wails + Svelte.

**Spec:** `docs/superpowers/specs/2026-11-09-meridian-adaudit-slice1-design.md`

## Global Constraints
- Go directive **1.23**; no new third-party deps (x/crypto already present).
- **Never** log/print/transmit recovered plaintext except into the in-memory report and the explicit encrypted save. No plaintext to disk unencrypted.
- Cracker and HTTP are injected seams; tests use fakes, never real hashcat/network.
- Additive only; do not modify existing engine packages (reuse, don't edit, `ipspace`).
- Defensive framing: the UI shows a one-line "authorized self-audit of your own directory; stays on this machine" note.

## Review Focus
- **Malformed dump lines** (short fields, blank, comments, `DOMAIN\user`, machine `name$`, LM-only, history entries) → parser skips/handles without panicking, extracts the NT hash from the right field. (Task 1 tests.)
- **Password reuse incl. edge cases** — multiple accounts with the same NT hash group together; the empty-password NT hash (`31d6cfe0d16ae931b73c59d7e0c089c0`) is flagged. (Task 2 tests.)
- **Cracker absent / returns nothing** — `Detect` reports unavailable with guidance; a run with no cracker errors cleanly; a crack that recovers 0 yields a valid all-uncracked report. (Task 3/4 tests.)
- **Checksum mismatch on wordlist download** — aborts, deletes the partial file, returns an error; never leaves a corrupt/partial wordlist in place. (Task 5 tests.)
- **Wrong passphrase on encrypted load** — returns an error, never partial plaintext; round-trip with the right passphrase recovers the report. (Task 6 tests.)

---

### Task 1: Hash input parsing
**Files:** Create `internal/pwaudit/parse.go`, `internal/pwaudit/parse_test.go`
**Produces:**
```go
type Account struct { Name, NTHash, RID string; Machine bool }
func ParseDump(r io.Reader) ([]Account, error)   // pwdump/secretsdump: DOMAIN\user:rid:lm:nt:::
func ParseRaw(text string) ([]Account, error)    // lines of 32-hex NT, or user:hash
```
- [ ] **Step 1: failing tests** — `TestParseDump` (a 4-line fixture: a normal user, a `DOMAIN\user`, a machine `WS01$`, a blank/comment line → 3 accounts, machine flagged, NT hash = 4th colon field lowercased); `TestParseRaw` (bare 32-hex line, a `user:hash` line, a non-hex line skipped). NT hashes validated as 32 hex chars.
- [ ] **Step 2: run, verify fail.**
- [ ] **Step 3: implement** — split on `:`, NT hash is field index 3 (0-based), name is field 0 (strip `DOMAIN\`), `Machine = strings.HasSuffix(name,"$")`; skip lines without a valid 32-hex NT field.
- [ ] **Step 4: run, verify pass.**
- [ ] **Step 5: commit** `feat(pwaudit): parse NTLM dump and raw hash input`.

### Task 2: Pattern / strength / reuse analysis
**Files:** Create `internal/pwaudit/analyze.go`, `analyze_test.go`
**Produces:**
```go
type Result struct { Account Account; Cracked bool; Plaintext, Strength, Pattern string }
type ReuseGroup struct { NTHash, Plaintext string; Accounts []string }
func Classify(plaintext string) (strength, pattern string)
func DetectReuse(results []Result) []ReuseGroup
```
- [ ] **Step 1: failing tests** — `TestClassify` cases: `Summer2024!`→pattern "season+year", `qwerty`→"keyboard walk", `12345678`→"all digits", `Password1`→"word+digit", a long random string→strength "strong"/pattern "none"; `TestDetectReuse` groups two accounts sharing an NT hash and labels the empty-password hash `31d6cfe0d16ae931b73c59d7e0c089c0` as "(empty password)".
- [ ] **Step 2–4:** implement rule-based classifier (ordered checks: season+year regex, keyboard-walk set, all-digits, dictionary-word+suffix, length/complexity → strength) and reuse grouping (map NT hash → cracked accounts, emit groups with ≥2 members or the empty-password hash); verify.
- [ ] **Step 5: commit** `feat(pwaudit): password pattern, strength, and reuse analysis`.

### Task 3: Cracker seam + hashcat driver
**Files:** Create `internal/pwaudit/crack.go`, `crack_test.go`
**Produces:**
```go
type CrackerInfo struct { Name, Version, Path string; Available bool }
type Progress struct { Recovered, Total int; Percent float64; Speed, ETA string }
type Cracker interface {
    Detect(ctx context.Context) CrackerInfo
    Crack(ctx context.Context, ntHashes []string, wordlist string, rules []string, onProgress func(Progress)) (map[string]string, error) // ntHash(lower)->plaintext
}
type HashcatCracker struct { Run func(ctx, name string, args ...string) *exec.Cmd; LookPath func(string)(string,error) } // seams
```
- [ ] **Step 1: failing tests** — inject a fake `LookPath`/`Run` (a tiny script echoing a hashcat `--show` style `nthash:plaintext` and status lines) → `Detect` reports Available with version; `Crack` returns the hash→plaintext map and calls `onProgress`. A `LookPath` miss → `Available:false`. Follow the exec-seam pattern used by `internal/tracerun`/`internal/sysconf` if present.
- [ ] **Step 2–4:** implement hashcat invocation `-m 1000 -a 0 <hashfile> <wordlist> [-r rules] --status --status-json --potfile-path <tmp>`, read cracked via `--show`/potfile parse, stream progress from status-json; verify.
- [ ] **Step 5: commit** `feat(pwaudit): cracker detection and hashcat driver (injectable)`.

### Task 4: Audit orchestration
**Files:** Create `internal/pwaudit/audit.go`, `audit_test.go`
**Consumes:** Tasks 1–3. **Produces:**
```go
type AuditReport struct {
    Total, Cracked int; PercentCracked float64
    Results []Result; Reuse []ReuseGroup
    Patterns []struct{ Pattern string; Count int }
    CrackerUsed string
}
func RunAudit(ctx, accounts []Account, c Cracker, wordlist string, rules []string, onProgress func(Progress)) (AuditReport, error)
```
- [ ] **Step 1: failing test** — `TestRunAudit` with a fake Cracker that cracks 2 of 3 accounts (two sharing a hash) → Total 3, Cracked 2, one reuse group, per-result Strength/Pattern populated, Patterns summarized. `TestRunAuditNoCracker` (unavailable) → error. 20s-free (cracker controls its own time; pass ctx through).
- [ ] **Step 2–4:** implement; map cracked back to accounts by NT hash, classify, detect reuse, summarize; verify + `go test ./internal/pwaudit/`.
- [ ] **Step 5: commit** `feat(pwaudit): audit orchestration and report`.

### Task 5: Wordlist manager
**Files:** Create `internal/wordlist/wordlist.go`, `wordlist_test.go`
**Produces:**
```go
type Entry struct { Name, Desc, URL, SHA256 string; Size int64 }
func Catalog() []Entry                       // rockyou + a few SecLists, pinned URL+sha256
func Installed(dir string) ([]string, error)
func Download(ctx, e Entry, dir string, client *http.Client, onProgress func(done,total int64)) (string, error) // verify sha256; abort+delete on mismatch
```
- [ ] **Step 1: failing tests** — `TestDownloadVerifies` (httptest server serves bytes whose sha256 matches an Entry → file written, returned path exists) and `TestDownloadChecksumMismatch` (served bytes don't match → error, no file left on disk). `TestInstalled` lists files in a temp dir.
- [ ] **Step 2–4:** implement streaming download with progress + rolling sha256, compare, `os.Remove` on mismatch; verify.
- [ ] **Step 5: commit** `feat(wordlist): checksum-verified wordlist catalog and downloader`.

### Task 6: Encrypted save (securebox + report persistence)
**Files:** Create `internal/securebox/securebox.go`, `securebox_test.go`; add `SaveReport`/`LoadReport` in `internal/pwaudit/store.go` (+ test)
**Produces:**
```go
// securebox: salt|nonce|secretbox(plaintext), key = argon2id(passphrase, salt)
func Seal(passphrase string, plaintext []byte) ([]byte, error)
func Open(passphrase string, blob []byte) ([]byte, error)
// pwaudit:
func SaveReport(path, passphrase string, r AuditReport) error   // json -> Seal -> atomic write
func LoadReport(path, passphrase string) (AuditReport, error)
```
- [ ] **Step 1: failing tests** — `TestSealOpenRoundTrip`, `TestOpenWrongPassphrase` (error, no plaintext); `TestSaveLoadReport` round-trips an AuditReport, `TestLoadReportWrongPassphrase` errors.
- [ ] **Step 2–4:** implement mirroring `internal/ipspace/crypto.go`'s format (argon2id params, 16-byte salt, 24-byte nonce, atomic temp-file write); verify.
- [ ] **Step 5: commit** `feat(securebox,pwaudit): encrypted audit-report persistence`.

### Task 7: CLI + App bindings
**Files:** Modify `internal/cli/cli.go` (+ `cli_adaudit_test.go`); create `internal/app/adaudit_methods.go` (+ test)
**Produces:** CLI `meridian-cli adaudit --dump <file> [--wordlist <path>] [--rules <path>] [--json]` (detect cracker, parse, run, print summary — no plaintext unless `--show`); App methods: `ADAuditDetect() CrackerInfo`, `ADAuditParseFile(path)/ADAuditParseText(text) ([]Account,error)`, `ADAuditRun(...) (AuditReport,error)` (emits `adaudit:progress` Wails events), `WordlistCatalog()/WordlistInstalled()/WordlistDownload(name)`, `ADAuditSave(path,passphrase)`.
- [ ] **Step 1: failing tests** — `cli_adaudit_test.go` runs `adaudit` with an injected fake cracker + a temp dump file, asserts the summary prints counts (and omits plaintext without `--show`); `adaudit_methods_test.go` drives `App.ADAuditRun` with a fake cracker.
- [ ] **Step 2–4:** implement, wiring the cracker seam so tests inject a fake; `go test ./...`; verify.
- [ ] **Step 5: commit** `feat(cli,app): adaudit command and bindings`.

### Task 8: AdAudit.svelte tool + Workbench wiring
**Files:** Create `frontend/src/lib/tools/AdAudit.svelte`; modify `frontend/src/lib/Workbench.svelte` (import, route, add a "Security" group entry `ready:true`); regenerated `wailsjs` bindings.
> UI verified by `wails build` (no frontend unit harness — TDD config/generated exception; stated in ledger at task start).
- [ ] **Step 1: regenerate bindings** (`wails generate module`); grep for `ADAuditRun`.
- [ ] **Step 2: build the tool** — the defensive-use note; an **Import** pane (file picker + paste box); a **cracker/wordlist status** row (detected hashcat + wordlist picker with a Download action for catalog entries); a **Run** button + live progress bar (via `adaudit:progress` events); a **results table** (account / status / password [masked by default, reveal toggle] / strength / pattern); **reuse** and **summary** panels; an **encrypted Save** action (passphrase). Plain-language captions; theme tokens; no horizontal overflow at phone width.
- [ ] **Step 3: wire Workbench** — new "Security" group, `{ id:"adaudit", name:"AD password audit", group:"Security", ready:true }`, import + route branch.
- [ ] **Step 4: verify** `wails build` succeeds.
- [ ] **Step 5: commit** `feat(ui): AD password audit tool`.

## Self-review notes
- **Spec coverage:** import (T1), analysis/pattern/reuse (T2), detect+crack (T3), orchestration (T4), wordlist manager (T5), encrypted save (T6), CLI/app (T7), UI (T8). DCSync/bundled-cracker/netcat are explicit non-goals.
- **Type consistency:** `Account` (T1) → `Result`/`ReuseGroup` (T2) → `AuditReport` (T4) consumed unchanged by T7/T8; `Cracker` seam (T3) injected in T4/T7 tests.
- **Review Focus:** all five rows map to named tests (malformed parse, reuse+empty-pw, cracker-absent, checksum-mismatch, wrong-passphrase).
- **Safety:** plaintext only in-memory + encrypted store; cracker/HTTP seams keep tests offline; additive-only; UI masks passwords by default.
- **Proportion:** engine is the substance (T1–T6), thin wiring (T7), presentation (T8).
