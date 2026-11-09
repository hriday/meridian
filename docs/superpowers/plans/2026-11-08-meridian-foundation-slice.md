# Meridian Foundation Slice Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A running Meridian desktop window whose first working tool is a v4/v6 subnet calculator, built on the any-value input recognizer and a CLI that every later tool will reuse.

**Architecture:** Pure Go engine packages under `internal/` (no UI or framework imports) hold all logic and own all tests. A thin Wails v2 controller (`internal/app`) binds engine functions to a Svelte+TypeScript frontend (the Workbench shell: tool sidebar, tabbed main area, ⌘K search). A separate `cmd/meridian-cli` binary exposes the same engines on the command line with `--json`. Engines are the source of truth; UI and CLI are adapters over them.

**Tech Stack:** Go 1.23+, Wails v2.10.1, Svelte + TypeScript + Vite (Wails template), `net/netip` (stdlib) + `go4.org/netipx` for prefix math, `math/big` for host counts. Testing: Go `testing` + `testify`.

**Spec:** `docs/superpowers/specs/2026-11-08-meridian-design.md` (this plan implements the first-milestone foundation; see "Follow-on plans" at the end for the rest of v1).

## Global Constraints

- Module path: `github.com/hriday/meridian`. GUI binary: `meridian`. CLI binary: `meridian-cli`.
- Licence header is not required per-file; repo is Apache 2.0 (one `LICENSE` file, Task 1).
- Repo stays **private** until the v1 core is polished; do not add public-facing release steps.
- IPv4 **and** IPv6 are first-class in every engine — no function may support one and not the other.
- Engine packages (`internal/netinput`, `internal/subnet`) import no UI, Wails, or framework code, so they stay unit-testable in isolation.
- Counts that can exceed 2^53 (IPv6 address counts) cross the Go↔JS boundary as **decimal strings**, never JS numbers.
- UI acceptance bar (from spec §3): interactions respond <100 ms; no engine call blocks the UI thread; large lists virtualised. Visual sign-off is the owner's.
- `.superpowers/` stays git-ignored (already in `.gitignore`).

## Review Focus

Spec-implied inputs no happy-path test exercises; each line's test is added to the owning task.
- **Malformed input** — garbage, empty, or wrong-type values to `Classify`/`Describe` return an error, never panic (Tasks 2, 3).
- **Host bits set in a CIDR** — `10.0.0.5/24` normalizes to network `10.0.0.0/24` rather than erroring (Task 3).
- **IPv6 magnitudes** — a `/0` or `/32` IPv6 block reports its astronomically large count as a correct decimal string, not an overflow (Task 3).
- **v4 corner prefixes** — `/31` (RFC 3021, 2 usable) and `/32` (1 host, no broadcast) report sane usable ranges (Task 3).
- **Summarize edge cases** — adjacent/overlapping prefixes aggregate; mixing v4 and v6 in one summarize call errors (Task 3).

---

### Task 1: Project scaffold — a window that runs on all three OSes

**Files:**
- Create: `go.mod`, `main.go`, `wails.json`, `LICENSE`, `README.md`
- Create: `internal/app/app.go` (the `App` controller, Greet-less)
- Create: `frontend/` (Wails Svelte-TS template: `index.html`, `src/App.svelte`, `src/main.ts`, `package.json`, `vite.config.ts`)
- Create: `.github/workflows/ci.yml`
- Test: `internal/app/app_test.go`

**Interfaces:**
- Produces: `app.New() *App`; `App` has `ctx context.Context` set in `startup(ctx)`. Later tasks add bound methods to this struct.

- [ ] **Step 1: Initialize the Wails project**

Run `wails init -n meridian -t svelte-ts` into a temp dir, then move its `frontend/`, `wails.json`, and template `main.go` into the repo root; set `go.mod` module to `github.com/hriday/meridian`. Replace the generated `app.go` location with `internal/app/app.go` and update `main.go`'s import and `Bind` to use `app.New()`.

- [ ] **Step 2: Write the failing test**

```go
func TestNewAppHasNoContextBeforeStartup(t *testing.T) {
    a := app.New()
    require.NotNil(t, a)
    require.Nil(t, a.Ctx()) // Ctx() exposes a.ctx for tests
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/app/ -run TestNewAppHasNoContextBeforeStartup -v`
Expected: FAIL (package/undefined `app.New` or `Ctx`).

- [ ] **Step 4: Implement `app.New() *App`, `(*App).startup(context.Context)`, and `(*App).Ctx() context.Context` in `internal/app/app.go`**

Mirror the Wails v2 binding pattern: unexported `ctx`, set in `startup`. `Ctx()` is a test accessor.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/app/ -v` → PASS.

- [ ] **Step 6: Verify the app builds and runs**

Run: `wails build` → Expected: a `meridian` binary in `build/bin/`. Manually run it: a window titled "Meridian" opens. (Record this manual check in the commit message.)

- [ ] **Step 7: CI stub**

`ci.yml` runs `go vet ./...` and `go test ./...` on ubuntu-latest. (GUI build is not run in CI for this slice.)

- [ ] **Step 8: Commit**

```bash
git add -A && git commit -m "feat: scaffold Meridian Wails app and CI"
```

---

### Task 2: Any-value input recognizer

**Files:**
- Create: `internal/netinput/netinput.go`
- Test: `internal/netinput/netinput_test.go`

**Interfaces:**
- Produces:
  ```go
  type Kind int
  const ( KindUnknown Kind = iota; KindIPv4; KindIPv6; KindPrefixV4; KindPrefixV6; KindASN; KindDomain; KindHostname )
  func (k Kind) String() string
  type Result struct { Kind Kind; Normalized string } // Normalized: canonical form (ASN→"AS13335", IP→netip canonical, domain→lowercased, no trailing dot)
  func Classify(input string) (Result, error)          // trims surrounding whitespace first
  ```

- [ ] **Step 1: Write the failing tests**

```go
func TestClassify(t *testing.T) {
    cases := []struct{ in string; kind netinput.Kind; norm string }{
        {"1.1.1.1", netinput.KindIPv4, "1.1.1.1"},
        {"  1.1.1.1 ", netinput.KindIPv4, "1.1.1.1"},
        {"2606:4700:4700::1111", netinput.KindIPv6, "2606:4700:4700::1111"},
        {"1.1.1.0/24", netinput.KindPrefixV4, "1.1.1.0/24"},
        {"2606:4700::/32", netinput.KindPrefixV6, "2606:4700::/32"},
        {"AS13335", netinput.KindASN, "AS13335"},
        {"as13335", netinput.KindASN, "AS13335"},
        {"13335", netinput.KindASN, "AS13335"},
        {"cloudflare.com", netinput.KindDomain, "cloudflare.com"},
        {"CloudFlare.com.", netinput.KindDomain, "cloudflare.com"},
        {"router1", netinput.KindHostname, "router1"},
    }
    for _, c := range cases {
        got, err := netinput.Classify(c.in)
        require.NoError(t, err, c.in)
        require.Equal(t, c.kind, got.Kind, c.in)
        require.Equal(t, c.norm, got.Normalized, c.in)
    }
}

func TestClassifyMalformed(t *testing.T) {
    for _, in := range []string{"", "   ", "...", "1.1.1.1/33", "AS", "99999999999999999999", "has space"} {
        _, err := netinput.Classify(in)
        require.Error(t, err, in)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/netinput/ -v` → FAIL (undefined).

- [ ] **Step 3: Implement `Classify` and `Kind.String()`**

Order of checks: trimmed empty → error; contains `/` → try `netip.ParsePrefix` (version picks PrefixV4/V6); try `netip.ParseAddr` (→ IPv4/IPv6); `^(?i:AS)?\d+$` with value ≤ 4294967295 → ASN (normalize to `AS<n>`); else a hostname/domain label check (`[A-Za-z0-9-]` labels, strip one trailing dot) → Domain if it has ≥2 labels, Hostname if one; anything else → error. Keep the ASN regex and domain label rules in one place.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/netinput/ -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/netinput && git commit -m "feat: any-value input recognizer"
```

---

### Task 3: Subnet calculator engine (v4/v6)

**Files:**
- Create: `internal/subnet/subnet.go`
- Test: `internal/subnet/subnet_test.go`
- Modify: `go.mod` (add `go4.org/netipx`)

**Interfaces:**
- Produces:
  ```go
  type Info struct {
      Prefix      string // normalized CIDR (host bits cleared)
      Version     int    // 4 or 6
      Network     string
      Broadcast   string // v4 only; "" for v6
      Netmask     string // v4 dotted-decimal; "" for v6
      Wildcard    string // v4 only; "" for v6
      PrefixLen   int
      TotalCount  string // decimal string (big)
      UsableFirst string
      UsableLast  string
      UsableCount string // decimal string (big)
  }
  func Describe(cidr string) (Info, error)              // accepts host-bits-set input, normalizes
  func Split(cidr string, newPrefixLen int) ([]string, error) // equal subnets at newPrefixLen
  func Summarize(cidrs []string) ([]string, error)     // minimal covering set; errors on mixed families
  ```

- [ ] **Step 1: Write the failing tests**

```go
func TestDescribeIPv4(t *testing.T) {
    got, err := subnet.Describe("10.0.0.5/24") // host bits set
    require.NoError(t, err)
    require.Equal(t, "10.0.0.0/24", got.Prefix)
    require.Equal(t, "10.0.0.255", got.Broadcast)
    require.Equal(t, "255.255.255.0", got.Netmask)
    require.Equal(t, "0.0.0.255", got.Wildcard)
    require.Equal(t, "256", got.TotalCount)
    require.Equal(t, "10.0.0.1", got.UsableFirst)
    require.Equal(t, "10.0.0.254", got.UsableLast)
    require.Equal(t, "254", got.UsableCount)
}

func TestDescribeV4CornerPrefixes(t *testing.T) {
    s32, _ := subnet.Describe("192.0.2.7/32")
    require.Equal(t, "1", s32.TotalCount)
    require.Equal(t, "192.0.2.7", s32.UsableFirst)
    require.Equal(t, "192.0.2.7", s32.UsableLast)
    require.Equal(t, "", s32.Broadcast)

    s31, _ := subnet.Describe("192.0.2.6/31") // RFC 3021: both usable
    require.Equal(t, "2", s31.TotalCount)
    require.Equal(t, "2", s31.UsableCount)
    require.Equal(t, "192.0.2.6", s31.UsableFirst)
    require.Equal(t, "192.0.2.7", s31.UsableLast)
}

func TestDescribeIPv6Magnitude(t *testing.T) {
    got, err := subnet.Describe("2606:4700::/32")
    require.NoError(t, err)
    require.Equal(t, 6, got.Version)
    require.Equal(t, "", got.Broadcast)
    require.Equal(t, "79228162514264337593543950336", got.TotalCount) // 2^96
    require.Equal(t, got.TotalCount, got.UsableCount)                 // v6: all usable
}

func TestDescribeMalformed(t *testing.T) {
    for _, in := range []string{"", "1.1.1.1", "nonsense", "10.0.0.0/33"} {
        _, err := subnet.Describe(in)
        require.Error(t, err, in)
    }
}

func TestSplit(t *testing.T) {
    out, err := subnet.Split("10.0.0.0/24", 26)
    require.NoError(t, err)
    require.Equal(t, []string{"10.0.0.0/26","10.0.0.64/26","10.0.0.128/26","10.0.0.192/26"}, out)
    _, err = subnet.Split("10.0.0.0/24", 23) // newLen must be longer
    require.Error(t, err)
}

func TestSummarize(t *testing.T) {
    out, err := subnet.Summarize([]string{"10.0.0.0/25","10.0.0.128/25"})
    require.NoError(t, err)
    require.Equal(t, []string{"10.0.0.0/24"}, out)
    _, err = subnet.Summarize([]string{"10.0.0.0/24","2001:db8::/32"})
    require.Error(t, err) // mixed families
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/subnet/ -v` → FAIL.

- [ ] **Step 3: Implement the engine**

Use `netip.ParsePrefix` then `.Masked()` to normalize. Counts: `big.Int` left-shift `1 << (bits - prefixLen)` where bits is 32 or 128, rendered with `.String()`. v4 usable: `/32`→1 (the address), `/31`→2 (both), else total−2 with first = network+1, last = broadcast−1; broadcast = last address of block. v6: usable = total, first = network, last = last address. Netmask/wildcard v4 only. Use `netipx` for last-address-in-prefix (`netipx.RangeOfPrefix`) and for `Summarize` (build `netipx.IPSetBuilder`, add prefixes, `IPSet.Prefixes()`); reject when input prefixes are not all the same `.Addr().Is4()`/`Is6()`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/subnet/ -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/subnet go.mod go.sum && git commit -m "feat: v4/v6 subnet calculator engine"
```

---

### Task 4: Bind engines and ship the Subnet Calculator tool in the Workbench

**Files:**
- Modify: `internal/app/app.go` (add bound methods), `main.go` (ensure bind list)
- Create: `frontend/src/lib/Workbench.svelte` (sidebar + tabs + ⌘K bar), `frontend/src/lib/tools/SubnetCalc.svelte`
- Modify: `frontend/src/App.svelte` (render Workbench)
- Test: `internal/app/app_methods_test.go`

**Interfaces:**
- Consumes: `netinput.Classify` (Task 2), `subnet.Describe`/`Split`/`Summarize` (Task 3).
- Produces (bound to frontend; Wails generates TS bindings in `frontend/wailsjs/go/app/App.*`):
  ```go
  func (a *App) Classify(input string) (netinput.Result, error)
  func (a *App) SubnetDescribe(cidr string) (subnet.Info, error)
  func (a *App) SubnetSplit(cidr string, newPrefixLen int) ([]string, error)
  ```

- [ ] **Step 1: Write the failing test**

```go
func TestAppSubnetDescribeDelegates(t *testing.T) {
    a := app.New()
    info, err := a.SubnetDescribe("10.0.0.0/24")
    require.NoError(t, err)
    require.Equal(t, "10.0.0.255", info.Broadcast)
}
func TestAppClassifyDelegates(t *testing.T) {
    a := app.New()
    r, err := a.Classify("AS13335")
    require.NoError(t, err)
    require.Equal(t, netinput.KindASN, r.Kind)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/ -run Delegates -v` → FAIL (undefined methods).

- [ ] **Step 3: Implement the three bound methods** as thin delegations to the engine packages in `internal/app/app.go`; confirm `main.go` binds `app.New()`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/ -v` → PASS.

- [ ] **Step 5: Build the Workbench shell and Subnet Calc panel**

`Workbench.svelte`: left sidebar with grouped tool list (only "Subnet calculator" active; others shown disabled), a top search input (⌘K focuses it), and a tabbed main area. `SubnetCalc.svelte`: an input accepting any value; on submit, call `Classify`; if a prefix, call `SubnetDescribe` and render `Info` as a labeled result card (and offer Split); non-prefix input shows a gentle "not a subnet" hint (Nielsen: clear error, no dead end). Keep styling aligned to the approved dark mockup; honor the <100 ms / non-blocking bar (call bindings async, never freeze the input).

- [ ] **Step 6: Verify end to end**

Run: `wails dev`; type `10.0.0.0/24` → result card shows network/broadcast/mask/usable. Type `10.0.0.5/24` → normalizes to `/24`. Type `cloudflare.com` → friendly hint. (Record this manual check in the commit.)

- [ ] **Step 7: Commit**

```bash
git add -A && git commit -m "feat: subnet calculator tool in Workbench shell"
```

---

### Task 5: CLI with JSON output

**Files:**
- Create: `cmd/meridian-cli/main.go`, `internal/cli/cli.go`
- Test: `internal/cli/cli_test.go`

**Interfaces:**
- Consumes: `netinput`, `subnet`.
- Produces: `func Run(args []string, stdout, stderr io.Writer) int` — testable entry; `main.go` calls `os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))`.
- Commands: `meridian-cli classify <value> [--json]`; `meridian-cli subnet <cidr> [--split <len>] [--json]`.

- [ ] **Step 1: Write the failing tests**

```go
func TestRunSubnetJSON(t *testing.T) {
    var out, errb bytes.Buffer
    code := cli.Run([]string{"subnet", "10.0.0.0/24", "--json"}, &out, &errb)
    require.Equal(t, 0, code)
    var info subnet.Info
    require.NoError(t, json.Unmarshal(out.Bytes(), &info))
    require.Equal(t, "10.0.0.255", info.Broadcast)
}
func TestRunClassifyText(t *testing.T) {
    var out, errb bytes.Buffer
    code := cli.Run([]string{"classify", "AS13335"}, &out, &errb)
    require.Equal(t, 0, code)
    require.Contains(t, out.String(), "ASN")
    require.Contains(t, out.String(), "AS13335")
}
func TestRunBadArgs(t *testing.T) {
    var out, errb bytes.Buffer
    code := cli.Run([]string{"subnet"}, &out, &errb) // missing arg
    require.NotEqual(t, 0, code)
    require.NotEmpty(t, errb.String())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -v` → FAIL.

- [ ] **Step 3: Implement `Run`** — dispatch on `args[0]`; parse flags with `flag`; `--json` marshals the engine struct, otherwise print a small aligned text view; unknown command / missing arg → write usage to stderr, return 2.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -v` → PASS.

- [ ] **Step 5: Verify the binary**

Run: `go build -o /tmp/meridian-cli ./cmd/meridian-cli && /tmp/meridian-cli subnet 2606:4700::/32 --json` → prints JSON with the 2^96 count string.

- [ ] **Step 6: Commit**

```bash
git add cmd/meridian-cli internal/cli && git commit -m "feat: meridian-cli subnet and classify commands"
```

---

## Follow-on plans (to reach the full v1 core)

Each is its own plan, built on this slice's engines and Workbench shell:
- **B — Lookups & entity pages:** RDAP client, routing/RPKI/PeeringDB data sources with source+freshness labels, the one-page-per-entity UI with pivoting and tabs.
- **C — AS-aware traceroute:** the traceroute/MTR engine (per-OS privilege handling) + AS annotation from Plan B's data sources + the live hop view.
- **D — IP-space records:** encrypted SQLite, nested blocks/subnets/addresses, append-only change log with undo, overlap/dup guards, actor/conflict metadata, CSV import/export, "Save to IP space" wired from entity pages.

Publishing the repo happens after B–D land and the core is polished to the acceptance bar.
