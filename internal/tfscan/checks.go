package tfscan

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// sensitivePorts are ports whose open-to-internet exposure is critical.
var sensitivePorts = map[int]string{
	22: "SSH", 23: "Telnet", 21: "FTP", 135: "RPC", 139: "NetBIOS", 445: "SMB",
	1433: "MSSQL", 3306: "MySQL", 3389: "RDP", 5432: "Postgres", 6379: "Redis",
	9200: "Elasticsearch", 27017: "MongoDB", 11211: "Memcached", 2379: "etcd",
}

// check runs the AWS exposure/misconfiguration checks over the parsed resources
// and returns a single Analysis (findings sorted by severity, counts filled).
// fixFor returns the concrete Terraform remediation for a finding title. Titles
// are stable strings defined in this file, so a map keeps the fix next to the
// problem without threading an extra argument through every add() call.
func fixFor(title string) string {
	switch title {
	case "Database is publicly accessible":
		return "Set publicly_accessible = false and place the instance in private subnets reached via VPC peering/VPN."
	case "Database storage encryption not confirmed":
		return "Set storage_encrypted = true (with a kms_key_id) so data is encrypted at rest."
	case "S3 bucket is public":
		return "Set acl = \"private\" and attach an aws_s3_bucket_public_access_block with all four block_public_* = true."
	case "S3 public access block is weakened":
		return "Set block_public_acls, block_public_policy, ignore_public_acls and restrict_public_buckets all = true."
	case "Instance gets a public IP":
		return "Set associate_public_ip_address = false; reach the instance via a bastion or SSM Session Manager in private subnets."
	case "EBS volume is not encrypted":
		return "Set encrypted = true on the volume (or enable EBS encryption-by-default for the account/region)."
	case "Firewall open to the internet":
		return "Replace the 0.0.0.0/0 CIDR with your office/VPN range or a referenced security group."
	case "Firewall exposes a sensitive port", "NSG rule exposes a sensitive port":
		return "Don't expose admin ports (SSH/RDP/DB) to the internet — use a bastion / SSM / Azure Bastion and scope the source to known IPs."
	case "Storage bucket is public": // GCS
		return "Remove the allUsers / allAuthenticatedUsers IAM member from the bucket binding."
	case "Cloud SQL open to the internet":
		return "Drop the 0.0.0.0/0 authorized network; use Private Service Connect or the Cloud SQL Auth Proxy."
	case "Instance gets an external IP":
		return "Remove the access_config {} block so no external IP is assigned; reach the VM via IAP or a bastion."
	case "NSG rule open to the internet":
		return "Restrict source_address_prefix from * / Internet to your specific IP ranges."
	case "Storage account permits public blobs":
		return "Set allow_nested_items_to_be_public = false on the storage account."
	case "Storage container is public":
		return "Set container_access_type = \"private\"."
	case "Managed database allows public network access":
		return "Set public_network_access_enabled = false and connect via a private endpoint."
	case "Database firewall allows the whole internet":
		return "Scope the firewall rule to known IPs — don't span 0.0.0.0–255.255.255.255."
	case "Public IP allocated":
		return "Only allocate a public IP when the resource must be internet-facing; otherwise remove it."
	}
	return ""
}

func check(resources []rawResource) Analysis {
	a := newAnalysis("")
	sgCount := 0

	expose := func(res *Resource, target, label, sev string) {
		res.Public = true
		res.Exposure = append(res.Exposure, label)
		a.Exposure = append(a.Exposure, ExposureEdge{Target: target, Label: label, Severity: sev})
	}
	add := func(sev, cat, title, detail string, r rawResource) {
		a.Findings = append(a.Findings, Finding{
			Severity: sev, Category: cat, Title: title, Detail: detail,
			Fix: fixFor(title), Resource: r.typ + "." + r.name, File: r.file, Line: r.line,
		})
	}

	for i := range resources {
		r := resources[i]
		key := r.typ + "." + r.name
		res := Resource{Type: r.typ, Name: r.name, Provider: r.provider, Dynamic: r.dynamic, File: r.file, Line: r.line}

		switch r.typ {
		case "aws_security_group":
			sgCount++
			for _, rule := range r.body.blocks("ingress") {
				if sev, label, dyn, open := ingressExposure(rule); open {
					res.Dynamic = res.Dynamic || dyn
					expose(&res, key, label, sev)
					add(sev, "exposure", "Security group open to the internet",
						fmt.Sprintf("%s allows %s from the internet (0.0.0.0/0)", key, label), r)
				}
			}
		case "aws_security_group_rule":
			if t, _ := r.body.str("type"); strings.EqualFold(t, "egress") {
				break
			}
			if sev, label, dyn, open := ingressExposure(r.body); open {
				res.Dynamic = res.Dynamic || dyn
				expose(&res, key, label, sev)
				add(sev, "exposure", "Security group rule open to the internet",
					fmt.Sprintf("%s allows %s from the internet (0.0.0.0/0)", key, label), r)
			}
		case "aws_db_instance", "aws_rds_cluster_instance", "aws_rds_cluster":
			if b, ok := r.body.str("publicly_accessible"); ok && b == "true" {
				expose(&res, key, "publicly_accessible", "critical")
				add("critical", "exposure", "Database is publicly accessible",
					key+" has publicly_accessible = true — it gets a public endpoint reachable from the internet", r)
			}
			// RDS defaults to unencrypted, so an absent storage_encrypted is also a
			// risk. Cluster *instances* inherit encryption from the cluster — skip them.
			if r.typ == "aws_db_instance" || r.typ == "aws_rds_cluster" {
				if enc, ok := r.body.str("storage_encrypted"); !ok || enc != "true" {
					add("warning", "encryption", "Database storage encryption not confirmed",
						key+" does not set storage_encrypted = true — RDS defaults to unencrypted storage at rest", r)
				}
			}
		case "aws_s3_bucket_acl", "aws_s3_bucket":
			if acl, ok := r.body.str("acl"); ok && strings.HasPrefix(acl, "public-") {
				expose(&res, key, acl, "critical")
				add("critical", "exposure", "S3 bucket is public",
					fmt.Sprintf("%s has acl = %q — objects are world-readable", key, acl), r)
			}
		case "aws_s3_bucket_public_access_block":
			for _, f := range []string{"block_public_acls", "block_public_policy", "ignore_public_acls", "restrict_public_buckets"} {
				if v, ok := r.body.str(f); ok && v == "false" {
					add("warning", "exposure", "S3 public access block is weakened",
						fmt.Sprintf("%s sets %s = false, allowing public access to be granted", key, f), r)
					break
				}
			}
		case "aws_instance":
			if v, ok := r.body.str("associate_public_ip_address"); ok && v == "true" {
				expose(&res, key, "public IP", "info")
				add("info", "exposure", "Instance gets a public IP",
					key+" sets associate_public_ip_address = true — confirm this tier is meant to be internet-facing", r)
			}
		case "aws_ebs_volume":
			if v, ok := r.body.str("encrypted"); ok && v == "false" {
				add("warning", "encryption", "EBS volume is not encrypted",
					key+" has encrypted = false", r)
			}

		// --- GCP ---
		case "google_compute_firewall":
			if d, ok := r.body.str("direction"); ok && strings.EqualFold(d, "EGRESS") {
				break
			}
			if src, _ := r.body.list("source_ranges"); !containsStr(src, "0.0.0.0/0") {
				break
			}
			for _, al := range r.body.blocks("allow") {
				proto, _ := al.str("protocol")
				ports, _ := al.list("ports")
				if len(ports) == 0 || strings.EqualFold(proto, "all") {
					expose(&res, key, "all ports", "critical")
					add("critical", "exposure", "Firewall open to the internet",
						key+" allows all ports from 0.0.0.0/0", r)
					continue
				}
				for _, p := range ports {
					from, to := portRange(p)
					if svc, pp := sensitiveInRange(from, to); pp != 0 {
						expose(&res, key, portLabel(proto, from, to)+" "+svc, "critical")
						add("critical", "exposure", "Firewall exposes a sensitive port",
							fmt.Sprintf("%s allows %s %s from the internet (0.0.0.0/0)", key, portLabel(proto, from, to), svc), r)
					} else {
						sev := "warning"
						if onlyWebPorts(from, to) {
							sev = "info"
						}
						expose(&res, key, portLabel(proto, from, to), sev)
						add(sev, "exposure", "Firewall open to the internet",
							fmt.Sprintf("%s allows %s from 0.0.0.0/0", key, portLabel(proto, from, to)), r)
					}
				}
			}
		case "google_storage_bucket_iam_member":
			if m, ok := r.body.str("member"); ok && (m == "allUsers" || m == "allAuthenticatedUsers") {
				expose(&res, key, "public ("+m+")", "critical")
				add("critical", "exposure", "Storage bucket is public",
					fmt.Sprintf("%s grants a role to %q — objects are world-readable", key, m), r)
			}
		case "google_storage_bucket_iam_binding":
			if ms, _ := r.body.list("members"); containsStr(ms, "allUsers") || containsStr(ms, "allAuthenticatedUsers") {
				expose(&res, key, "public", "critical")
				add("critical", "exposure", "Storage bucket is public",
					key+" binds a role to allUsers / allAuthenticatedUsers", r)
			}
		case "google_sql_database_instance":
			for _, s := range r.body.blocks("settings") {
				for _, ipc := range s.blocks("ip_configuration") {
					for _, an := range ipc.blocks("authorized_networks") {
						if v, ok := an.str("value"); ok && (v == "0.0.0.0/0" || v == "0.0.0.0") {
							expose(&res, key, "authorized 0.0.0.0/0", "critical")
							add("critical", "exposure", "Cloud SQL open to the internet",
								key+" authorizes 0.0.0.0/0 — the database is reachable from anywhere", r)
						}
					}
				}
			}
		case "google_compute_instance":
			for _, ni := range r.body.blocks("network_interface") {
				if len(ni.blocks("access_config")) > 0 {
					expose(&res, key, "external IP", "info")
					add("info", "exposure", "Instance gets an external IP",
						key+" has an access_config block — it gets a public IP", r)
					break
				}
			}

		// --- Azure ---
		case "azurerm_network_security_rule":
			acc, _ := r.body.str("access")
			dir, _ := r.body.str("direction")
			if !strings.EqualFold(acc, "Allow") || !strings.EqualFold(dir, "Inbound") {
				break
			}
			src, _ := r.body.str("source_address_prefix")
			srcs, _ := r.body.list("source_address_prefixes")
			if !(src == "*" || src == "0.0.0.0/0" || strings.EqualFold(src, "Internet") || containsStr(srcs, "0.0.0.0/0") || containsStr(srcs, "*")) {
				break
			}
			pr, _ := r.body.str("destination_port_range")
			from, to := portRange(pr)
			if pr == "*" {
				from, to = 0, 65535
			}
			switch {
			case from == 0 && to >= 65535:
				expose(&res, key, "all ports", "critical")
				add("critical", "exposure", "NSG rule open to the internet",
					key+" allows all ports inbound from the internet", r)
			default:
				if svc, pp := sensitiveInRange(from, to); pp != 0 {
					expose(&res, key, "port "+pr+" "+svc, "critical")
					add("critical", "exposure", "NSG rule exposes a sensitive port",
						fmt.Sprintf("%s allows %s (port %s) inbound from the internet", key, svc, pr), r)
				} else {
					sev := "warning"
					if onlyWebPorts(from, to) {
						sev = "info"
					}
					expose(&res, key, "port "+pr, sev)
					add(sev, "exposure", "NSG rule open to the internet",
						fmt.Sprintf("%s allows port %s inbound from the internet", key, pr), r)
				}
			}
		case "azurerm_storage_account":
			if v, ok := r.body.str("allow_nested_items_to_be_public"); ok && v == "true" {
				expose(&res, key, "public blobs allowed", "warning")
				add("warning", "exposure", "Storage account permits public blobs",
					key+" has allow_nested_items_to_be_public = true — containers can be made anonymously readable", r)
			}
		case "azurerm_storage_container":
			if v, ok := r.body.str("container_access_type"); ok && (v == "blob" || v == "container") {
				expose(&res, key, "public ("+v+")", "critical")
				add("critical", "exposure", "Storage container is public",
					fmt.Sprintf("%s has container_access_type = %q — blobs are anonymously readable", key, v), r)
			}
		case "azurerm_postgresql_server", "azurerm_mysql_server", "azurerm_mariadb_server", "azurerm_mssql_server", "azurerm_sql_server":
			if v, ok := r.body.str("public_network_access_enabled"); ok && v == "true" {
				expose(&res, key, "public network access", "critical")
				add("critical", "exposure", "Managed database allows public network access",
					key+" has public_network_access_enabled = true — the database endpoint is reachable from the internet", r)
			}
		case "azurerm_sql_firewall_rule", "azurerm_mssql_firewall_rule", "azurerm_postgresql_firewall_rule", "azurerm_mysql_firewall_rule":
			start, _ := r.body.str("start_ip_address")
			end, _ := r.body.str("end_ip_address")
			if start == "0.0.0.0" && end == "255.255.255.255" {
				expose(&res, key, "firewall 0.0.0.0–255.255.255.255", "critical")
				add("critical", "exposure", "Database firewall allows the whole internet",
					key+" allows 0.0.0.0–255.255.255.255 — the database is reachable from anywhere", r)
			}
		case "azurerm_public_ip":
			expose(&res, key, "public IP", "info")
			add("info", "exposure", "Public IP allocated",
				key+" allocates a public IP — confirm what it fronts", r)
		}

		a.Resources = append(a.Resources, res)
	}

	// counts
	exposed, critical := 0, 0
	for _, res := range a.Resources {
		if res.Public {
			exposed++
		}
	}
	for _, f := range a.Findings {
		if f.Severity == "critical" {
			critical++
		}
	}
	a.Counts = Counts{Resources: len(a.Resources), SecurityGroups: sgCount, Exposed: exposed, Critical: critical, Findings: len(a.Findings)}

	rank := map[string]int{"critical": 0, "warning": 1, "info": 2}
	sort.SliceStable(a.Findings, func(i, j int) bool {
		if rank[a.Findings[i].Severity] != rank[a.Findings[j].Severity] {
			return rank[a.Findings[i].Severity] < rank[a.Findings[j].Severity]
		}
		return a.Findings[i].Resource < a.Findings[j].Resource
	})
	return a
}

// ingressExposure evaluates one ingress rule body (inline block or a
// *_security_group_rule). It reports the severity, a human port label, whether
// a value was unresolved, and whether the rule is open to the internet.
func ingressExposure(rule rawBody) (sev, label string, dynamic, open bool) {
	v4, v4res := rule.list("cidr_blocks")
	v6, _ := rule.list("ipv6_cidr_blocks")
	if !containsStr(v4, "0.0.0.0/0") && !containsStr(v6, "::/0") {
		return "", "", false, false // no cidr (e.g. SG-ref ingress) or not world-open
	}
	dynamic = !v4res && len(v4) > 0 // literals present but list didn't fully resolve

	proto, _ := rule.str("protocol")
	fromS, fromOK := rule.str("from_port")
	toS, toOK := rule.str("to_port")
	from, to := atoi(fromS), atoi(toS)

	switch {
	case proto == "-1" || proto == "all" || (fromOK && toOK && from == 0 && to >= 65535):
		return "critical", "all ports", dynamic, true
	default:
		if svc, p := sensitiveInRange(from, to); p != 0 {
			return "critical", portLabel(proto, from, to) + " " + svc, dynamic, true
		}
		if onlyWebPorts(from, to) {
			return "info", portLabel(proto, from, to), dynamic, true
		}
		return "warning", portLabel(proto, from, to), dynamic, true
	}
}

// sensitiveInRange returns the service name and the lowest sensitive port the
// [from,to] range covers, or ("",0) if none.
func sensitiveInRange(from, to int) (string, int) {
	if from > to {
		from, to = to, from
	}
	best, bestName := 0, ""
	for p, name := range sensitivePorts {
		if p >= from && p <= to && (best == 0 || p < best) {
			best, bestName = p, name
		}
	}
	return bestName, best
}

// onlyWebPorts reports whether every port in [from,to] is 80 or 443.
func onlyWebPorts(from, to int) bool {
	if from > to {
		from, to = to, from
	}
	if to-from > 1024 {
		return false
	}
	for p := from; p <= to; p++ {
		if p != 80 && p != 443 {
			return false
		}
	}
	return true
}

func portLabel(proto string, from, to int) string {
	if proto == "" {
		proto = "tcp"
	}
	if from == to {
		return fmt.Sprintf("%s/%d", proto, from)
	}
	return fmt.Sprintf("%s/%d-%d", proto, from, to)
}

func containsStr(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// portRange parses a GCP/Azure port spec: "22" (single) or "8000-9000" (range).
func portRange(s string) (from, to int) {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '-'); i >= 0 {
		return atoi(s[:i]), atoi(s[i+1:])
	}
	n := atoi(s)
	return n, n
}
