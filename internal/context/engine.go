// Package context resolves cursor locations into semantic action proposals.
package context

import (
	"fmt"
	"github.com/Agrim-P777/qubex/internal/query"
	"strings"
)

type Cursor struct {
	View  string
	Path  query.Path
	Value any
}
type Action struct {
	ID, Name, Description string
	Arguments             []string
}
type ActionProposal struct {
	Scope       query.Path
	Operations  []query.Operation
	Description string
	NeedsScope  bool
	Scopes      []query.Path
}
type Resolution struct {
	Relative query.Path
	Mapped   bool
	Reason   string
	Scopes   []query.Path
}

// Propose returns an atomic semantic transaction. No UI adapter builds jq.
func Propose(s query.StageContext, request query.Operation, auxiliaryID string) ActionProposal {
	proposal := ActionProposal{Scope: s.Origin, Description: request.Kind + " at " + request.Path.String()}
	switch request.Kind {
	case "sort", "unique":
		proposal.Operations = PrepareCollection(s, request, auxiliaryID)
	case "extract_transform":
		if len(request.Path) > 0 {
			proposal.Operations = append(proposal.Operations, query.Operation{ID: auxiliaryID, Kind: "select", Path: request.Path})
		}
		request.Kind = "transform"
		request.Path = nil
		proposal.Operations = append(proposal.Operations, request)
	default:
		proposal.Operations = []query.Operation{request}
	}
	return proposal
}

func prefix(pattern, path query.Path) bool {
	if len(pattern) > len(path) {
		return false
	}
	for i, p := range pattern {
		v := path[i]
		if p.Kind == "each" && v.Kind == "index" {
			continue
		}
		if p != v {
			return false
		}
	}
	return true
}
func Resolve(c Cursor, s query.StageContext) Resolution {
	if c.View == "stage" {
		return Resolution{Relative: c.Path, Mapped: true}
	}
	if s.HasOrigin && prefix(s.Origin, c.Path) {
		return Resolution{Relative: append(query.Path{}, c.Path[len(s.Origin):]...), Mapped: true}
	}
	for name, p := range s.FieldOrigins {
		if prefix(p, c.Path) {
			r := query.Path{query.Field(name)}
			r = append(r, c.Path[len(p):]...)
			return Resolution{Relative: r, Mapped: true}
		}
	}
	return Resolution{Reason: "This source location is not available at the current stage. Switch to Stage or inspect an earlier pipeline step."}
}
func AncestorArrays(root any, p query.Path) []query.Path {
	var r []query.Path
	v := root
	at := query.Path{}
	if _, ok := v.([]any); ok {
		r = append(r, at)
	}
	for _, seg := range p {
		n, ok := query.Get(v, query.Path{seg})
		if !ok {
			break
		}
		at = at.Append(seg)
		v = n
		if _, ok := v.([]any); ok {
			r = append(r, at)
		}
	}
	return r
}
func Actions(c Cursor, s query.StageContext) []Action {
	r := Resolve(c, s)
	a := []Action{{"copy_path", "Copy path", "Copy this JSON path", nil}, {"copy_value", "Copy value", "Copy the selected value", nil}}
	a = append(a, Action{"count", "Count current results", "Collect and count all current query results", nil})
	if s.Stream || s.Shape.Allows("array") {
		a = append(a, Action{"sort", "Sort current results", "Sort the current query collection by a field", []string{"key", "direction"}}, Action{"unique", "Unique current results", "Keep distinct collection values", []string{"key"}})
	}
	if s.Shape.Allows("object") {
		a = append(a, Action{"project", "Select multiple fields", "Construct an object from current-stage fields", []string{"fields"}})
	}
	if !r.Mapped {
		return a
	}
	a = append(a, Action{"select", "Extract value", "Select this relative path", nil}, Action{"filter", "Filter", "Keep values matching a predicate", []string{"path", "operator", "value"}})
	t := query.TypeOf(c.Value)
	if shape, err := query.At(s.Shape, r.Relative); err == nil && shape != nil && len(shape.Types) == 1 && !shape.Types["unknown"] {
		for kind := range shape.Types {
			t = kind
		}
	}
	if t == "array" {
		a = append(a, Action{"iterate", "Iterate elements", "Emit each array element", nil}, Action{"map", "Map elements", "Project or transform each element", []string{"operation"}})
	}
	if t == "string" || t == "number" || t == "array" {
		a = append(a, Action{"extract_transform", "Extract and transform", "Return the transformed selected value", []string{"transform"}}, Action{"update", "Update field in object", "Transform the field and retain its object", []string{"transform"}})
	}
	if t != "boolean" {
		a = append(a, Action{"length", "Length of value", "Length of this value (not result count)", nil})
	}
	return a
}

// PrepareCollection records automatic collection as an explicit operation.
func PrepareCollection(s query.StageContext, o query.Operation, id string) []query.Operation {
	if s.Stream {
		from := o.From
		o.From = ""
		return []query.Operation{{ID: id, Kind: "collect", From: from}, o}
	}
	return []query.Operation{o}
}

// RelativePredicate makes intervening arrays explicit. Each choice is supplied
// by the UI; no implicit existential filtering is introduced.
func RelativePredicate(p query.Path, kind string, value any, quantifiers []string) (query.Predicate, error) {
	for i, seg := range p {
		if seg.Kind == "index" || seg.Kind == "each" {
			if len(quantifiers) == 0 {
				return query.Predicate{}, fmt.Errorf("choose Any or All for %s", p[:i].String())
			}
			q := strings.ToLower(quantifiers[0])
			if q != "any" && q != "all" {
				return query.Predicate{}, fmt.Errorf("invalid quantifier")
			}
			child, e := RelativePredicate(p[i+1:], kind, value, quantifiers[1:])
			return query.Predicate{Kind: q, Path: p[:i], Children: []query.Predicate{child}}, e
		}
	}
	return query.Predicate{Kind: kind, Path: p, Value: value}, nil
}
