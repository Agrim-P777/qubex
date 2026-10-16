package tree

import (
	"context"
	"github.com/Agrim-P777/qubex/internal/query"
	"testing"
)

func TestLazyNavigation(t *testing.T) {
	a := make([]any, 100000)
	for i := range a {
		a[i] = i
	}
	tr := New([]any{a}, false)
	rows := tr.Rows(20)
	if len(rows) != 20 || len(tr.keys) != 0 || len(tr.Expanded) != 1 {
		t.Fatal("materialized tree", len(rows))
	}
	tr.Move(1)
	if tr.Cursor.Path[0].Index != 0 {
		t.Fatal(tr.Cursor)
	}
	tr.Move(-1)
	if len(tr.Cursor.Path) != 0 {
		t.Fatal(tr.Cursor)
	}
	tr.End()
	if tr.Cursor.Path[0].Index != 99999 {
		t.Fatal(tr.Cursor)
	}
	tr.Left()
	tr.Left()
	if len(tr.Rows(20)) != 1 {
		t.Fatal("collapse failed")
	}
}
func TestSearch(t *testing.T) {
	roots := []any{map[string]any{"nested": []any{map[string]any{"status": "Pending"}}}}
	r, e := Search(context.Background(), roots, SearchOptions{Text: "pnd", Target: "values"})
	if e != nil || len(r) != 1 {
		t.Fatal(r, e)
	}
	tr := New(roots, false)
	tr.Jump(r[0])
	if len(tr.Cursor.Path) != 3 {
		t.Fatal(tr.Cursor)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := Search(ctx, roots, SearchOptions{Text: "x"}); e == nil {
		t.Fatal("not cancelled")
	}
	if (Ref{}).Key() != (Ref{Path: query.Path{}}).Key() {
		t.Fatal("root identity")
	}
}
func BenchmarkVisibleRows(b *testing.B) {
	a := make([]any, 1000000)
	tr := New([]any{a}, false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Rows(40)
	}
}
