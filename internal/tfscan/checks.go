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
			Resource: r.typ + "." + r.name, File: r.file, Line: r.line,
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
			sgCount++
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
			if enc, ok := r.body.str("storage_encrypted"); ok && enc == "false" {
				add("warning", "encryption", "Database storage is not encrypted",
					key+" has storage_encrypted = false — data at rest is unencrypted", r)
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
	from := atoi(firstOK(rule.str("from_port")))
	to := atoi(firstOK(rule.str("to_port")))

	switch {
	case proto == "-1" || proto == "all" || (from == 0 && (to == 0 || to >= 65535)):
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

func firstOK(s string, _ bool) string { return s }

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
