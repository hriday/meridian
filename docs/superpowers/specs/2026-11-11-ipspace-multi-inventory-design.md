# IP space: multiple inventories + encrypted-file hand-off — design

## Intent and the why

Today the IP space tool holds **one** encrypted inventory. But a network engineer
often does fractional work for several companies at once, and their address
spaces must never be co-mingled — company A's `10.0.0.0/8` has nothing to do with
company B's. Today that forces one person to run separate installs or separate
passphrases with no way to switch.

This feature makes IP space **multi-inventory**: up to three separate, fully
isolated inventories (e.g. one per client), each with its own passphrase and its
own sync target, switched by a selector. It also adds an **encrypted-file
hand-off** so an inventory can be carried as a file — handed to a colleague, or
parked to free a slot and restored later.

The sync backends (local / S3 / SFTP / git) already shipped; this builds on them.
Each inventory syncs independently to its own backend.

## Goals

- Up to **3 inventories**, kept fully isolated. A selector switches the active one.
- Each inventory is independent end-to-end: name, passphrase, sync backend, base,
  history.
- **Export** an inventory to an encrypted `.enc` file (two modes: *park* and *share*).
- **Import** a `.enc` file with a directed destination the user always chooses:
  merge into a specific inventory, land as a new inventory, or just look (ephemeral).
- **Never auto-merge.** Import always asks.
- **Pull records across inventories** on demand ("reference + merge later").

## Non-goals / anti-goals

- **No auto-merge of imports, ever.** An import that lands anywhere the user
  didn't explicitly pick is a failure, even if "convenient".
- **No co-mingling.** Two inventories never share records implicitly; a cross-
  inventory pull is an explicit, reviewed action.
- **No credential leak via the share path.** A *share* export must never contain
  sync credentials.
- Not more than 3 live inventories. (Files are the overflow; see Rotation.)
- No change to the merge/conflict engine (`ipspace.Merge`) or the four sync backends.

## The model

### Inventories and slots

- An **inventory** is what today's single store is: an append-only event log,
  sealed on disk, with a derived record tree, a sync backend config, and a base.
- There are up to **3 slots**, total (your own counts as one). A slot holds one
  inventory. Each slot has a user-given **name** (e.g. the client's name) and its
  own **passphrase**.
  - The per-inventory passphrase isn't really optional: a company's shared sync
    blob is sealed with that company's passphrase, so matching it *is* the
    inventory's passphrase. You unlock a company when you open it.
- A **selector** at the top of the IP space tool switches the active inventory.
  Everything below (tree, change log, add/edit, sync panel, tool pivots) acts on
  the active inventory only.

### Rotation (slots are a live working set)

Three slots bound what's "checked out" at once; files are cold storage for any
number beyond three:

1. **Park** inventory A to a `.enc` file (park-mode export) and remove it from its
   slot — freeing the slot.
2. **Import** another inventory (say D) into the free slot.
3. Work D; when done, **park** D to a file and **re-import** A.

Park/export is lossless: re-importing A restores it, wired back to A's backend.

### Import — directed, never automatic

Importing a `.enc`:

1. Pick the file, enter its **hand-off password** (see Passwords). Decrypt → the
   file's event log + (for a park file) its sync config.
2. **Preview**: show what's inside — record count, and, when a destination
   inventory is chosen, the incoming/outgoing diff against it (same diff UI as a
   backend sync).
3. **Choose destination:**
   - **Merge into A / B / C** — runs `Merge` against that inventory, shows the
     diff + any conflicts, applies on confirm. (Conflicts pause, same as sync.)
   - **New slot** — lands as its own separate inventory (only if a slot is free;
     otherwise the user parks one first). A park file restores its backend; a
     share file lands with no backend (user attaches one later if they want sync).
   - **Just look** — opens the file's contents read-only in a temporary view;
     nothing is written to any inventory, and it's discarded when closed.

### Export — two modes

- **Park / back-up** (for yourself, to swap out and restore): the inventory's
  event log **and** its sync config (credentials included). Sealed with a hand-off
  password. Re-import restores the inventory fully wired to its backend.
- **Share** (to hand to someone else): the event log **only — no credentials, no
  sync config.** Sealed with a hand-off password the sender communicates out of
  band.

### Cross-inventory pull ("reference + merge later")

From the active inventory, the user can pull records from another open inventory:
select records (or all) from inventory X and merge them into the active one. This
is an explicit, reviewed action (shows a diff, honors conflicts) — never automatic.
*(This is the lowest-priority slice; see Slicing.)*

## Passwords (important, easy to get wrong)

- **Store passphrase (per inventory):** unlocks that inventory on disk and seals
  its local store and its backend's shared blob. One per inventory.
- **Hand-off password (per file):** seals an exported `.enc` file. Chosen at
  export, entered at import. **Deliberately separate** from any store passphrase,
  because a hand-off can go between two people who share no store and must not
  reveal each other's store passphrase. If two people happen to share a team
  passphrase, they can just type that as the hand-off password.
- A file imported into a slot is **re-sealed under the destination inventory's
  store passphrase** once it lands, so the importer opens their slots with their
  own passphrases, not a kept list of hand-off passwords.

## Constraints

- Go 1.25 + Wails v2; pure engine in `internal/ipspace`, thin App methods, one
  Svelte tool. Reuse `ipspace.Merge`, `Seal`/`Open`, and the four backends
  unchanged.
- File open/save uses the Wails runtime dialogs (`runtime.OpenFileDialog` /
  `SaveFileDialog`).
- Secrets (store passphrases, hand-off passwords, backend credentials) never
  touch disk in cleartext and never appear in a label, error, or the UI beyond a
  masked field. A *share* export carries no credentials — enforced by the export
  code building a credential-free config, and covered by a test.
- Must migrate the one existing single inventory into inventory/slot #1 with no
  data loss. The current store/sidecar paths become slot #1's.

## What's left to the builder

- Exact on-disk layout for N inventories (a per-slot subdirectory vs. suffixed
  sidecars), given today's single `ipspace.enc` + `.sync.enc` + `.base.enc`.
- The slot registry (names, which slot maps to which files) — itself an
  unencrypted index (names only, no secrets) vs. per-slot metadata.
- The `.enc` hand-off file format: a small versioned header distinguishing
  *park* (with sync config) from *share* (data only), wrapping the existing
  sealed-events format.
- Svelte: the selector/tab UI, the import destination chooser, the read-only
  "just look" view, and the park/share export affordances.
- Whether "new slot" import and "create blank inventory" share one code path.

## Slicing (each its own PR + review, TDD)

1. **Multi-inventory core + selector.** Engine + App refactor from a single store
   to up to 3 named inventories, each with its own passphrase/store/sync
   config/base; migrate the existing store into slot #1; a selector in the UI;
   create / rename / remove a blank inventory. (No files yet.) *The big one.*
2. **Export** (park + share modes) to a `.enc` file via a save dialog.
3. **Import**: file + hand-off password → preview → destination chooser (merge
   into a chosen inventory · new slot · just look).
4. **Cross-inventory record pull** ("merge later").

Each slice reuses the shipped `Merge` / `Seal`-`Open` / backend code. Slice 1 is
the foundation and the largest; 2–4 are additive.
