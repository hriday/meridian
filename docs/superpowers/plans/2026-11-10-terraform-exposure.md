# Terraform Exposure Analyzer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A new `internal/tfscan` package + UI tool that reads Terraform (HCL `.tf` or `terraform show -json`) offline and surfaces internet exposure and common misconfigurations, with an exposure graph, findings, and a resource table.

**Architecture:** Both parsers produce `[]rawResource`, where each resource carries a `rawBody` (`str`/`list`/`blocks` accessors). One `checks.go` engine runs the AWS checks over `[]rawResource`, filling a single `Analysis`. `analyze.go` auto-detects HCL vs JSON and dispatches. A Wails binding + Svelte tool render it per the approved mockup.

**Tech Stack:** Go 1.23, `github.com/hashicorp/hcl/v2` v2.20.1 (hclsyntax), `encoding/json`, Svelte.

**Spec:** `docs/superpowers/specs/2026-11-10-terraform-exposure-design.md`

## Global Constraints

- Offline only; never connect to a cloud or read state files.
- No secret value (passwords, keys, `user_data`) in any `Attrs`, `Finding`, or exposure label — the checks read only the whitelisted security attributes below.
- A value from a variable/local/module/function is `Dynamic` (noted), never guessed; `list` still returns any literal elements present (element-wise).
- Single `Analysis` return; all slices non-nil (JSON emits `[]`).
- Fixtures reflect real `.tf` / real `terraform show -json` structure (owner rule).

## Review Focus

- **HCL with variables** (`cidr_blocks = var.x`): no crash, no false `0.0.0.0/0` finding; resource marked Dynamic. (Task 2.)
- **Partially-dynamic list** (`["0.0.0.0/0", var.x]`): the open rule IS flagged. (Task 1 via mapBody + Task 2 via HCL.)
- **Malformed HCL / non-plan JSON**: clear error, no panic. (Task 2/3/4.)
- **Secret leakage**: `password`/`user_data`/keys never surfaced. (Task 2/3 secret test.)
- **IPv6 `::/0`**, port **ranges** (0-65535, protocol `-1`), **egress excluded**, **SG-ref ingress skipped**. (Task 1.)
- **JSON nesting**: resources in `child_modules`; not missed/double-counted. (Task 3.)

---

### Task 1: Model, `rawBody` bridge, and the checks engine

**Files:** Create `internal/tfscan/model.go`, `internal/tfscan/checks.go`, `internal/tfscan/checks_test.go`.

**Interfaces:**
- Produces:
  - `rawBody` interface: `str(name)(string,bool)`, `list(name)([]string,bool)`, `blocks(name)[]rawBody`.
  - `mapBody` — a `rawBody` over `map[string]any` (shared with the JSON parser and the tests).
  - `rawResource struct{ typ,name,provider,file string; line int; dynamic bool; body rawBody }`.
  - `check(resources []rawResource) Analysis` — runs all AWS checks, fills Findings/Exposure/Counts, sorts findings by severity.
  - model types (`Resource`, `Finding`, `ExposureEdge`, `Counts`, `Analysis`, `newAnalysis`).
- Consumes: nothing (leaf).

Checks (per the design's corrected table): `aws_security_group`(+`_rule`) ingress → severity by port range/protocol, scanning `cidr_blocks`(0.0.0.0/0) + `ipv6_cidr_blocks`(::/0); `aws_db_instance`/`aws_rds_cluster_instance` `publicly_accessible`; `aws_s3_bucket_acl`/`aws_s3_bucket` `acl` public; `aws_s3_bucket_public_access_block` block_* false; `associate_public_ip_address`/`aws_eip`; `storage_encrypted`/`aws_ebs_volume.encrypted` false. Each exposed resource → an `ExposureEdge{Target, Label, Severity}` (Internet → resource).

- [ ] **Step 1: Write failing tests** in `checks_test.go` building `[]rawResource` with `mapBody` literals (no parser):
  - `TestCheckOpenSSH`: SG with an ingress block (from_port 22,to_port 22, cidr_blocks [0.0.0.0/0]) → one critical finding + an exposure edge labeled with 22.
  - `TestCheckAllPortsCritical`: ingress protocol "-1" (or 0-65535) → critical "all ports".
  - `TestCheckWebPortsInfo`: ingress 443 from 0.0.0.0/0 → info, not critical.
  - `TestCheckEgressAndSgRefSkipped`: an egress 0.0.0.0/0 block and an ingress with only `security_groups` (no cidr) → no exposure finding.
  - `TestCheckPublicRDSAndBucket`: `publicly_accessible=true` → critical; `aws_s3_bucket_acl acl=public-read` → critical; each an exposure edge.
  - `TestCheckUnencrypted`: `storage_encrypted=false` and `aws_ebs_volume encrypted=false` → warning (encryption), NO exposure edge.
  - `TestCheckPartialDynamicListFlagged`: SG ingress whose `cidr_blocks` list-accessor returns `(["0.0.0.0/0"], false)` → still flagged critical, resource Dynamic.
  - `TestCheckCountsAndSort`: counts correct; findings sorted critical→warning→info.
  - `TestCheckNoSecretsSurfaced`: a resource whose body has a `password`/`user_data` key → that value appears in no finding/edge/Attrs (JSON-marshal grep).
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `model.go` (+ `mapBody`) and `checks.go`.
- [ ] **Step 4: Run, verify pass.**
- [ ] **Step 5: Commit** (`feat(tfscan): exposure/misconfig checks engine + model`).

---

### Task 2: HCL parser

**Files:** Create `internal/tfscan/hcl.go`, `internal/tfscan/hcl_test.go`.

**Interfaces:**
- Produces: `parseHCL(text string) ([]rawResource, error)`; an `hclBody` implementing `rawBody` with element-wise `list` (iterate `*hclsyntax.TupleConsExpr.Exprs`, `Value(nil)` each, keep literals, `resolved=false` on any dynamic element) and dynamic-aware `str`; `blocks(name)` → repeated `*hclsyntax.Block` wrapped as `hclBody`.
- Consumes: `rawResource`, `check` (end-to-end test).

- [ ] **Step 1: Write failing tests** in `hcl_test.go` against a realistic multi-resource `.tf` fixture (an SG with two `ingress` blocks — one literal `0.0.0.0/0:22`, one `cidr_blocks = var.allowed`; a public `aws_db_instance` with a `password`; an `aws_s3_bucket_acl` public-read):
  - `TestHCLParsesResources`: resource count, types/names, line numbers.
  - `TestHCLLiteralAndDynamic`: the literal ingress is read; the `var.allowed` one marks the SG Dynamic and raises no false finding (run `check` on the parsed resources).
  - `TestHCLEndToEndFindings`: `check(parseHCL(fixture))` → critical SSH + critical public RDS + critical public bucket.
  - `TestHCLSecretNotSurfaced`: the db `password` value is in no finding (end-to-end JSON marshal).
  - `TestHCLMalformed`: a syntax-broken paste → error, no panic.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `hcl.go`.
- [ ] **Step 4: Run, verify pass** + full package.
- [ ] **Step 5: Commit** (`feat(tfscan): HCL (.tf) parser`).

---

### Task 3: Plan JSON parser

**Files:** Create `internal/tfscan/plan.go`, `internal/tfscan/plan_test.go`.

**Interfaces:**
- Produces: `parsePlanJSON(text string) ([]rawResource, error)` — decode, walk `planned_values.root_module` (resources + recursive `child_modules`), falling back to `values.root_module`; each resource's `values` map wrapped as `mapBody` (from Task 1); `type`/`name`/`provider_name` fields populate the rawResource.
- Consumes: `mapBody`, `rawResource`, `check`.

- [ ] **Step 1: Write failing tests** in `plan_test.go` against a realistic `terraform show -json` fixture (root module + one child_module; an SG with an `ingress` array open to 0.0.0.0/0; a public RDS; a `password` value):
  - `TestPlanParsesNestedModules`: resources from root AND child_module both present, none double-counted.
  - `TestPlanEndToEndFindings`: `check(parsePlanJSON(fixture))` → the expected criticals.
  - `TestPlanSecretNotSurfaced`: password not surfaced.
  - `TestPlanNonPlanJSON`: a JSON object that isn't a plan (`{"foo":1}`) → empty resources or a clear error, no panic.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `plan.go`.
- [ ] **Step 4: Run, verify pass** + full package.
- [ ] **Step 5: Commit** (`feat(tfscan): terraform show -json plan parser`).

---

### Task 4: Analyze dispatch + app binding

**Files:** Create `internal/tfscan/analyze.go`; create `internal/app/tfscan_methods.go`, `internal/app/tfscan_methods_test.go`.

**Interfaces:**
- Produces: `Analyze(text, source string) (Analysis, error)` — auto-detect (first non-space `{` → JSON else HCL) when source is ""; dispatch to `parseHCL`/`parsePlanJSON` then `check`; set `Analysis.Source`. Binding `func (a *App) AnalyzeTerraform(text, source string) (tfscan.Analysis, error)`.
- Consumes: Task 1-3.

- [ ] **Step 1: Write failing tests** (`analyze_test.go` + `tfscan_methods_test.go`): auto-detect picks JSON for a `{`-leading input and HCL otherwise; empty input → error; `AnalyzeTerraform` delegates and returns a populated Analysis for an HCL fixture.
- [ ] **Step 2: Run, verify fail.**
- [ ] **Step 3: Implement** `analyze.go` + binding.
- [ ] **Step 4: Run, verify pass** + `go test ./...`.
- [ ] **Step 5: Commit** (`feat(app): AnalyzeTerraform binding + dispatch`).

---

### Task 5: UI tool

**Files:** Create `frontend/src/lib/tools/TerraformExposure.svelte`; modify `frontend/src/lib/Workbench.svelte`; regenerate bindings.

**Interfaces:** Consumes `AnalyzeTerraform(text, source)`.

Per the approved mockup: source selector (HCL / plan JSON, auto by default), count tiles (Resources, Security groups, Internet-exposed, Critical, Findings), an SVG internet-exposure diagram (Internet node → exposed resources, colored by severity, reusing the FirewallAnalyzer diagram approach), severity-sorted findings, and a resource table. A "dynamic values — analyze a plan for concrete results" note when any resource is Dynamic.

- [ ] **Step 1: Implement** the Svelte tool; register under *Config insight* (`{ id:"terraform", name:"Terraform exposure", group:"Config insight", ready:true }`).
- [ ] **Step 2: Build** `npm run build` → no errors.
- [ ] **Step 3: Regenerate bindings** (`$(go env GOPATH)/bin/wails generate module`; `git checkout -- frontend/wailsjs/runtime`).
- [ ] **Step 4: Manual check** via the app (Playwright): paste an HCL fixture, confirm tiles, diagram, findings, table render.
- [ ] **Step 5: Commit** (`feat(app,ui): Terraform exposure tool`).

---

## Self-review notes

- Spec coverage: checks+model (T1), HCL (T2), plan JSON (T3), dispatch+binding (T4), UI (T5).
- Review Focus: dynamic/partial-dynamic (T1,T2), malformed (T2,T3), secrets (T1,T2,T3), IPv6/ranges/egress/sg-ref (T1), JSON nesting (T3).
- Types consistent: both parsers → `[]rawResource` → `check` → `Analysis`; `mapBody` shared by T1/T3.
- Proportion: plan gives interfaces, the checks table (in the spec), and test names/assertions; parser bodies are the implementer's.
