package query

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strconv"
)

type Shape struct {
	Types   map[string]bool
	Fields  map[string]*Shape
	Element *Shape
	Present int
	Samples int
	Sampled bool
	// Closed identifies a constructed object, whose omitted fields are known.
	Closed bool
}

func Unknown() *Shape        { return &Shape{Types: map[string]bool{"unknown": true}} }
func OfType(t string) *Shape { return &Shape{Types: map[string]bool{t: true}} }
func TypeOf(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case string:
		return "string"
	case json.Number, float64, float32, int, int64, *big.Int:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return "unknown"
}
func (s *Shape) Allows(t string) bool { return s == nil || s.Types["unknown"] || s.Types[t] }
func (s *Shape) Names() []string {
	if s == nil {
		return nil
	}
	a := make([]string, 0, len(s.Fields))
	for k := range s.Fields {
		a = append(a, k)
	}
	sort.Strings(a)
	return a
}
func Infer(v any) *Shape { budget := 8192; return infer(v, 0, &budget) }
func infer(v any, depth int, budget *int) *Shape {
	if depth > 32 || *budget <= 0 {
		return Unknown()
	}
	*budget--
	s := OfType(TypeOf(v))
	s.Samples = 1
	s.Present = 1
	switch x := v.(type) {
	case []any:
		s.Element = nil
		limit := min(len(x), 128)
		s.Sampled = limit < len(x)
		for _, v := range x[:limit] {
			s.Element = Merge(s.Element, infer(v, depth+1, budget))
		}
		if s.Element == nil {
			s.Element = Unknown()
		}
	case map[string]any:
		s.Fields = map[string]*Shape{}
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		s.Sampled = len(keys) > 256
		for _, k := range keys[:min(len(keys), 256)] {
			s.Fields[k] = infer(x[k], depth+1, budget)
		}
	}
	return s
}
func Merge(a, b *Shape) *Shape {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	r := &Shape{Types: map[string]bool{}, Fields: map[string]*Shape{}, Samples: a.Samples + b.Samples, Present: a.Present + b.Present, Sampled: a.Sampled || b.Sampled}
	for k := range a.Types {
		r.Types[k] = true
	}
	for k := range b.Types {
		r.Types[k] = true
	}
	for k, v := range a.Fields {
		r.Fields[k] = v
	}
	for k, v := range b.Fields {
		r.Fields[k] = Merge(r.Fields[k], v)
	}
	r.Element = Merge(a.Element, b.Element)
	return r
}
func At(s *Shape, p Path) (*Shape, error) {
	for _, part := range p {
		if s == nil {
			return Unknown(), nil
		}
		switch part.Kind {
		case "field":
			if !s.Allows("object") && !s.Allows("null") {
				return Unknown(), fmt.Errorf("field %q requires an object", part.Key)
			}
			v, ok := s.Fields[part.Key]
			if !ok {
				if s.Closed {
					return Unknown(), fmt.Errorf("field %q was removed by an earlier projection", part.Key)
				}
				return Unknown(), nil
			}
			s = v
		case "index", "each":
			if !s.Allows("array") {
				return Unknown(), fmt.Errorf("array access requires an array")
			}
			s = s.Element
		default:
			return Unknown(), fmt.Errorf("invalid path segment")
		}
	}
	if s == nil {
		return Unknown(), nil
	}
	return s, nil
}
func ResolvePointer(s *Shape, p Path) Path {
	out := Path{}
	for _, seg := range p {
		if s != nil && s.Types["array"] && !s.Types["object"] && seg.Kind == "field" {
			if n, e := strconv.Atoi(seg.Key); e == nil && n >= 0 {
				seg = Index(n)
			}
		}
		out = append(out, seg)
		s, _ = At(s, Path{seg})
	}
	return out
}

type StageContext struct {
	Shape           *Shape
	Stream          bool
	Origin          Path
	HasOrigin       bool
	FieldOrigins    map[string]Path
	CollectionStart string
}

func Analyze(root *Shape, q QueryDocument) ([]StageContext, []Diagnostic) {
	if root == nil {
		root = Unknown()
	}
	s := StageContext{Shape: root, HasOrigin: true}
	stages := []StageContext{s}
	var ds []Diagnostic
	for index, o := range q.Operations {
		next, err := advance(s, o)
		if err == nil && (o.Kind == "collect" || o.Kind == "count") && o.From != "" {
			found := false
			for i := 0; i < index; i++ {
				if q.Operations[i].ID == o.From {
					next.Stream = stages[i].Stream
					found = true
					break
				}
			}
			if !found {
				err = fmt.Errorf("missing collection boundary %s", o.From)
			}
		}
		if err != nil {
			ds = append(ds, Diagnostic{o.ID, err.Error()})
			next = StageContext{Shape: Unknown(), Stream: s.Stream}
		}
		s = next
		stages = append(stages, s)
	}
	return stages, ds
}
func advance(s StageContext, o Operation) (StageContext, error) {
	r := s
	target := s.Shape
	var e error
	if o.Kind != "sort" && o.Kind != "unique" {
		target, e = At(s.Shape, o.Path)
		if e != nil {
			return r, e
		}
	}
	require := func(t string) error {
		if !s.Shape.Allows(t) {
			return fmt.Errorf("%s requires %s input", o.Kind, t)
		}
		return nil
	}
	switch o.Kind {
	case "select":
		r.Shape = target
		for _, segment := range o.Path {
			if segment.Kind == "each" {
				r.Stream = true
				if !s.Stream {
					r.CollectionStart = o.ID
				}
				break
			}
		}
		if r.HasOrigin {
			r.Origin = append(append(Path{}, s.Origin...), o.Path...)
		} else if len(o.Path) > 0 && o.Path[0].Kind == "field" {
			if p, ok := s.FieldOrigins[o.Path[0].Key]; ok {
				r.Origin = append(append(Path{}, p...), o.Path[1:]...)
				r.HasOrigin = true
			}
		}
		r.FieldOrigins = nil
	case "iterate":
		if !target.Allows("array") {
			return r, fmt.Errorf("iterate requires an array")
		}
		r.Shape = target.Element
		if r.Shape == nil {
			r.Shape = Unknown()
		}
		r.Stream = true
		if !s.Stream {
			r.CollectionStart = o.ID
		}
		if r.HasOrigin {
			r.Origin = append(append(append(Path{}, s.Origin...), o.Path...), Each())
		}
		r.FieldOrigins = nil
	case "filter":
		if o.Predicate == nil {
			return r, fmt.Errorf("missing predicate")
		}
		if e := checkPredicate(s.Shape, *o.Predicate); e != nil {
			return r, e
		}
	case "project":
		r.Shape = OfType("object")
		r.Shape.Fields = map[string]*Shape{}
		r.Shape.Closed = true
		r.FieldOrigins = map[string]Path{}
		for _, f := range o.Fields {
			t, e := At(s.Shape, f.Path)
			if e != nil {
				return r, e
			}
			r.Shape.Fields[f.Name] = t
			if s.HasOrigin {
				r.FieldOrigins[f.Name] = append(append(Path{}, s.Origin...), f.Path...)
			} else if len(f.Path) > 0 {
				if p, ok := s.FieldOrigins[f.Path[0].Key]; ok {
					r.FieldOrigins[f.Name] = append(append(Path{}, p...), f.Path[1:]...)
				}
			}
		}
		r.HasOrigin = false
	case "collect":
		r.Shape = &Shape{Types: map[string]bool{"array": true}, Element: s.Shape}
		r.Stream = false
		r.HasOrigin = false
		r.FieldOrigins = nil
		r.CollectionStart = ""
	case "count", "length":
		if o.Kind == "length" && !s.Shape.Allows("array") && !s.Shape.Allows("object") && !s.Shape.Allows("string") && !s.Shape.Allows("number") && !s.Shape.Allows("null") {
			return r, fmt.Errorf("length is not defined on this value")
		}
		r.Shape = OfType("number")
		r.HasOrigin = false
		r.FieldOrigins = nil
		if o.Kind == "count" {
			r.Stream = false
			r.CollectionStart = ""
		}
	case "sort", "unique":
		if s.Stream && !s.Shape.Types["array"] {
			return r, fmt.Errorf("collect the result stream before %s", o.Kind)
		}
		if e := require("array"); e != nil {
			return r, e
		}
		if _, e := At(s.Shape.Element, o.Path); e != nil {
			return r, e
		}
		// Array indexes after ordering/deduplication are not source identities.
		r.HasOrigin = false
		r.FieldOrigins = nil
	case "transform":
		if o.Transform == nil {
			return r, fmt.Errorf("missing transform")
		}
		t, e := transformShape(s.Shape, *o.Transform)
		if e != nil {
			return r, e
		}
		r.Shape = t
		r.HasOrigin = false
		r.FieldOrigins = nil
	case "update":
		var t *Shape
		if o.Transform != nil {
			t, e = transformShape(target, *o.Transform)
		} else {
			st, ds := Analyze(target, QueryDocument{Version: 1, Operations: o.Body})
			if len(ds) > 0 {
				return r, ds[0]
			}
			t = st[len(st)-1].Shape
		}
		if e != nil {
			return r, e
		}
		r.Shape = replaceShape(s.Shape, o.Path, t)
	case "map":
		if e := require("array"); e != nil {
			return r, e
		}
		st, ds := Analyze(s.Shape.Element, QueryDocument{Version: 1, Operations: o.Body})
		if len(ds) > 0 {
			return r, ds[0]
		}
		r.Shape = &Shape{Types: map[string]bool{"array": true}, Element: st[len(st)-1].Shape}
		r.HasOrigin = false
		r.FieldOrigins = nil
	default:
		return r, fmt.Errorf("unknown operation %q", o.Kind)
	}
	return r, nil
}
func replaceShape(s *Shape, p Path, t *Shape) *Shape {
	if len(p) == 0 {
		return t
	}
	if s == nil {
		return Unknown()
	}
	r := *s
	if p[0].Kind == "field" {
		r.Fields = map[string]*Shape{}
		for k, v := range s.Fields {
			r.Fields[k] = v
		}
		r.Fields[p[0].Key] = replaceShape(s.Fields[p[0].Key], p[1:], t)
	} else {
		r.Element = replaceShape(s.Element, p[1:], t)
	}
	return &r
}
func transformShape(s *Shape, t Transform) (*Shape, error) {
	want := "number"
	out := "number"
	switch t.Kind {
	case "upper", "lower", "trim", "replace":
		want = "string"
		out = "string"
	case "split":
		want = "string"
		out = "array"
	case "join":
		want = "array"
		out = "string"
	}
	if !s.Allows(want) {
		return nil, fmt.Errorf("%s requires %s", t.Kind, want)
	}
	r := OfType(out)
	if out == "array" {
		r.Element = OfType("string")
	}
	return r, nil
}
func checkPredicate(s *Shape, p Predicate) error {
	t, e := At(s, p.Path)
	if e != nil {
		return e
	}
	if p.Kind == "any" || p.Kind == "all" {
		if !t.Allows("array") {
			return fmt.Errorf("%s requires an array", p.Kind)
		}
		for _, c := range p.Children {
			if e := checkPredicate(t.Element, c); e != nil {
				return e
			}
		}
	} else {
		for _, c := range p.Children {
			if e := checkPredicate(s, c); e != nil {
				return e
			}
		}
	}
	return nil
}
