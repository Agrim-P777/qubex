package context

import (
	"github.com/Agrim-P777/qubex/internal/query"
	"testing"
)

func TestSourceMapping(t *testing.T) {
	input := map[string]any{"pods": []any{map[string]any{"name": "api", "status": "ok"}}}
	q := query.QueryDocument{Version: 1, Operations: []query.Operation{{ID: "1", Kind: "iterate", Path: query.Path{query.Field("pods")}}, {ID: "2", Kind: "filter", Predicate: &query.Predicate{Kind: "eq", Path: query.Path{query.Field("status")}, Value: "ok"}}, {ID: "3", Kind: "project", Fields: []query.Projection{{Name: "label", Path: query.Path{query.Field("name")}}}}}}
	st, ds := query.Analyze(query.Infer(input), q)
	if len(ds) != 0 {
		t.Fatal(ds)
	}
	c := Cursor{View: "source", Path: query.Path{query.Field("pods"), query.Index(19), query.Field("name")}}
	r := Resolve(c, st[3])
	if !r.Mapped || r.Relative.String() != ".label" {
		t.Fatal(r)
	}
	c.Path[2] = query.Field("status")
	if Resolve(c, st[3]).Mapped {
		t.Fatal("removed field mapped")
	}
	c.View = "stage"
	c.Path = query.Path{query.Field("label")}
	if !Resolve(c, st[3]).Mapped {
		t.Fatal("stage mapping failed")
	}
}
func TestExplicitNestedQuantifiers(t *testing.T) {
	p := query.Path{query.Field("endpoints"), query.Index(0), query.Field("scopes"), query.Index(2)}
	if _, e := RelativePredicate(p, "eq", "write", nil); e == nil {
		t.Fatal("silently chose a quantifier")
	}
	r, e := RelativePredicate(p, "eq", "write", []string{"all", "any"})
	if e != nil || r.Kind != "all" || r.Children[0].Kind != "any" {
		t.Fatal(r, e)
	}
}
func TestInvalidDownstream(t *testing.T) {
	q := query.QueryDocument{Version: 1, Operations: []query.Operation{{ID: "p", Kind: "project", Fields: []query.Projection{{Name: "a", Path: query.Path{query.Field("a")}}}}, {ID: "s", Kind: "select", Path: query.Path{query.Field("b")}}}}
	_, ds := query.Analyze(query.Infer(map[string]any{"a": 1, "b": 2}), q)
	if len(ds) != 1 || ds[0].OperationID != "s" {
		t.Fatal(ds)
	}
}

func TestSortDoesNotMapSourceIndexes(t *testing.T) {
	q := query.QueryDocument{Version: 1, Operations: []query.Operation{{ID: "s", Kind: "sort"}}}
	st, ds := query.Analyze(query.Infer([]any{2, 1}), q)
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	if Resolve(Cursor{View: "source", Path: query.Path{query.Index(0)}}, st[1]).Mapped {
		t.Fatal("source index was interpreted as sorted index")
	}
}
func TestMixedAndEmpty(t *testing.T) {
	s := query.Infer([]any{map[string]any{"a": 1}, map[string]any{"a": "x"}, map[string]any{}})
	if !s.Element.Fields["a"].Allows("number") || !s.Element.Fields["a"].Allows("string") {
		t.Fatal(s)
	}
	q := query.QueryDocument{Version: 1, Operations: []query.Operation{{ID: "i", Kind: "iterate"}, {ID: "p", Kind: "project", Fields: []query.Projection{{Name: "a", Path: query.Path{query.Field("a")}}}}}}
	st, ds := query.Analyze(query.Infer([]any{}), q)
	if len(ds) > 0 || !st[2].Shape.Closed || len(st[2].Shape.Names()) != 1 {
		t.Fatal(st, ds)
	}
}
