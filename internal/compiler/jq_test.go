package compiler_test

import (
	"context"
	"encoding/json"
	"github.com/Agrim-P777/qubex/internal/compiler"
	"github.com/Agrim-P777/qubex/internal/data"
	"github.com/Agrim-P777/qubex/internal/execute"
	"github.com/Agrim-P777/qubex/internal/query"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func field(k string) query.Path { return query.Path{query.Field(k)} }
func run(t *testing.T, input string, ops ...query.Operation) any {
	t.Helper()
	for i := range ops {
		if ops[i].ID == "" {
			ops[i].ID = string(rune('a' + i))
		}
	}
	d, e := data.Read(context.Background(), strings.NewReader(input), "fixture", 1<<20, nil)
	if e != nil {
		t.Fatal(e)
	}
	q := query.QueryDocument{Version: 1, Operations: ops}
	c, e := compiler.Compile(q)
	if e != nil {
		t.Fatal(e)
	}
	r := execute.New().Run(context.Background(), q, d.Value, 1, execute.Limits{}, nil)
	if r.Error != nil {
		t.Fatalf("%s: %v", c.Text, r.Error)
	}
	b, _ := json.Marshal(r.Values)
	var values any
	json.Unmarshal(b, &values)
	if bin, e := exec.LookPath("jq"); e == nil {
		cmd := exec.Command(bin, "-c", c.Text)
		cmd.Stdin = strings.NewReader(input)
		out, e := cmd.Output()
		if e != nil {
			t.Fatalf("jq rejected %s: %v", c.Text, e)
		}
		dec := json.NewDecoder(strings.NewReader(string(out)))
		var expected []any
		for dec.More() {
			var v any
			if e := dec.Decode(&v); e != nil {
				t.Fatal(e)
			}
			expected = append(expected, v)
		}
		b, _ = json.Marshal(expected)
		var normalized any
		json.Unmarshal(b, &normalized)
		if !reflect.DeepEqual(values, normalized) {
			t.Fatalf("gojq/jq difference for %s: %v != %v", c.Text, values, normalized)
		}
	} else if os.Getenv("QBX_REQUIRE_JQ") == "1" {
		t.Fatal("jq is required for compatibility tests")
	}
	return values
}
func TestAcceptance(t *testing.T) {
	ops := []query.Operation{{Kind: "iterate", Path: field("services")}, {Kind: "filter", Predicate: &query.Predicate{Kind: "eq", Path: field("language"), Value: "python"}}, {Kind: "project", Fields: []query.Projection{{Name: "name", Path: field("name")}, {Name: "replicas", Path: field("replicas")}}}, {Kind: "collect"}, {Kind: "sort", Path: field("replicas"), Descending: true}}
	got := run(t, `{"services":[{"name":"api","language":"python","replicas":3},{"name":"worker","language":"python","replicas":8},{"name":"web","language":"go","replicas":2}]}`, ops...)
	b, _ := json.Marshal(got)
	if string(b) != `[[{"name":"worker","replicas":8},{"name":"api","replicas":3}]]` {
		t.Fatal(string(b))
	}
}
func TestTransforms(t *testing.T) {
	tests := []struct {
		input string
		tr    query.Transform
		want  string
	}{{`"AbC"`, query.Transform{Kind: "lower"}, `["abc"]`}, {`"AbC"`, query.Transform{Kind: "upper"}, `["ABC"]`}, {`" \t hi \n"`, query.Transform{Kind: "trim"}, `["hi"]`}, {`"a.b.a"`, query.Transform{Kind: "replace", Value: ".", Replacement: "/"}, `["a/b/a"]`}, {`"a,b"`, query.Transform{Kind: "split", Value: ","}, `[["a","b"]]`}, {`["a","b"]`, query.Transform{Kind: "join", Value: "-"}, `["a-b"]`}, {`5`, query.Transform{Kind: "add", Value: 2}, `[7]`}, {`5`, query.Transform{Kind: "subtract", Value: 2}, `[3]`}, {`5`, query.Transform{Kind: "multiply", Value: 2}, `[10]`}, {`5`, query.Transform{Kind: "divide", Value: 2}, `[2.5]`}, {`2.6`, query.Transform{Kind: "floor"}, `[2]`}, {`2.2`, query.Transform{Kind: "ceil"}, `[3]`}, {`2.6`, query.Transform{Kind: "round"}, `[3]`}}
	for _, tt := range tests {
		t.Run(tt.tr.Kind, func(t *testing.T) {
			got := run(t, tt.input, query.Operation{Kind: "transform", Transform: &tt.tr})
			b, _ := json.Marshal(got)
			if string(b) != tt.want {
				t.Fatalf("got %s want %s", b, tt.want)
			}
		})
	}
}
func TestPredicates(t *testing.T) {
	for _, kind := range []string{"eq", "ne", "gt", "ge", "lt", "le"} {
		t.Run(kind, func(t *testing.T) {
			run(t, `[0,1,2,null,"x"]`, query.Operation{Kind: "iterate"}, query.Operation{Kind: "filter", Predicate: &query.Predicate{Kind: kind, Value: 1}})
		})
	}
	for _, kind := range []string{"contains", "startswith", "endswith"} {
		t.Run(kind, func(t *testing.T) {
			run(t, `["abc","bc","abcd",""]`, query.Operation{Kind: "iterate"}, query.Operation{Kind: "filter", Predicate: &query.Predicate{Kind: kind, Value: "bc"}})
		})
	}
	run(t, `[{"a":null},{}, {"a":false}]`, query.Operation{Kind: "iterate"}, query.Operation{Kind: "filter", Predicate: &query.Predicate{Kind: "exists", Path: field("a")}})
}
func TestNestedQuantifiersAndEmpty(t *testing.T) {
	for _, kind := range []string{"any", "all"} {
		p := query.Predicate{Kind: kind, Path: field("endpoints"), Children: []query.Predicate{{Kind: "any", Path: field("scopes"), Children: []query.Predicate{{Kind: "eq", Value: "write"}}}}}
		run(t, `[{"endpoints":[]},{"endpoints":[{"scopes":["read","write"]}]},{"endpoints":[{"scopes":[]}]}]`, query.Operation{Kind: "iterate"}, query.Operation{Kind: "filter", Predicate: &p})
	}
}
func TestMapUpdateUniqueAndLength(t *testing.T) {
	run(t, `[{"name":"a"},{"name":"b"}]`, query.Operation{Kind: "map", Body: []query.Operation{{ID: "child", Kind: "update", Path: field("name"), Transform: &query.Transform{Kind: "upper"}}}})
	run(t, `[{"n":2},{"n":1},{"n":2}]`, query.Operation{Kind: "unique", Path: field("n")}, query.Operation{Kind: "length"})
	run(t, `[1,1,2]`, query.Operation{Kind: "unique"})
	run(t, `[]`, query.Operation{Kind: "iterate"}, query.Operation{Kind: "count"})
}
func TestEscapingAndLocations(t *testing.T) {
	key := "x\" | error(\"bad\") #"
	b, _ := json.Marshal(map[string]any{key: "safe"})
	got := run(t, string(b), query.Operation{Kind: "select", Path: field(key)})
	if !reflect.DeepEqual(got, []any{"safe"}) {
		t.Fatal(got)
	}
	q := query.QueryDocument{Version: 1, Operations: []query.Operation{{ID: "a", Kind: "iterate"}, {ID: "b", Kind: "collect"}}}
	c, e := compiler.Compile(q)
	if e != nil {
		t.Fatal(e)
	}
	l := c.Locations["a"]
	if c.Text[l.Start:l.End] != ".[]" {
		t.Fatal(c)
	}
	pretty := compiler.Pretty(`select(.x == "a | b") | .x`)
	if !strings.Contains(pretty, `"a | b"`) {
		t.Fatal(pretty)
	}
}
func TestNestedCollectionBoundary(t *testing.T) {
	got := run(t, `[[3,1],[2,0]]`, query.Operation{ID: "outer", Kind: "iterate"}, query.Operation{ID: "inner", Kind: "iterate"}, query.Operation{ID: "collect", Kind: "collect", From: "inner"}, query.Operation{ID: "sort", Kind: "sort"})
	b, _ := json.Marshal(got)
	if string(b) != `[[1,3],[0,2]]` {
		t.Fatal(string(b))
	}
}
