# TLS certificate inspector Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A TLS certificate inspector — engine, CLI, and a Workbench tool — that connects to a host, reports the presented certificate/chain, and gives a plain-language verdict.

**Architecture:** New pure engine `internal/tlscheck` (stdlib `crypto/tls`+`crypto/x509`, dial + roots seams for offline tests). Thin `internal/cli` `tls` command and `internal/app` `TLSCheck` method over it; a Svelte `Tls.svelte` tool in the Workbench.

**Tech Stack:** Go 1.23 stdlib only; Svelte/TS; wails bindings via `wails generate module`.

**Spec:** `docs/superpowers/specs/2026-11-10-meridian-tlscheck-design.md`

## Global Constraints
- Go 1.23; **stdlib only**, no new dependency.
- Two test seams as package vars: `dialTLS` (tests point at a local `tls.Listener`) and `systemRoots` (tests inject a test CA pool; default returns `nil` → system roots). Fully offline tests.
- Default port **443**; expiry warning threshold **21 days**.
- Generic behavior, no user identity on the wire. CLI has `--json` (§4.9).
- `wails generate module` (binary at `$(go env GOPATH)/bin/wails`) churns unrelated `frontend/wailsjs/runtime/*`; keep only the `models.ts` + `App.js` + `App.d.ts` delta for the new method, `git checkout -- frontend/wailsjs/runtime` the rest.

## Review Focus
- **Unreachable host / connection refused / timeout** — `Check` returns a clear error, no partial report, no panic. (Task 1 `TestCheckDialError`.)
- **Host serving a broken cert** (expired, self-signed, wrong hostname) — still returns a full report with the correct non-Valid verdict rather than erroring. (Task 1 `TestCheckExpired/SelfSigned/HostnameMismatch`.)
- **Trust vs expiry vs hostname are independent signals** — an expired-but-CA-signed cert still reports `ChainTrusted` true (trust checked within the cert's validity window), and a trusted cert reached on the wrong name reports `HostnameMatch` false with `ChainTrusted` true. (Task 1 `TestCheckExpired`, `TestCheckHostnameMismatch`.)
- **Weak crypto** — RSA < 2048 and SHA-1/MD5 signatures are flagged even on an otherwise-valid, trusted cert. (Task 1 `TestCheckWeakKey`.)
- **Target parsing** — `host`, `host:port`, and `scheme://host/path` all resolve to the right host+port. (Task 1 `TestSplitTarget`.)

---

### Task 1: Engine `internal/tlscheck`

**Files:**
- Create: `internal/tlscheck/tlscheck.go`
- Test: `internal/tlscheck/tlscheck_test.go`

**Interfaces:**
- Produces:
  - `type CertReport struct { Host, Port, Subject, Issuer string; SANs []string; Serial string; NotBefore, NotAfter time.Time; DaysToExpiry int; KeyType string; KeyBits int; SigAlg string; TLSVersion, CipherSuite string; SelfSigned, ChainTrusted, HostnameMatch bool; Verdict string; Warnings []string }` (json camelCase)
  - `func Check(ctx context.Context, target string, timeout time.Duration) (CertReport, error)`
  - package vars `dialTLS` and `systemRoots` (seams), and `splitTarget(target string) (host, port string)`.

- [ ] **Step 1: Write failing tests.** A test helper generates certs with `crypto/x509.CreateCertificate` (a CA + a leaf signed by it, properties per test), starts a one-shot `tls.Listener` on `127.0.0.1:0` serving the leaf, and sets `systemRoots` to a pool with the CA (or an empty pool for the untrusted cases). Tests:

```go
func TestSplitTarget(t *testing.T) {
    for in, want := range map[string][2]string{
        "example.com":            {"example.com", "443"},
        "example.com:8443":       {"example.com", "8443"},
        "https://example.com/x":  {"example.com", "443"},
    } {
        h, p := splitTarget(in)
        require.Equal(t, want[0], h, in)
        require.Equal(t, want[1], p, in)
    }
}

func TestCheckTrustedValid(t *testing.T) { // CA-signed leaf, SAN localhost, RSA2048, SHA256, valid
    addr := serveLeaf(t, validLeaf(t))          // sets systemRoots to the CA pool
    r, err := Check(context.Background(), "localhost:"+port(addr), 3*time.Second)
    require.NoError(t, err)
    require.Equal(t, "Valid", r.Verdict)
    require.True(t, r.ChainTrusted)
    require.True(t, r.HostnameMatch)
    require.False(t, r.SelfSigned)
    require.Equal(t, "RSA", r.KeyType)
    require.Equal(t, 2048, r.KeyBits)
    require.Contains(t, r.SANs, "localhost")
    require.Greater(t, r.DaysToExpiry, 0)
    require.Empty(t, r.Warnings)
}

func TestCheckExpired(t *testing.T) { /* NotAfter in past → Verdict "Expired", Warnings has it, ChainTrusted still true */ }
func TestCheckSelfSigned(t *testing.T) { /* self-signed leaf, empty roots → SelfSigned true, ChainTrusted false, Verdict "Self-signed" */ }
func TestCheckHostnameMismatch(t *testing.T) { /* CA-signed leaf SAN "example.com", dial localhost → HostnameMatch false, ChainTrusted true, Verdict "Hostname mismatch" */ }
func TestCheckWeakKey(t *testing.T) { /* CA-signed RSA1024 leaf, SAN localhost, valid → Verdict "Weak key", Warnings has it */ }
func TestCheckDialError(t *testing.T) { _, err := Check(context.Background(), "127.0.0.1:1", 1*time.Second); require.Error(t, err) }
```

- [ ] **Step 2: Run, verify they fail** (`go test ./internal/tlscheck/ -v`) — Expected: FAIL (package/undefined).

- [ ] **Step 3: Implement `tlscheck.go`.** `Check`: `splitTarget`; `context.WithTimeout`; `dialTLS` with `tls.Config{ServerName: host, InsecureSkipVerify: true}`; read `ConnectionState`; error if no `PeerCertificates`. Fill fields from `leaf` (`crypto/x509.Certificate`): key type/bits via a `PublicKey` type switch (rsa/ecdsa/ed25519), `SigAlg` from `SignatureAlgorithm.String()`, `SANs` = `DNSNames` + `IPAddresses` (stringified). Compute: `HostnameMatch` = `leaf.VerifyHostname(host)==nil`; `SelfSigned` = issuer==subject && `leaf.CheckSignatureFrom(leaf)==nil`; `ChainTrusted` = `leaf.Verify(x509.VerifyOptions{Roots: systemRoots(), Intermediates: <pool of PeerCertificates[1:]>, CurrentTime: leaf.NotBefore})` returns nil (trust checked inside validity so expiry doesn't mask it). `DaysToExpiry` from `NotAfter`. Build `Warnings` for each failing signal and set `Verdict` to the worst, in order: Not yet valid, Expired, Self-signed, Untrusted chain, Hostname mismatch, Expiring soon (<21d), Weak key (RSA<2048), Weak signature (SHA1/MD5 family); else "Valid". `dialTLS`/`systemRoots` default implementations as in Global Constraints.

- [ ] **Step 4: Run, verify pass** (`go test ./internal/tlscheck/ -v`) — Expected: PASS.

- [ ] **Step 5: Commit** (`internal/tlscheck/*`).

---

### Task 2: CLI `tls` command

**Files:**
- Modify: `internal/cli/cli.go` (usage text, switch, `runTLS`, `printTLSReport`)
- Test: `internal/cli/cli_test.go`

**Interfaces:**
- Consumes: `tlscheck.Check`.

- [ ] **Step 1: Write failing test** — `TestRunTLSJSON`-style: run `cli.Run([]string{"tls", "localhost:<port>", "--json"}, …)` against a listener the test starts via the tlscheck test helper is overkill for CLI; instead unit-test `printTLSReport` directly (internal `package cli` test file, like `printasn_test.go`) with a built `tlscheck.CertReport` and assert the output contains the verdict, subject, a SAN, and "Expiring"/days. Also assert `cli.Run([]string{"tls"}, …)` returns exit 2 (missing arg).

- [ ] **Step 2: Run, verify it fails.**

- [ ] **Step 3: Implement.** Add `case "tls": return runTLS(args[1:], stdout, stderr)` to the switch and a `tls <host[:port]>` line to `usage`. `runTLS`: flagset with `--json` and an optional `--timeout` (default 8s); one positional (host); call `tlscheck.Check`; on `--json` use `writeJSON`, else `printTLSReport` with `reportRow`s (Verdict, Subject, Issuer, SANs joined, Valid-until + days, Key, Signature, TLS) and a `Warnings` line.

- [ ] **Step 4: Run, verify pass** (`go test ./internal/cli/ -v`).

- [ ] **Step 5: Commit** (`internal/cli/cli.go`, test file).

---

### Task 3: App method, bindings, and Workbench tool (UI)

**Files:**
- Create: `internal/app/tls_methods.go` (+ `internal/app/tls_methods_test.go`)
- Modify: `frontend/wailsjs/go/app/App.js`, `App.d.ts`, `frontend/wailsjs/go/models.ts` (regenerated)
- Create: `frontend/src/lib/tools/Tls.svelte`
- Modify: `frontend/src/lib/Workbench.svelte` (import, tools array, route)

**Interfaces:**
- Consumes: `tlscheck.Check`. Produces `App.TLSCheck(target string) (tlscheck.CertReport, error)`.

- [ ] **Step 1: Write failing Go test** — `internal/app/tls_methods_test.go`: `TestTLSCheckMethod` starts a listener (reuse the tlscheck approach via an exported test helper or a plain `httptest.NewTLSServer`), calls `App.TLSCheck(addr)`, asserts a non-empty `Verdict`. (If `httptest.NewTLSServer`, its cert isn't in system roots → Verdict "Untrusted chain"/"Self-signed" is fine; assert `Verdict != ""` and `Subject != ""`.)

- [ ] **Step 2: Run, verify it fails** (`go test ./internal/app/ -run TestTLSCheck`).

- [ ] **Step 3: Implement the method** in `internal/app/tls_methods.go`: `func (a *App) TLSCheck(target string) (tlscheck.CertReport, error) { return tlscheck.Check(a.lookupCtx(), target, 8*time.Second) }`. Run tests to pass.

- [ ] **Step 4: Regenerate bindings** — `$(go env GOPATH)/bin/wails generate module`, then `git checkout -- frontend/wailsjs/runtime` (keep only `models.ts`/`App.js`/`App.d.ts`). Confirm `TLSCheck` appears in `App.js` and `CertReport` in `models.ts`.

- [ ] **Step 5: Build the UI** — `frontend/src/lib/tools/Tls.svelte`: a host input + Check button calling `TLSCheck`; while running show a spinner/disabled; on result a card with the **Verdict** (coloured pill: good for Valid, bad otherwise), Subject/Issuer, SAN list, valid-until + days-to-expiry, key + signature, TLS version/cipher, and a Warnings list; on error a clear message. Each metric captioned in plain language. Reuse existing tool/card CSS. Register in `Workbench.svelte`: import `Tls`, add `{ id: "tls", name: "TLS certificate", group: "Internet", ready: true }` to `tools`, and `{:else if active === "tls"}<Tls />` to the route.

- [ ] **Step 6: Verify the build** — `cd frontend && npm run build` — Expected: no errors. Commit all of Task 3.

---

## Self-review notes
- Spec coverage: connect + retrieve-even-if-broken, all report fields, verdict rules, CLI+JSON, Workbench tool (T1–T3); entity-page button is an explicit non-goal.
- Type consistency: `CertReport` defined T1, consumed T2/T3; seams `dialTLS`/`systemRoots` T1 only.
- Proportion: engine-heavy (T1), thin CLI (T2), wiring+UI (T3, build-verified — no Svelte unit harness, matching existing tools).
