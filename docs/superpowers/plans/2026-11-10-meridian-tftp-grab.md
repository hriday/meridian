# TFTP grab Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans.

**Goal:** A receive-only, ephemeral in-app TFTP server that catches files devices push (e.g. `copy run tftp`), holds them in memory, and lets you preview/save them.

**Architecture:** An `internal/tftp` wrapper around the maintained `github.com/pin/tftp/v3` server: a write handler captures uploads into a bounded in-memory store (per-file size cap + file-count cap, source IP, time); the read handler denies RRQ (receive-only — never serves files). App methods `TftpStart/Stop/List/Preview/Save`; a `TftpReceiver` Svelte tool in the **Collectors** group polls the file list.

**Spec:** (ad-hoc — collector-style ephemeral tool, owner-approved receive-only.)

## Global Constraints

- Receive-only: accept WRQ (uploads); the read handler **denies RRQ** so it can't serve/exfiltrate files. In-memory, ephemeral; cleared on Stop; auto-stop on app shutdown.
- Bounded: per-file size cap (16 MB) and file-count cap (64, oldest evicted); bind 0.0.0.0 on a **high default port** (6969; 69 is privileged), editable.
- Untrusted-input safety is delegated to the vetted pin/tftp library; our wrapper adds the size cap + a `recover`-free, bounds-checked store.

## Review Focus

- **Read requests are denied** (no file serving). (Task 1 test.)
- **Oversize upload is rejected/capped**, not buffered unbounded. (Task 1 test.)
- **A received file is captured with name/source/size** and retrievable. (Task 1 test via a real pin/tftp client.)
- **Lifecycle**: Start/Stop clean, bind-in-use errors, OnShutdown stops it, store cleared on Stop. (Task 1/2 tests.)

---

### Task 1: `internal/tftp` server

**Files:** Create `internal/tftp/server.go`, `internal/tftp/server_test.go`. (go.mod/go.sum already have pin/tftp.)

**Interfaces (produced):**
- `type File struct { ID int \`json:"id"\`; Name string \`json:"name"\`; Source string \`json:"source"\`; Size int \`json:"size"\`; T time.Time \`json:"t"\` }` (metadata; content held internally).
- `type Server struct { … }`, `func New() *Server`
- `func (s *Server) Start(port int) error` (bind 0.0.0.0:port, 0 → OS-chosen; serve pin/tftp on the conn in a goroutine)
- `func (s *Server) Stop()` (Shutdown + clear)
- `func (s *Server) List() (running bool, port int, files []File)`
- `func (s *Server) Content(id int) ([]byte, bool)` and `func (s *Server) Preview(id, max int) (string, bool)` (printable text / hex, capped)

- [ ] **Step 1: Failing tests** (use pin/tftp's own client to drive real transfers):
```go
func TestTftpReceivesFile(t)   // Start(0); tftp.NewClient(addr).Send("sw1.cfg","octet").ReadFrom(bytes.NewReader(cfg)); Eventually List() has 1 file name "sw1.cfg", Source 127.0.0.1, Size len(cfg); Content(id)==cfg
func TestTftpDeniesRead(t)      // client.Receive("x","octet").WriteTo(buf) → error (reads denied)
func TestTftpCapsSize(t)        // sending > maxFile → the file is NOT stored (handler errored); List empty
func TestTftpStopClears(t)      // after a file + Stop(): List running=false, files empty
func TestTftpBindInUse(t)       // held port → Start error
```
- [ ] **Step 2–4: fail → implement → pass (race).** `server.go`: `tftp.NewServer(denyRead, s.onWrite)`; `denyRead` returns an error; `onWrite(filename, wt)` reads `wt.WriteTo(limitWriter{max})` (cap), gets the source via `wt.(interface{ RemoteAddr() net.UDPAddr })`, appends to the bounded store under the mutex (evict oldest past the count cap). `Start`: `net.ListenUDP` then `go srv.Serve(conn)`; record port/running under the mutex. `Stop`: `srv.Shutdown()`, clear files, running=false. `List`/`Content`/`Preview` copy under the mutex. Full `go test ./...` green; cross-compile linux/windows.
- [ ] **Step 5: Commit** — `feat(tftp): receive-only TFTP server (in-memory grab)`.

---

### Task 2: App methods + TFTP tool

**Files:** Create `internal/app/tftp_methods.go`, `internal/app/tftp_methods_test.go`, `frontend/src/lib/tools/TftpReceiver.svelte`; modify `internal/app/app.go` (a `tftpSrv *tftp.Server` + OnShutdown stop), `frontend/src/lib/Workbench.svelte`; regenerate bindings.

**Interfaces (produced):**
- `func (a *App) TftpStart(port int) error`, `TftpStop()`, `TftpList() TftpView`, `TftpPreview(id int) string`, `TftpSave(id int) (string, error)` (Wails `SaveFileDialog` → write the content; returns the saved path or "" if cancelled).
- `type TftpView struct { Running bool; Port int; Files []tftp.File }`

- [ ] **Step 1: Failing app test** `tftp_methods_test.go`: `TestTftpStartListStop` — `TftpStart(0)`; pin/tftp client sends a file to the port; `Eventually` `TftpList().Files` has it; `TftpStop()` → Running false. `TestTftpStartBindInUse`.
- [ ] **Step 2–4: fail → implement → pass.** `app.go`: `tftpSrv: tftp.New()`; OnShutdown stops it. Methods delegate; `TftpSave` uses `wruntime.SaveFileDialog(a.ctx, …)` then `os.WriteFile`. Full `go test ./...` green.
- [ ] **Step 5: Build `TftpReceiver.svelte`** — port (default 6969) + Start/Stop; a received-files table (name · source · size · time); click a row → a text preview pane (`TftpPreview`) + a **Save** button (`TftpSave`). Polls `TftpList` ~1s; restore on mount; clear on Stop; an in-app usage note ("point a device's `copy … tftp://<this-host>:<port>/<name>` here; received files are in memory, cleared on Stop; read requests are refused").
- [ ] **Step 6: Wire Workbench** — `{ id:"tftp", name:"TFTP grab", group:"Collectors", ready:true }` + import + branch.
- [ ] **Step 7: Regenerate bindings; `npm run build` green.**
- [ ] **Step 8: Manual check** (dev server + Playwright, real transfer): `TftpStart` on 6969; from the shell `tftp 127.0.0.1 6969` + `put <file>` (or a python/pin client) to upload a config; confirm it appears with name/source/size, the preview shows the text, Save works; a read (`get`) is refused; Stop clears. Regression on another Collectors tool.
- [ ] **Step 9: Commit** — `feat(app,ui): TFTP grab tool`.

---

## Self-Review notes
- Spec coverage: Task 1 = the receive-only server (deny reads, size/count caps); Task 2 = app + tool. Non-goals: serving files (the push direction), persistence.
- Review Focus: deny-read + size-cap + capture on Task 1; lifecycle on Task 2.
- Dependency: pin/tftp v3 (focused, maintained); go.mod/go.sum committed; cross-compiles.
