// Package compiler is the sole owner of jq-specific lowering.
package compiler

import (
	"encoding/json"
	"fmt"
	"github.com/Agrim-P777/qubex/internal/query"
	"strings"
)

type Location struct{ Start, End int }
type CompiledQuery struct {
	Text      string
	Locations map[string]Location
}

type fragment struct {
	text string
	ids  map[string]Location
}

func Compile(q query.QueryDocument) (CompiledQuery, error) {
	if q.Version != query.SchemaVersion {
		return CompiledQuery{}, fmt.Errorf("unsupported query schema %d", q.Version)
	}
	seen := map[string]bool{}
	if err := validateIDs(q.Operations, seen); err != nil {
		return CompiledQuery{}, err
	}
	return compile(q.Operations)
}
func validateIDs(ops []query.Operation, seen map[string]bool) error {
	for _, o := range ops {
		if o.ID == "" || seen[o.ID] {
			return fmt.Errorf("operation IDs must be nonempty and unique: %q", o.ID)
		}
		seen[o.ID] = true
		if err := validateIDs(o.Body, seen); err != nil {
			return err
		}
	}
	return nil
}
func compile(ops []query.Operation) (CompiledQuery, error) {
	var parts []fragment
	for _, o := range ops {
		if o.Kind == "collect" || o.Kind == "count" {
			start := 0
			if o.From != "" {
				start = -1
				for i, p := range parts {
					if location, ok := p.ids[o.From]; ok {
						if location.Start != 0 {
							return CompiledQuery{}, fmt.Errorf("%s: collection boundary %s is already enclosed", o.ID, o.From)
						}
						start = i
						break
					}
				}
				if start < 0 {
					return CompiledQuery{}, fmt.Errorf("%s: missing collection boundary %s", o.ID, o.From)
				}
			}
			inner := join(partsToText(parts[start:]))
			wrapped := "[" + inner + "]"
			if o.Kind == "count" {
				wrapped += " | length"
			}
			ids := map[string]Location{}
			offset := 1
			for _, p := range parts[start:] {
				for id, l := range p.ids {
					ids[id] = Location{l.Start + offset, l.End + offset}
				}
				offset += len(p.text) + 3
			}
			ids[o.ID] = Location{0, len(wrapped)}
			parts = append(parts[:start], fragment{wrapped, ids})
			continue
		}
		s, err := operation(o)
		if err != nil {
			return CompiledQuery{}, fmt.Errorf("%s: %w", o.ID, err)
		}
		locations := map[string]Location{o.ID: {0, len(s)}}
		if len(o.Body) > 0 && (o.Kind == "map" || o.Kind == "update") {
			body, e := compile(o.Body)
			if e != nil {
				return CompiledQuery{}, e
			}
			offset := len("map(")
			if o.Kind == "update" {
				offset = len(o.Path.String() + " |= (")
			}
			for id, l := range body.Locations {
				locations[id] = Location{offset + l.Start, offset + l.End}
			}
		}
		parts = append(parts, fragment{s, locations})
	}
	texts := []string{}
	loc := map[string]Location{}
	offset := 0
	for _, p := range parts {
		texts = append(texts, p.text)
		for id, l := range p.ids {
			loc[id] = Location{offset + l.Start, offset + l.End}
		}
		offset += len(p.text) + 3
	}
	return CompiledQuery{join(texts), loc}, nil
}

func partsToText(p []fragment) []string {
	r := make([]string, len(p))
	for i, v := range p {
		r[i] = v.text
	}
	return r
}
func join(p []string) string {
	if len(p) == 0 {
		return "."
	}
	return strings.Join(p, " | ")
}
func literal(v any) (string, error) { b, e := json.Marshal(v); return string(b), e }
func path(p query.Path) (string, error) {
	for _, s := range p {
		if s.Kind != "field" && s.Kind != "index" && s.Kind != "each" {
			return "", fmt.Errorf("invalid path segment %q", s.Kind)
		}
	}
	return p.String(), nil
}
func operation(o query.Operation) (string, error) {
	p, err := path(o.Path)
	if err != nil {
		return "", err
	}
	switch o.Kind {
	case "select":
		return p, nil
	case "iterate":
		if p == "." {
			return ".[]", nil
		}
		return p + "[]", nil
	case "filter":
		if o.Predicate == nil {
			return "", fmt.Errorf("filter requires a predicate")
		}
		s, e := predicate(*o.Predicate)
		return "select(" + s + ")", e
	case "project":
		var fields []string
		seen := map[string]bool{}
		for _, f := range o.Fields {
			if seen[f.Name] {
				return "", fmt.Errorf("duplicate output field %q", f.Name)
			}
			seen[f.Name] = true
			k, _ := literal(f.Name)
			v, e := path(f.Path)
			if e != nil {
				return "", e
			}
			fields = append(fields, k+": "+v)
		}
		return "{" + strings.Join(fields, ", ") + "}", nil
	case "map":
		b, e := compile(o.Body)
		return "map(" + b.Text + ")", e
	case "update":
		if o.Transform != nil {
			s, e := transform(*o.Transform)
			return p + " |= (" + s + ")", e
		}
		b, e := compile(o.Body)
		return p + " |= (" + b.Text + ")", e
	case "transform":
		if o.Transform == nil {
			return "", fmt.Errorf("missing transform")
		}
		return transform(*o.Transform)
	case "sort":
		s := "sort_by(" + p + ")"
		if o.Descending {
			s += " | reverse"
		}
		return s, nil
	case "unique":
		if len(o.Path) == 0 {
			return "unique", nil
		}
		return "unique_by(" + p + ")", nil
	case "length":
		return "length", nil
	default:
		return "", fmt.Errorf("unknown operation %q", o.Kind)
	}
}
func predicate(p query.Predicate) (string, error) {
	x, e := path(p.Path)
	if e != nil {
		return "", e
	}
	v, e := literal(p.Value)
	if e != nil {
		return "", e
	}
	switch p.Kind {
	case "eq", "ne", "gt", "ge", "lt", "le":
		op := map[string]string{"eq": "==", "ne": "!=", "gt": ">", "ge": ">=", "lt": "<", "le": "<="}[p.Kind]
		return "(" + x + " " + op + " " + v + ")", nil
	case "contains", "startswith", "endswith":
		return "(" + x + " | " + p.Kind + "(" + v + "))", nil
	case "type":
		return "(" + x + " | type == " + v + ")", nil
	case "exists":
		if len(p.Path) == 0 {
			return "true", nil
		}
		last := p.Path[len(p.Path)-1]
		parent := p.Path[:len(p.Path)-1]
		k := any(last.Key)
		typ := "object"
		if last.Kind == "index" {
			k = last.Index
			typ = "array"
		}
		key, _ := literal(k)
		return "(" + parent.String() + " | if type == \"" + typ + "\" then has(" + key + ") else false end)", nil
	case "and", "or":
		if len(p.Children) == 0 {
			return "", fmt.Errorf("%s requires predicates", p.Kind)
		}
		var s []string
		for _, c := range p.Children {
			v, e := predicate(c)
			if e != nil {
				return "", e
			}
			s = append(s, v)
		}
		return "(" + strings.Join(s, " "+p.Kind+" ") + ")", nil
	case "not":
		if len(p.Children) != 1 {
			return "", fmt.Errorf("not requires one predicate")
		}
		s, e := predicate(p.Children[0])
		return "(" + s + " | not)", e
	case "any", "all":
		if len(p.Children) != 1 {
			return "", fmt.Errorf("%s requires one predicate", p.Kind)
		}
		s, e := predicate(p.Children[0])
		return "(" + x + " | " + p.Kind + "(.[]; " + s + "))", e
	default:
		return "", fmt.Errorf("unknown predicate %q", p.Kind)
	}
}
func transform(t query.Transform) (string, error) {
	v, e := literal(t.Value)
	if e != nil {
		return "", e
	}
	switch t.Kind {
	case "upper":
		return "ascii_upcase", nil
	case "lower":
		return "ascii_downcase", nil
	case "trim":
		return `gsub("^\\s+|\\s+$"; "")`, nil
	case "replace":
		if s, ok := t.Value.(string); !ok || s == "" {
			return "", fmt.Errorf("replacement search must be a nonempty string")
		}
		r, _ := literal(t.Replacement)
		return "split(" + v + ") | join(" + r + ")", nil
	case "split", "join":
		if _, ok := t.Value.(string); !ok {
			return "", fmt.Errorf("%s requires a string separator", t.Kind)
		}
		return t.Kind + "(" + v + ")", nil
	case "add", "subtract", "multiply", "divide":
		switch t.Value.(type) {
		case json.Number, float64, int, int64:
		default:
			return "", fmt.Errorf("arithmetic requires a number")
		}
		op := map[string]string{"add": "+", "subtract": "-", "multiply": "*", "divide": "/"}[t.Kind]
		return ". " + op + " " + v, nil
	case "floor", "ceil", "round":
		return t.Kind, nil
	default:
		return "", fmt.Errorf("unknown transform %q", t.Kind)
	}
}

// Pretty only changes separators outside JSON strings; literals stay intact.
func Pretty(s string) string {
	var b strings.Builder
	quoted, escape := false, false
	for _, r := range s {
		if quoted {
			b.WriteRune(r)
			if escape {
				escape = false
			} else if r == '\\' {
				escape = true
			} else if r == '"' {
				quoted = false
			}
			continue
		}
		if r == '"' {
			quoted = true
		}
		if r == '|' {
			b.WriteString("\n| ")
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
