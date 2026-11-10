# Meridian — Security suite slice 1: AD password audit (design)

**Status:** approved by owner (2026-11-09) with the three decisions below.

## Intent and why
A **defensive self-audit of one's own Active Directory.** An administrator
imports their *own* domain's password hashes and finds which accounts have weak
or guessable passwords, so they can fix policy. The tool shows the recovered
plaintext and the *pattern* (e.g. "Season+Year+!") because you cannot fix what
you cannot see — the owner's standard: "pointless to tell someone their password
is weak but you don't know what it is." Everything is **local**; nothing is ever
uploaded, and Meridian hosts nothing for anyone.

This is standard blue-team tooling (internal password audits are routine). It is
scoped to auditing hashes the operator already administers — **not** attacking
systems they don't own.

## Decisions (confirmed)
1. **Detect & shell out** to an installed **hashcat** (and/or **John the
   Ripper**) — Meridian does not bundle a cracker. Detect it, drive it, and show
   install guidance if absent.
2. **Slice 1 = AD audit + the wordlist manager** it depends on. Netcat is a
   separate later slice.
3. **Input: an NTLM dump file** (`secretsdump`/pwdump `user:rid:lm:nt:::`) **and
   pasting raw NT hashes.** (DCSync/live-DC pull is a deferred non-goal.)

## Goals
- **Import** accounts + NT hashes from:
  - a **dump file** in pwdump/secretsdump format (`DOMAIN\user:rid:lmhash:nthash:::`),
    flagging machine accounts (`name$`) and skipping blanks/comments;
  - **pasted raw hashes** — lines of 32-hex NT hashes, or `user:hash`.
- **Detect** hashcat/John on the system (PATH + common install dirs); report
  which is available and its version, with guidance when none is found.
- **Run** the crack by shelling out (hashcat NTLM mode `-m 1000 -a 0` against a
  chosen wordlist, optional rules), streaming **live progress** (recovered count,
  %, speed, ETA) parsed from the cracker's status output.
- **Analyze** (pure, deterministic, unit-tested):
  - map recovered plaintext back to accounts by NT hash;
  - a **strength** rating per account (cracked ⇒ at least "weak");
  - a detected **pattern** label (season+year, keyboard walk, single dictionary
    word, name+digits, all-digits, date, etc.);
  - **password reuse** — accounts sharing an NT hash (same password), including
    the empty-password and well-known-hash cases;
  - **summary** — % cracked, counts, weakest accounts, top patterns, reuse
    clusters.
- **Wordlist manager:** a built-in catalog (rockyou, a few SecLists lists) with
  pinned URLs + **SHA256 verification**; download with progress; list installed
  wordlists; pick one for a run. Never auto-downloads without the user asking.
- **Results are sensitive** (they contain plaintext passwords): the report lives
  in memory by default; **persisting is an explicit, passphrase-encrypted action**
  reusing the IP-space encryption approach (argon2id + NaCl secretbox). No
  plaintext passwords are ever written unencrypted to disk, logged, or sent out.
- **CLI** counterpart (`meridian-cli adaudit …`) for the same engine.
- UI follows the approved "security" template: an import pane, cracker/wordlist
  status, a run button with a live progress bar, and a results table (account /
  status / password / strength / pattern) with reuse and summary panels. Plain-
  language captions throughout (the NTP-screen standard).

## Non-goals / deferred
- **No DCSync / live domain-controller extraction**, no credential capture off
  the wire, no AD write/attack actions — import only.
- **No bundled cracker**, no cracking backend of our own, no GPU management.
- **Netcat** utility — next slice.
- Kerberos (AS-REP/Kerberoast hash) cracking, LM-hash cracking, mask/brute-force
  attacks, distributed cracking — possible later, not slice 1.
- Any hosting/sharing of hashes, wordlists, or results for others.

## Constraints (global)
- Go directive stays **1.23**; reuse the IP-space crypto approach for the
  optional encrypted save. New deps only if unavoidable (prefer stdlib
  `os/exec`, `crypto/sha256`, `net/http`).
- The cracker invocation goes through an injected **seam** (an interface) so the
  engine is unit-tested with a fake cracker — tests never require hashcat.
- The wordlist downloader's HTTP goes through an injected client seam; checksum
  mismatch **aborts** and deletes the partial file.
- Never log, print, or transmit recovered plaintext except into the in-memory
  report and (on explicit request) the encrypted store. Outbound requests keep
  the generic `meridian/*` User-Agent; no user identity.
- Defensive framing surfaced in the UI: a one-line "authorized self-audit of
  your own directory; stays on this machine" note.

## What the builder decides
- Go package layout within `internal/` (suggest `internal/pwaudit` for the audit
  engine + `internal/wordlist` for the downloader), exact helper names, the
  pattern-classifier's rule set (beyond the categories listed), and the Svelte
  layout details within the security template and the plain-language standard.
- Whether the optional encrypted save reuses `internal/ipspace`'s crypto
  directly or a small shared helper extracted from it — builder's call, as long
  as no plaintext is persisted unencrypted.
