<script lang="ts">
  import { onMount } from "svelte";
  import {
    IPSpaceStatus, IPSpaceUnlock, IPSpaceTree, IPSpaceHistory,
    IPSpaceAdd, IPSpaceEdit, IPSpaceDelete, IPSpaceUndo, IPSpaceImportCSV, IPSpaceExportCSV,
    IPSpaceList, IPSpaceActive, IPSpaceSetActive, IPSpaceCreateInventory, IPSpaceRenameInventory, IPSpaceRemoveInventory,
    IPSpaceSyncInfo, IPSpaceSyncConfigureLocal, IPSpaceSyncConfigureS3, IPSpaceSyncConfigureSFTP, IPSpaceSyncConfigureGit, IPSpaceSyncPreview, IPSpaceSyncApply,
  } from "../../../wailsjs/go/app/App.js";
  import { requestedTool } from "../nav";

  let phase: "loading" | "create" | "locked" | "ready" = "loading";
  let invList: any[] = [], activeId = "", creatingInv = false, newName = "";
  let pass = "", pass2 = "", gateErr = "";
  let tree: any[] = [], history: any[] = [];
  let selected: any = null;
  let showAdd = false, addErr = "", addMsg = "";
  let form = { prefix: "", name: "", status: "active", tags: "", notes: "" };
  let editing = false, editForm = { name: "", status: "active", tags: "", notes: "" };
  let showCsv = false, importText = "", exportText = "", importReport: any = null, csvErr = "";

  // Sync state.
  let showSync = false, syncInfo: any = { configured: false }, syncPath = "";
  let backendKind: "local" | "s3" | "sftp" | "git" = "local";
  let s3f = { endpoint: "", region: "", bucket: "", key: "meridian/ipspace.enc", accessKey: "", secret: "" };
  let sf = { host: "", port: "22", user: "", password: "", privateKey: "", keyPassphrase: "", path: "" };
  let gf = { url: "", branch: "main", path: "ipspace.enc", username: "", token: "" };
  let syncResult: any = null, syncErr = "", syncMsg = "", syncBusy = false;
  $: syncChanges = syncResult ? (syncResult.incoming?.length || 0) + (syncResult.outgoing?.length || 0) : 0;

  // Flatten the tree into rows with a depth, for rendering.
  function flatten(nodes: any[], depth = 0, out: any[] = []): any[] {
    for (const n of nodes ?? []) {
      out.push({ rec: n.record, depth });
      flatten(n.children, depth + 1, out);
    }
    return out;
  }
  $: rows = flatten(tree);

  async function init() { await loadList(); }

  async function loadList() {
    invList = await IPSpaceList();
    activeId = await IPSpaceActive();
    if (invList.length === 0) { phase = "create"; creatingInv = false; return; }
    const s = await IPSpaceStatus();
    if (s.unlocked) { phase = "ready"; await refresh(); }
    else { phase = "locked"; selected = null; tree = []; history = []; }
  }

  async function switchTo(id: string) {
    if (id === activeId) return;
    gateErr = ""; pass = "";
    try { await IPSpaceSetActive(id); selected = null; await loadList(); }
    catch (e) { gateErr = String(e); }
  }

  async function create() {
    gateErr = "";
    if (!newName.trim()) { gateErr = "Give the inventory a name."; return; }
    if (pass.length < 6) { gateErr = "Use a passphrase of at least 6 characters."; return; }
    if (pass !== pass2) { gateErr = "The passphrases don't match."; return; }
    try {
      await IPSpaceCreateInventory(newName.trim(), pass);
      newName = ""; pass = pass2 = ""; creatingInv = false;
      await loadList();
    } catch (e) { gateErr = String(e); }
  }

  async function unlock() {
    gateErr = "";
    try { await IPSpaceUnlock(pass); pass = ""; await loadList(); }
    catch (e) { gateErr = String(e); }
  }

  async function renameActive() {
    const name = prompt("Rename inventory", activeName());
    if (!name || !name.trim()) return;
    try { await IPSpaceRenameInventory(activeId, name.trim()); await loadList(); } catch (e) { gateErr = String(e); }
  }
  async function removeActive() {
    if (!confirm(`Delete “${activeName()}” and all its records? This can't be undone.`)) return;
    try { await IPSpaceRemoveInventory(activeId); await loadList(); } catch (e) { gateErr = String(e); }
  }
  function activeName(): string { return (invList.find((i) => i.id === activeId) || {}).name || ""; }
  const CHIPS = ["var(--accent)", "#a78bfa", "#f0a85a"];
  function chipColor(id: string): string { const i = invList.findIndex((x) => x.id === id); return CHIPS[i % CHIPS.length]; }
  async function refresh() {
    tree = await IPSpaceTree();
    history = (await IPSpaceHistory()).slice().reverse();
    if (selected) selected = findRec(rows, selected.id);
    try { syncInfo = await IPSpaceSyncInfo(); } catch (e) { syncInfo = { configured: false }; }
  }

  async function configureSync() {
    syncErr = ""; syncMsg = ""; syncBusy = true;
    try {
      if (backendKind === "local") {
        await IPSpaceSyncConfigureLocal(syncPath.trim());
      } else if (backendKind === "s3") {
        await IPSpaceSyncConfigureS3(s3f.endpoint.trim(), s3f.region.trim(), s3f.bucket.trim(), s3f.key.trim(), s3f.accessKey.trim(), s3f.secret);
      } else if (backendKind === "sftp") {
        await IPSpaceSyncConfigureSFTP(sf.host.trim(), sf.port.trim(), sf.user.trim(), sf.password, sf.privateKey, sf.keyPassphrase, sf.path.trim());
      } else {
        await IPSpaceSyncConfigureGit(gf.url.trim(), gf.branch.trim(), gf.path.trim(), gf.username.trim(), gf.token);
      }
      syncInfo = await IPSpaceSyncInfo();
      syncResult = null;
    } catch (e) { syncErr = String(e); }
    finally { syncBusy = false; }
  }
  $: canConfigure =
    backendKind === "local" ? !!syncPath.trim() :
    backendKind === "s3" ? !!(s3f.bucket.trim() && s3f.key.trim() && s3f.accessKey.trim() && s3f.secret) :
    backendKind === "sftp" ? !!(sf.host.trim() && sf.user.trim() && sf.path.trim() && (sf.password || sf.privateKey.trim())) :
    !!(gf.url.trim() && gf.path.trim() && gf.token);
  async function previewSync() {
    syncErr = ""; syncMsg = ""; syncBusy = true; syncResult = null;
    try { syncResult = await IPSpaceSyncPreview(); }
    catch (e) { syncErr = String(e); }
    finally { syncBusy = false; }
  }
  async function applySync() {
    syncErr = ""; syncMsg = ""; syncBusy = true;
    try {
      const r = await IPSpaceSyncApply();
      if (r.applied) {
        const inc = r.incoming?.length || 0, out = r.outgoing?.length || 0;
        syncMsg = `Synced — pulled ${inc} change${inc === 1 ? "" : "s"}, pushed ${out} of your own.`;
        syncResult = null; // the diff is applied; clear it so the columns don't linger
        await refresh();
      } else {
        syncResult = r; // conflicts — keep them on screen to resolve
      }
    } catch (e) {
      // ErrRemoteChanged lands here: someone pushed while we were syncing.
      syncErr = String(e);
      syncResult = null;
    }
    finally { syncBusy = false; }
  }
  function changeLabel(c: any): string {
    const p = c.record?.prefix || c.prev?.prefix || c.record?.id || "";
    const n = c.record?.name || c.prev?.name || "";
    return `${c.kind} ${p}${n ? " — " + n : ""}`;
  }
  function findRec(list: any[], id: string) { return (list.find((r) => r.rec.id === id) || {}).rec || null; }
  function select(rec: any) { selected = rec; editing = false; }

  async function doAdd() {
    addErr = ""; addMsg = "";
    try {
      const tags = form.tags.split(",").map((s) => s.trim()).filter(Boolean);
      const rec = await IPSpaceAdd(form.prefix.trim(), form.name.trim(), form.status, tags, form.notes);
      addMsg = `Added ${rec.prefix}${rec.parentId ? " — nested under its enclosing block" : " at top level"}.`;
      form = { prefix: "", name: "", status: "active", tags: "", notes: "" };
      showAdd = false;
      await refresh();
    } catch (e) { addErr = String(e); }
  }
  function startEdit() {
    editing = true;
    editForm = { name: selected.name, status: selected.status, tags: (selected.tags || []).join(", "), notes: selected.notes };
  }
  async function saveEdit() {
    try {
      const tags = editForm.tags.split(",").map((s) => s.trim()).filter(Boolean);
      await IPSpaceEdit(selected.id, editForm.name, editForm.status, tags, editForm.notes);
      editing = false; await refresh();
    } catch (e) { addErr = String(e); }
  }
  async function del() { if (!selected) return; try { await IPSpaceDelete(selected.id); selected = null; await refresh(); } catch (e) { addErr = String(e); } }
  async function undo() { try { await IPSpaceUndo(); selected = null; await refresh(); } catch (e) {} }
  function pivot(tool: string) {
    if (!selected) return;
    // Lookup takes an IP/ASN (prefix pages come later), so pass the base address.
    const value = tool === "lookup" ? selected.prefix.split("/")[0] : selected.prefix;
    requestedTool.set({ tool, value });
  }

  async function openExport() { exportText = await IPSpaceExportCSV(); showCsv = true; importReport = null; csvErr = ""; }
  async function doImport() {
    csvErr = "";
    try { importReport = await IPSpaceImportCSV(importText); await refresh(); }
    catch (e) { csvErr = String(e); }
  }

  onMount(init);
</script>

<div class="tool">
  {#if phase === "loading"}
    <p class="muted">Loading…</p>

  {:else}
    {#if invList.length > 0}
      <div class="invbar">
        {#each invList as inv (inv.id)}
          <button class="invtab" class:on={inv.id === activeId} on:click={() => switchTo(inv.id)}>
            <span class="ichip" style={`background:${chipColor(inv.id)}`}></span>
            <span class="iname">{inv.name}</span>
            <span class="ilock">{inv.unlocked ? "🔓" : "🔒"}</span>
          </button>
        {/each}
        {#if invList.length < 3}
          <button class="invtab add" on:click={() => { creatingInv = true; gateErr = ''; newName = ''; pass = ''; pass2 = ''; }}>+ inventory</button>
        {:else}
          <span class="slotnote">3 of 3 slots</span>
        {/if}
      </div>
    {/if}

    {#if creatingInv || phase === "create"}
      <div class="gate">
        <h2>{invList.length === 0 ? "Create your first inventory" : "New inventory"}</h2>
        <p class="sub">A separate, isolated IP space — e.g. one company's. Encrypted on disk with its own passphrase. <b>There's no recovery</b> if you lose it.</p>
        <input class="field" placeholder="Name (e.g. Acme Corp)" bind:value={newName} />
        <input class="field" type="password" placeholder="Passphrase (≥ 6 chars)" bind:value={pass} />
        <input class="field" type="password" placeholder="Confirm passphrase" bind:value={pass2} on:keydown={(e) => e.key === "Enter" && create()} />
        {#if gateErr}<div class="err">{gateErr}</div>{/if}
        <div class="formbtns">
          <button class="btn" on:click={create}>Create inventory</button>
          {#if invList.length > 0}<button class="btn ghost" on:click={() => { creatingInv = false; gateErr = ''; }}>Cancel</button>{/if}
        </div>
      </div>

    {:else if phase === "locked"}
      <div class="gate">
        <h2>Unlock {activeName()}</h2>
        <p class="sub">Enter this inventory's passphrase.</p>
        <input class="field" type="password" placeholder="Passphrase" bind:value={pass} on:keydown={(e) => e.key === "Enter" && unlock()} />
        {#if gateErr}<div class="err">{gateErr}</div>{/if}
        <button class="btn" on:click={unlock}>Unlock</button>
      </div>

    {:else}
    <div class="toolbar">
      <button class="btn" on:click={() => { showAdd = !showAdd; addErr = ""; addMsg = ""; }}>+ Add block</button>
      <button class="btn ghost" on:click={undo} disabled={!history.length}>Undo</button>
      <button class="btn ghost" on:click={openExport}>CSV…</button>
      <button class="btn ghost" class:on={showSync} on:click={() => { showSync = !showSync; syncErr = ""; }}>Sync…{#if syncInfo.configured}<span class="dot"></span>{/if}</button>
      <button class="btn ghost" on:click={renameActive} title="Rename this inventory">Rename</button>
      <button class="btn ghost" on:click={removeActive} title="Delete this inventory">Delete</button>
      <span class="lock">🔓 {activeName()} · encrypted at rest</span>
    </div>

    {#if showSync}
      <div class="card synccard">
        <div class="chead"><h3>Sync — shared IP space</h3><button class="x" on:click={() => (showSync = false)}>✕</button></div>
        <p class="hint" style="margin:0 0 10px">
          Host one encrypted copy of this inventory somewhere your team can reach — a mounted
          share or synced folder, or an S3-compatible bucket (AWS S3, Cloudflare R2, and
          compatibles) — and everyone pulls and pushes to it. Changes stay on your machine until
          you sync. Everyone uses the same store passphrase for the shared copy.
        </p>

        <div class="segs">
          <button class="seg" class:on={backendKind === "local"} on:click={() => (backendKind = "local")}>Local / mounted path</button>
          <button class="seg" class:on={backendKind === "s3"} on:click={() => (backendKind = "s3")}>S3-compatible</button>
          <button class="seg" class:on={backendKind === "sftp"} on:click={() => (backendKind = "sftp")}>SFTP</button>
          <button class="seg" class:on={backendKind === "git"} on:click={() => (backendKind = "git")}>Git</button>
        </div>

        {#if backendKind === "local"}
          <div class="form" style="grid-template-columns:1fr auto;align-items:end;gap:8px">
            <label>Shared file path
              <input class="field mono" bind:value={syncPath} placeholder="/Volumes/share/meridian-ipspace.enc" />
            </label>
            <button class="btn" on:click={configureSync} disabled={!canConfigure || syncBusy}>Use this location</button>
          </div>
        {:else if backendKind === "s3"}
          <div class="form">
            <label>Bucket<input class="field mono" bind:value={s3f.bucket} placeholder="inventory" /></label>
            <label>Object key<input class="field mono" bind:value={s3f.key} placeholder="meridian/ipspace.enc" /></label>
            <label>Endpoint (blank for AWS S3)<input class="field mono" bind:value={s3f.endpoint} placeholder="https://&lt;acct&gt;.r2.cloudflarestorage.com" /></label>
            <label>Region<input class="field mono" bind:value={s3f.region} placeholder="auto (R2) · us-east-1 (AWS/MinIO)" /></label>
            <label>Access key ID<input class="field mono" bind:value={s3f.accessKey} autocomplete="off" /></label>
            <label>Secret access key<input class="field mono" type="password" bind:value={s3f.secret} autocomplete="off" /></label>
          </div>
          <div class="formbtns">
            <button class="btn" on:click={configureSync} disabled={!canConfigure || syncBusy}>{syncBusy ? "Checking…" : "Use this bucket"}</button>
            <span class="hint">AWS S3, Cloudflare R2, and S3-compatible storage that supports conditional writes — Meridian checks when you connect and won't sync somewhere two people could overwrite each other. Credentials are encrypted with your store passphrase, never written in the clear.</span>
          </div>
        {:else if backendKind === "sftp"}
          <div class="form">
            <label>Host<input class="field mono" bind:value={sf.host} placeholder="sftp.example.com" /></label>
            <label>Port<input class="field mono" bind:value={sf.port} placeholder="22" /></label>
            <label>Username<input class="field mono" bind:value={sf.user} autocomplete="off" /></label>
            <label>Remote file path<input class="field mono" bind:value={sf.path} placeholder="/srv/meridian/ipspace.enc" /></label>
            <label>Password<input class="field mono" type="password" bind:value={sf.password} autocomplete="off" placeholder="(or use a private key)" /></label>
            <label>Key passphrase (if key is encrypted)<input class="field mono" type="password" bind:value={sf.keyPassphrase} autocomplete="off" /></label>
            <label class="wide">Private key (PEM, optional)<textarea class="field mono small" bind:value={sf.privateKey} spellcheck="false" autocomplete="off" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"></textarea></label>
          </div>
          <div class="formbtns">
            <button class="btn" on:click={configureSync} disabled={!canConfigure || syncBusy}>{syncBusy ? "Connecting…" : "Use this server"}</button>
            <span class="hint">The server's host key is pinned the first time you connect; a later change is refused (to catch impersonation). Credentials are encrypted with your store passphrase, never written in the clear.</span>
          </div>
        {:else}
          <div class="form">
            <label>Repository URL (HTTPS)<input class="field mono" bind:value={gf.url} placeholder="https://github.com/acme/ipam.git" /></label>
            <label>Branch<input class="field mono" bind:value={gf.branch} placeholder="main" /></label>
            <label>File path in repo<input class="field mono" bind:value={gf.path} placeholder="ipspace.enc" /></label>
            <label>Username<input class="field mono" bind:value={gf.username} autocomplete="off" placeholder="git (or your username)" /></label>
            <label>Access token<input class="field mono" type="password" bind:value={gf.token} autocomplete="off" placeholder="personal access token" /></label>
          </div>
          <div class="formbtns">
            <button class="btn" on:click={configureSync} disabled={!canConfigure || syncBusy}>{syncBusy ? "Connecting…" : "Use this repo"}</button>
            <span class="hint">The inventory is a file committed and pushed each sync — its full history is kept. Authenticate with an HTTPS access token (GitHub/GitLab/Bitbucket/self-hosted). The repo must already exist with an initial commit. The token is encrypted with your store passphrase, never written in the clear.</span>
          </div>
        {/if}
        {#if syncInfo.configured}
          <div class="synced mono">Syncing through {syncInfo.location}</div>
          <div class="formbtns">
            <button class="btn" on:click={previewSync} disabled={syncBusy}>{syncBusy ? "Working…" : "Pull & review"}</button>
            <button class="btn ghost" on:click={applySync} disabled={syncBusy || (syncResult && syncResult.conflicts?.length)}>Sync now</button>
            <span class="hint">Review pulls the shared copy and shows the diff; Sync now applies it and pushes yours back.</span>
          </div>
        {/if}

        {#if syncErr}<div class="err" style="margin-top:10px">{syncErr}</div>{/if}
        {#if syncMsg}<div class="ok" style="margin:10px 0 0">{syncMsg}</div>{/if}

        {#if syncResult}
          {#if syncResult.conflicts?.length}
            <div class="conflict">
              <b>{syncResult.conflicts.length} conflict{syncResult.conflicts.length === 1 ? "" : "s"}</b> — these records were changed on both sides since your last sync. Sync is paused so nothing gets clobbered; resolve them (edit one side to match, or undo your change) and review again.
              <ul>
                {#each syncResult.conflicts as c}
                  <li class="mono">{c.prefix || c.id} — yours: “{c.localDeleted ? "deleted" : c.local?.name}” · theirs: “{c.remoteDeleted ? "deleted" : c.remote?.name}”</li>
                {/each}
              </ul>
            </div>
          {/if}
          {#if syncResult.duplicatePrefixes?.length}
            <div class="dup">Heads up: {syncResult.duplicatePrefixes.length} prefix{syncResult.duplicatePrefixes.length === 1 ? "" : "es"} now recorded by more than one entry (added independently on both sides).</div>
          {/if}
          {#if !syncResult.applied && !syncResult.conflicts?.length && syncChanges === 0}
            <div class="ok" style="margin-top:10px">Already in sync — nothing to pull or push.</div>
          {/if}
          <div class="diffcols">
            <div>
              <p class="lbl">Incoming ({syncResult.incoming?.length || 0}) — from the shared copy</p>
              {#if syncResult.incoming?.length}<ul class="difflist">{#each syncResult.incoming as c}<li class="mono i-{c.kind}">{changeLabel(c)}</li>{/each}</ul>{:else}<p class="muted">None.</p>{/if}
            </div>
            <div>
              <p class="lbl">Outgoing ({syncResult.outgoing?.length || 0}) — your local changes</p>
              {#if syncResult.outgoing?.length}<ul class="difflist">{#each syncResult.outgoing as c}<li class="mono i-{c.kind}">{changeLabel(c)}</li>{/each}</ul>{:else}<p class="muted">None.</p>{/if}
            </div>
          </div>
        {/if}
      </div>
    {/if}

    {#if addMsg}<div class="ok">{addMsg}</div>{/if}

    {#if showAdd}
      <div class="card addcard">
        <div class="form">
          <label>CIDR or address<input class="field mono" bind:value={form.prefix} placeholder="10.1.0.0/24  or  10.1.10.5" /></label>
          <label>Name<input class="field" bind:value={form.name} placeholder="HQ servers" /></label>
          <label>Status
            <select class="field" bind:value={form.status}><option>active</option><option>reserved</option><option>deprecated</option></select>
          </label>
          <label>Tags (comma)<input class="field" bind:value={form.tags} placeholder="prod, servers" /></label>
          <label class="wide">Notes<input class="field" bind:value={form.notes} /></label>
        </div>
        {#if addErr}<div class="err">{addErr}</div>{/if}
        <div class="formbtns">
          <button class="btn" on:click={doAdd}>Save</button>
          <button class="btn ghost" on:click={() => (showAdd = false)}>Cancel</button>
          <span class="hint">Overlaps nest automatically; duplicates are rejected. Saved as an undoable event.</span>
        </div>
      </div>
    {/if}

    {#if showCsv}
      <div class="card">
        <div class="chead"><h3>CSV</h3><button class="x" on:click={() => (showCsv = false)}>✕</button></div>
        <p class="lbl">Export (copy this):</p>
        <textarea class="field mono" rows="4" readonly>{exportText}</textarea>
        <p class="lbl">Import (paste rows: prefix,name,status,tags,notes):</p>
        <textarea class="field mono" rows="4" bind:value={importText}></textarea>
        <div class="formbtns"><button class="btn" on:click={doImport}>Import</button></div>
        {#if csvErr}<div class="err">{csvErr}</div>{/if}
        {#if importReport}<div class="ok">Imported {importReport.added}, skipped {importReport.skipped}.{#if importReport.errors?.length} ({importReport.errors.length} errors){/if}</div>{/if}
      </div>
    {/if}

    <div class="cols">
      <div class="card">
        <div class="chead"><h3>Your IP space</h3><span class="pill">{rows.length} records</span></div>
        {#if !rows.length}
          <p class="muted">No records yet — add a block to get started.</p>
        {:else}
          <div class="tree">
            {#each rows as r}
              <button class="node" class:sel={selected && selected.id === r.rec.id} style={`padding-left:${6 + r.depth * 18}px`} on:click={() => select(r.rec)}>
                <span class="cidr mono">{r.rec.prefix}</span>
                <span class="nm">{r.rec.name}</span>
                <span class="st pill {r.rec.status}">{r.rec.status}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <div style="display:flex;flex-direction:column;gap:14px">
        {#if selected}
          <div class="card">
            <div class="chead"><h3>{selected.prefix}</h3></div>
            <div class="acts">
              {#if !editing}<button class="btn ghost sm" on:click={startEdit}>Edit</button>{/if}
              <button class="btn ghost sm" on:click={() => { form.prefix = selected.prefix.split("/")[0] + "/"; showAdd = true; }}>Add child</button>
              <button class="btn ghost sm" on:click={() => pivot("subnet")}>Subnet calc ↗</button>
              <button class="btn ghost sm" on:click={() => pivot("lookup")}>Look up ↗</button>
              <button class="btn ghost sm danger" on:click={del}>Delete</button>
            </div>
            {#if editing}
              <div class="form" style="margin-top:4px">
                <label>Name<input class="field" bind:value={editForm.name} /></label>
                <label>Status<select class="field" bind:value={editForm.status}><option>active</option><option>reserved</option><option>deprecated</option></select></label>
                <label>Tags<input class="field" bind:value={editForm.tags} /></label>
                <label class="wide">Notes<input class="field" bind:value={editForm.notes} /></label>
              </div>
              <div class="formbtns"><button class="btn" on:click={saveEdit}>Save</button><button class="btn ghost" on:click={() => (editing = false)}>Cancel</button></div>
            {:else}
              <dl>
                <dt>Name</dt><dd>{selected.name || "—"}</dd>
                <dt>Status</dt><dd><span class="pill {selected.status}">{selected.status}</span></dd>
                <dt>Tags</dt><dd>{(selected.tags || []).join(" · ") || "—"}</dd>
                <dt>Added by</dt><dd class="mono">{selected.createdBy}</dd>
                <dt>Last change</dt><dd class="mono">{selected.changedBy}</dd>
                <dt>Notes</dt><dd>{selected.notes || "—"}</dd>
              </dl>
            {/if}
          </div>
        {/if}
        <div class="card">
          <div class="chead"><h3>Change log</h3><span class="pill">append-only</span></div>
          {#if !history.length}<p class="muted">No changes yet.</p>{:else}
            <div class="log">
              {#each history.slice(0, 12) as e}
                <div class="logrow"><span class="who mono">{e.actor}</span> <span class="act">{e.type} {e.record.prefix || e.record.id}</span></div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>
    {/if}
  {/if}
</div>

<style>
  .tool { padding: 14px 18px 40px; }
  .invbar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-bottom: 14px; }
  .invtab { display: inline-flex; align-items: center; gap: 8px; background: var(--card); border: 1px solid var(--border); border-radius: 9px; padding: 6px 11px; cursor: pointer; font: inherit; font-size: 0.84rem; color: var(--text-dim); }
  .invtab:hover { border-color: var(--field-border); }
  .invtab.on { color: var(--text-strong); background: var(--active); border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .invtab .ichip { width: 9px; height: 9px; border-radius: 50%; flex: none; }
  .invtab .iname { font-weight: 600; }
  .invtab .ilock { font-size: 0.74rem; }
  .invtab.add { border-style: dashed; color: var(--text-faint); }
  .slotnote { font-size: 0.72rem; color: var(--text-faint); }
  h2 { margin: 0 0 4px; font-size: 1.15rem; font-weight: 600; color: var(--text-strong); }
  .sub { color: var(--text-dim); font-size: 0.85rem; margin: 0 0 12px; }
  .muted { color: var(--text-dim); font-size: 0.85rem; }
  .gate { max-width: 420px; display: flex; flex-direction: column; gap: 10px; margin-top: 8px; }
  .field { background: var(--well); border: 1px solid var(--field-border); border-radius: 7px; padding: 8px 11px; color: var(--text); font-size: 0.88rem; outline: none; width: 100%; font-family: var(--font-sans); }
  .field.mono { font-family: var(--font-mono); }
  .field:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  textarea.field { resize: vertical; }
  .btn { background: var(--btn); border: 1px solid var(--btn-border); color: var(--on-accent); border-radius: 7px; padding: 7px 14px; font: inherit; font-size: 0.84rem; cursor: pointer; }
  .btn.ghost { background: var(--card); border-color: var(--field-border); color: var(--text); }
  .btn.sm { padding: 3px 10px; font-size: 0.78rem; }
  .btn.danger { color: var(--bad, #ff6b6b); border-color: transparent; }
  .btn:disabled { opacity: .5; cursor: default; }
  .toolbar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
  .lock { margin-left: auto; font-size: 0.74rem; color: var(--good, #5ad17a); border: 1px solid var(--border); border-radius: 20px; padding: 3px 11px; }
  .err { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 8px 11px; font-size: 0.82rem; }
  .ok { background: var(--good-bg, #102516); color: var(--good, #5ad17a); border-radius: 7px; padding: 7px 11px; font-size: 0.82rem; margin-bottom: 10px; }
  .card { background: var(--card); border: 1px solid var(--border-soft); border-radius: 10px; padding: 12px 14px; }
  .addcard { margin-bottom: 14px; }
  .chead { display: flex; align-items: center; justify-content: space-between; margin-bottom: 9px; }
  .chead h3 { margin: 0; font-size: 0.72rem; text-transform: uppercase; letter-spacing: .05em; color: var(--text-dim); font-weight: 600; }
  .x { background: none; border: none; color: var(--text-dim); cursor: pointer; font-size: 0.9rem; }
  .lbl { color: var(--text-dim); font-size: 0.72rem; margin: 8px 0 3px; }
  .form { display: grid; grid-template-columns: 1fr 1fr; gap: 10px 14px; }
  @media (max-width: 700px) { .form { grid-template-columns: 1fr; } }
  .form label { display: flex; flex-direction: column; gap: 4px; font-size: 0.66rem; text-transform: uppercase; letter-spacing: .04em; color: var(--text-dim); }
  .form .wide { grid-column: 1 / -1; }
  .formbtns { display: flex; align-items: center; gap: 8px; margin-top: 11px; flex-wrap: wrap; }
  .hint { color: var(--text-faint); font-size: 0.72rem; }
  .cols { display: grid; grid-template-columns: 1.4fr 1fr; gap: 14px; }
  @media (max-width: 900px) { .cols { grid-template-columns: 1fr; } }
  .tree { display: flex; flex-direction: column; }
  .node { display: flex; align-items: center; gap: 9px; width: 100%; text-align: left; background: none; border: none; color: var(--text); padding: 5px 6px; border-radius: 6px; cursor: pointer; font: inherit; }
  .node:hover { background: var(--hover); } .node.sel { background: var(--active); }
  .node .cidr { color: var(--text-strong); font-size: 0.82rem; }
  .node .nm { color: var(--text-dim); font-size: 0.82rem; }
  .node .st { margin-left: auto; }
  .acts { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 11px; }
  dl { display: grid; grid-template-columns: 96px 1fr; gap: 6px 12px; margin: 0; font-size: 0.85rem; }
  dt { color: var(--text-dim); } dd { margin: 0; color: var(--text); }
  .pill { display: inline-flex; font-size: 0.64rem; padding: 1px 8px; border-radius: 20px; border: 1px solid var(--border); color: var(--text-dim); }
  .pill.active { color: var(--good, #5ad17a); border-color: transparent; background: var(--good-bg, #102516); }
  .pill.reserved { color: var(--accent); border-color: var(--accent); }
  .pill.deprecated { color: var(--text-faint); }
  .btn.on { border-color: var(--accent); color: var(--accent); }
  .dot { display: inline-block; width: 6px; height: 6px; border-radius: 50%; background: var(--good, #5ad17a); margin-left: 6px; vertical-align: middle; }
  .synccard { margin-bottom: 14px; }
  .segs { display: inline-flex; border: 1px solid var(--field-border); border-radius: 7px; overflow: hidden; margin-bottom: 12px; }
  .seg { background: var(--card); border: none; border-right: 1px solid var(--field-border); color: var(--text-dim); padding: 6px 13px; font: inherit; font-size: 0.8rem; cursor: pointer; }
  .seg:last-child { border-right: none; }
  .seg.on { background: var(--active); color: var(--text-strong); }
  .form textarea.field.small { min-height: 72px; font-size: 0.74rem; line-height: 1.4; resize: vertical; }
  .synced { font-size: 0.78rem; color: var(--text-dim); margin: 10px 0; padding: 6px 10px; background: var(--well); border-radius: 6px; }
  .conflict { background: var(--warn-bg); border: 1px solid var(--warn-border); color: var(--warn-text); border-radius: 7px; padding: 9px 12px; font-size: 0.82rem; margin-top: 10px; }
  .conflict ul, .diffcols ul { margin: 6px 0 0; padding-left: 18px; }
  .conflict li { font-size: 0.78rem; margin: 2px 0; }
  .dup { color: var(--text-dim); font-size: 0.8rem; margin-top: 8px; }
  .diffcols { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-top: 12px; }
  @media (max-width: 700px) { .diffcols { grid-template-columns: 1fr; } }
  .difflist { list-style: none; margin: 4px 0 0; padding: 0; }
  .difflist li { font-size: 0.78rem; padding: 2px 0 2px 10px; border-left: 2px solid var(--border); color: var(--text-dim); }
  .difflist li.i-add { border-left-color: var(--good, #5ad17a); }
  .difflist li.i-edit { border-left-color: var(--accent); }
  .difflist li.i-delete { border-left-color: var(--bad, #ff6b6b); }
  .log { font-size: 0.8rem; }
  .logrow { display: flex; gap: 8px; align-items: baseline; padding: 4px 0; border-bottom: 1px solid var(--border-soft); }
  .logrow .who { color: var(--text-strong); } .logrow .act { color: var(--text-dim); }
  .mono { font-family: var(--font-mono); }
</style>
