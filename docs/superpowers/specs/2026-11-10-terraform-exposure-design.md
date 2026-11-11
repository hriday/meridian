# Terraform exposure analyzer — design

**Date:** 2026-11-10
**Status:** design (awaiting builder-check, then plan)

## Intent & why

Meridian reads network/firewall/web configs offline and surfaces what's exposed.
The owner asked for the same lens on **Terraform**: paste a config and see what the
infrastructure exposes to the internet — security groups open to `0.0.0.0/0`, public
buckets and databases, unencrypted storage — with a **map of what's reachable from the
outside**, plus a findings list and a resource table. The approved mockup
(claude.ai/artifact/7p8V3Rq53htc8JbEG3dAnk) sets the shape: count tiles, an
internet-exposure diagram, severity-sorted findings, and a resource table.

This is a *security reading* of infrastructure-as-code, offline, same posture as the
other analyzers — never connects to a cloud account, never runs Terraform.

## Goals

- Accept **both** input forms (owner's choice): raw **HCL `.tf`** (parsed with the
  maintained `github.com/hashicorp/hcl/v2`) and a **`terraform show -json` plan**
  (parsed with `encoding/json`). Auto-detect which was pasted.
- Extract cloud resources into a neutral model and run a **findings engine** that flags
  internet exposure and common misconfigurations, focused on **AWS** first (the
  highest-value, most-common surface; Azure/GCP noted as follow-on).
- Derive an **exposure graph**: an "Internet (0.0.0.0/0)" node with edges to the
  resources reachable from it, labeled by how (port, public ACL, public endpoint) and
  colored by severity.
- A new **Terraform exposure** tool under *Config insight* in the UI, matching the
  mockup and the existing analyzers' look.

## Non-goals / anti-goals

- Not a Terraform validator, linter, or `terraform plan` substitute; not policy-as-code
  (no OPA/Sentinel). It's a focused exposure read.
- Not full multi-cloud parity this slice — AWS resource types drive the checks; other
  providers' resources are listed in the table but not deeply analyzed (noted).
- Not variable/module resolution for HCL. A value that comes from `var.x`, a `local`, a
  module output, or a function is reported as **dynamic (unresolved)** rather than
  guessed — the plan-JSON path is the way to get resolved values, and the UI says so.
- No cloud API calls, no state file reading (state can hold secrets).

## Input detection

`Analyze(text, source)` where source is `"hcl"`, `"json"`, or `""` (auto):
- auto: if the first non-space byte is `{` → JSON plan; else HCL.
- JSON: parse the `terraform show -json` shape — prefer `planned_values.root_module`
  (+ `child_modules` recursively) for resources with resolved `values`; fall back to
  `values.root_module` (state show) if present.
- HCL: `hclsyntax.ParseConfig` each pasted file; walk `resource` blocks.

## Model

```
Resource{ Type, Name, Provider string; Public bool; Exposure []string; // human labels
          Attrs map[string]string; // selected resolved attrs for the drill-in
          Dynamic bool;  // any security-relevant value was unresolved (HCL vars)
          File string; Line int }
Finding{ Severity string; // critical | warning | info
         Category string; // exposure | encryption | best-practice
         Title, Detail, Resource, File string; Line int }
ExposureEdge{ Target string; // "aws_db_instance.main"
              Label string;   // "tcp/22 SSH", "public-read", "publicly_accessible"
              Severity string }
Counts{ Resources, SecurityGroups, Exposed, Critical, Findings int }
Analysis{ Source string; Resources []Resource; Findings []Finding;
          Exposure []ExposureEdge; Counts Counts; Notes []string }
```

All slices initialized non-nil (JSON emits `[]`). No secret value is ever placed in
`Attrs`, a `Finding`, or an `Exposure` label (see Review focus).

## AWS checks (findings + exposure edges)

| Resource / attribute | Finding | Exposure edge |
|----------------------|---------|---------------|
| `aws_security_group` / `aws_security_group_rule` ingress with `cidr_blocks`/`cidr_ipv6` containing `0.0.0.0/0` or `::/0` | critical for port 22/3389/3306/5432/6379/27017/9200/23/etc (sensitive), warning for other ports, info for 80/443 | Internet → the SG, label `tcp/<port> <svc>` |
| `aws_db_instance` / `aws_rds_cluster_instance` `publicly_accessible = true` | critical | Internet → the db, label `publicly_accessible` |
| `aws_s3_bucket_acl` `acl = public-read`/`public-read-write`, or legacy `aws_s3_bucket` `acl = public-*` | critical | Internet → the bucket, label `public-read` |
| `aws_s3_bucket_public_access_block` any of the four `block_*` = false | warning | — |
| `aws_instance`/`aws_lb` `associate_public_ip_address = true` / `aws_eip` / public `aws_lb` | info (exposed by design; confirm) | Internet → the instance/lb, label `public IP` |
| `aws_db_instance`/`aws_rds_cluster` `storage_encrypted = false` (or absent) | warning (encryption) | — |
| `aws_ebs_volume` `encrypted = false` | warning (encryption) | — |

An SG's exposure is only drawn to the resources that *use* it when that linkage is
resolvable (an `aws_instance` referencing the SG); otherwise the SG itself is the
exposed node (as in the mockup). A `Dynamic` resource whose security value was a
variable gets an info note ("value is dynamic — analyze a plan for a concrete result"),
not a false finding.

## File structure

- `internal/tfscan/model.go` — the model + `newAnalysis`.
- `internal/tfscan/hcl.go` + `hcl_test.go` — `parseHCL(text) ([]rawResource, error)` using hclsyntax; literal attribute extraction with dynamic-marking.
- `internal/tfscan/plan.go` + `plan_test.go` — `parsePlanJSON(text) ([]rawResource, error)`.
- `internal/tfscan/checks.go` + `checks_test.go` — the AWS checks + exposure derivation over the neutral `rawResource` form (shared by both parsers), filling Findings/Exposure/Counts.
- `internal/tfscan/analyze.go` — `Analyze(text, source string) (Analysis, error)` with auto-detect + dispatch.
- `internal/app/tfscan_methods.go` (+ test) — `AnalyzeTerraform(text, source string) (tfscan.Analysis, error)`.
- `frontend/src/lib/tools/TerraformExposure.svelte` — the tool (source selector, tiles, exposure SVG diagram, findings, resource table), per the mockup.
- `frontend/src/lib/Workbench.svelte` — register under *Config insight*.
- Bindings regenerated.

`rawResource` is the bridge: `{ Type, Name, Provider, File string; Line int; Dynamic bool; attr(path) (value, resolved) }` — both parsers produce it, `checks.go` consumes it, so the AWS checks are written once.

## Review focus (inputs the happy-path tests won't cover)

- **HCL with variables/locals/modules**: `cidr_blocks = var.allow` or a module call must
  NOT crash and must NOT emit a false `0.0.0.0/0` finding — it's `Dynamic`, noted.
- **Malformed HCL / JSON**: a partial paste, an HCL syntax error, a non-plan JSON object
  → a clear error, never a panic.
- **Secret leakage**: `aws_db_instance.password`, `user_data`, access keys in a
  provider block or variable default must never reach `Attrs`/a finding/a label.
- **IPv6 `::/0`** open ingress is treated the same as `0.0.0.0/0`.
- **`terraform show -json` nesting**: resources inside `child_modules`, and the
  `resource_changes` vs `planned_values` shapes — resources must not be missed or
  double-counted.
- **SG rule as a separate resource** (`aws_security_group_rule`) vs inline `ingress`
  blocks — both must be analyzed.

## Builder-check resolutions (settled calls)

1. **The bridge is a `rawBody`, not a flat `attr(path)`.** Repeated blocks and lists
   can't be a single-value path, so define:
   ```
   type rawBody interface {
     str(name string)  (string, bool)   // scalar (string/bool/number→string), resolved?
     list(name string) ([]string, bool) // list of strings; a scalar coerces to 1 elem
     blocks(name string) []rawBody       // repeated sub-blocks (HCL ingress{}, JSON array)
   }
   type rawResource struct { typ, name, provider, file string; line int; body rawBody }
   ```
   Both parsers produce `[]rawResource`; `checks.go` does `sg.body.blocks("ingress")` and
   reads each rule's `from_port`/`to_port`/`cidr_blocks` via that sub-body. This is the
   one accessor set every check shares.
2. **`list` evaluates element-wise** to avoid a false negative. For HCL, when the attr
   is a `*hclsyntax.TupleConsExpr`, iterate `.Exprs`, `Value(nil)` each, keep the
   literals, and set `resolved=false` if any element was dynamic — but still return the
   literal elements. So `["0.0.0.0/0", var.extra]` returns `["0.0.0.0/0"]`, resolved=false:
   the open-to-internet rule is still flagged (true positive) and the resource is marked
   `Dynamic`. A wholly-dynamic value returns `(nil, false)`.
3. **SG rule semantics (corrected):**
   - IPv6 field is `ipv6_cidr_blocks` (not `cidr_ipv6`). Scan `cidr_blocks` for
     `0.0.0.0/0` and `ipv6_cidr_blocks` for `::/0`.
   - Ports are always a `from_port`/`to_port` **range**; there is no single `port`.
     Severity: `protocol = "-1"` / `"all"`, or a range of `0`–`65535` → **critical**
     (everything open to the world); else a range that *contains* a sensitive port
     (22, 23, 21, 135, 139, 445, 1433, 3306, 3389, 5432, 6379, 9200, 27017) → **critical**;
     a range that is only 80 and/or 443 → **info**; any other open range → **warning**.
   - An ingress referencing another SG (`source_security_group_id` / `security_groups`,
     no cidr) is **skipped** — it's not internet exposure.
   - Only **ingress** is analyzed: inline `egress` blocks and
     `aws_security_group_rule` with `type = "egress"` are excluded.
4. **No SG→instance linkage this slice.** Each exposed thing is its own exposure node:
   Internet → the SG (for open ingress), Internet → the bucket / db / public-IP instance
   (for the self-contained public resources). No reference resolution between resources;
   instances are listed in the resource table. (Revisits the mockup's "via sg web" edge —
   simpler and accurate; a later slice can add linkage.)
5. **Single `Analysis`** (confirmed) — like webconf; no per-context splitting.

Also settled: parse a multi-file HCL paste as one body (note cross-file refs are
unresolved); one exposure edge per sensitive open port on an SG, capped with a "+N more"
beyond a few.
