package tfscan

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// securityAttrs are the attributes whose unresolved (variable/function) value
// makes a resource "dynamic" — a concrete answer needs a plan, not raw HCL.
var securityAttrs = map[string]bool{
	"cidr_blocks": true, "ipv6_cidr_blocks": true, "publicly_accessible": true,
	"acl": true, "encrypted": true, "storage_encrypted": true,
	"associate_public_ip_address": true,
}

// parseHCL parses raw Terraform HCL (one or more .tf files pasted together) into
// rawResources. Attribute values are read literally via a nil EvalContext;
// values from variables/functions/references are left unresolved.
func parseHCL(text string) ([]rawResource, error) {
	f, diags := hclsyntax.ParseConfig([]byte(text), "config.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("terraform HCL has a syntax error: %s", diags.Error())
	}
	body, ok := f.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("terraform HCL: unexpected body type")
	}
	var out []rawResource
	for _, blk := range body.Blocks {
		if blk.Type != "resource" || len(blk.Labels) < 2 {
			continue
		}
		out = append(out, rawResource{
			typ:     blk.Labels[0],
			name:    blk.Labels[1],
			file:    blk.DefRange().Filename,
			line:    blk.DefRange().Start.Line,
			dynamic: hclUnresolvedSecurity(blk.Body),
			body:    &hclBody{body: blk.Body},
		})
	}
	return out, nil
}

// hclUnresolvedSecurity reports whether any security-relevant attribute in the
// body (or a nested block) can't be resolved against a nil context.
func hclUnresolvedSecurity(body *hclsyntax.Body) bool {
	for name, attr := range body.Attributes {
		if securityAttrs[name] {
			if _, d := attr.Expr.Value(nil); d.HasErrors() {
				return true
			}
		}
	}
	for _, b := range body.Blocks {
		if hclUnresolvedSecurity(b.Body) {
			return true
		}
	}
	return false
}

// hclBody is a rawBody over an hclsyntax.Body.
type hclBody struct{ body *hclsyntax.Body }

func (h *hclBody) str(name string) (string, bool) {
	attr, ok := h.body.Attributes[name]
	if !ok {
		return "", false
	}
	v, d := attr.Expr.Value(nil)
	if d.HasErrors() {
		return "", false
	}
	return ctyToStr(v)
}

func (h *hclBody) list(name string) ([]string, bool) {
	attr, ok := h.body.Attributes[name]
	if !ok {
		return nil, false
	}
	// Element-wise for a literal list, so ["0.0.0.0/0", var.x] still yields the
	// literal while reporting the list as not fully resolved.
	if tup, ok := attr.Expr.(*hclsyntax.TupleConsExpr); ok {
		var out []string
		resolved := true
		for _, e := range tup.Exprs {
			v, d := e.Value(nil)
			if d.HasErrors() {
				resolved = false
				continue
			}
			if s, ok := ctyToStr(v); ok {
				out = append(out, s)
			}
		}
		return out, resolved
	}
	v, d := attr.Expr.Value(nil)
	if d.HasErrors() {
		return nil, false
	}
	if t := v.Type(); t.IsTupleType() || t.IsListType() || t.IsSetType() {
		var out []string
		for _, e := range v.AsValueSlice() {
			if s, ok := ctyToStr(e); ok {
				out = append(out, s)
			}
		}
		return out, true
	}
	if s, ok := ctyToStr(v); ok { // scalar coerces to a one-element list
		return []string{s}, true
	}
	return nil, false
}

func (h *hclBody) blocks(name string) []rawBody {
	var out []rawBody
	for _, b := range h.body.Blocks {
		if b.Type == name {
			out = append(out, &hclBody{body: b.Body})
		}
	}
	return out
}

// ctyToStr renders a known, non-null scalar cty value as a string.
func ctyToStr(v cty.Value) (string, bool) {
	if v.IsNull() || !v.IsKnown() {
		return "", false
	}
	switch v.Type() {
	case cty.String:
		return v.AsString(), true
	case cty.Bool:
		if v.True() {
			return "true", true
		}
		return "false", true
	case cty.Number:
		bf := v.AsBigFloat()
		if i, acc := bf.Int64(); acc == big.Exact {
			return strconv.FormatInt(i, 10), true
		}
		return bf.Text('f', -1), true
	}
	return "", false
}
