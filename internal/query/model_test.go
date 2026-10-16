package query

import "testing"

func TestTimelineIsolation(t *testing.T) {
	tl := NewTimeline(New())
	q := New()
	q.Operations = append(q.Operations, Operation{ID: "x", Kind: "select", Path: Path{Field("x")}})
	tl.Commit(q)
	q.Operations[0].Path[0].Key = "changed"
	if tl.Current.Operations[0].Path[0].Key != "x" {
		t.Fatal("snapshot alias")
	}
	if !tl.Undo() || len(tl.Current.Operations) != 0 || !tl.Redo() {
		t.Fatal(tl)
	}
	tl.Undo()
	tl.Commit(New())
	if tl.Redo() {
		t.Fatal("redo survived branch")
	}
}
func TestPointers(t *testing.T) {
	p, e := ParsePath("/a~1b/~0/0")
	if e != nil || p[0].Key != "a/b" || p[1].Key != "~" || p[2].Kind != "field" {
		t.Fatal(p, e)
	}
	if p.Pointer() != "/a~1b/~0/0" {
		t.Fatal(p.Pointer())
	}
	s := Infer(map[string]any{"a": []any{1}})
	p = ResolvePointer(s, Path{Field("a"), Field("0")})
	if p[1].Kind != "index" {
		t.Fatal(p)
	}
}
