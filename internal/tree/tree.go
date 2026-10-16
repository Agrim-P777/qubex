// Package tree navigates JSON without flattening all visible descendants.
package tree

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Agrim-P777/qubex/internal/query"
	"sort"
	"strings"
)

type Ref struct {
	Result int
	Path   query.Path
}

func (r Ref) Key() string {
	if len(r.Path) == 0 {
		return fmt.Sprintf("%d:root", r.Result)
	}
	b, _ := json.Marshal(r)
	return string(b)
}

type Row struct {
	Ref                 Ref
	Value               any
	Label               string
	Depth               int
	Container, Expanded bool
}
type Tree struct {
	Roots       []any
	Cursor, Top Ref
	Expanded    map[string]bool
	keys        map[string][]string
	Sequence    bool
	AllExpanded bool
}

func (t *Tree) IsExpanded(r Ref) bool {
	if v, ok := t.Expanded[r.Key()]; ok {
		return v
	}
	return t.AllExpanded
}

func New(roots []any, sequence bool) *Tree {
	t := &Tree{Roots: roots, Expanded: map[string]bool{}, keys: map[string][]string{}, Sequence: sequence}
	t.Expanded[t.Cursor.Key()] = true
	return t
}
func (t *Tree) Value(r Ref) (any, bool) {
	if r.Result < 0 || r.Result >= len(t.Roots) {
		return nil, false
	}
	return query.Get(t.Roots[r.Result], r.Path)
}
func (t *Tree) Keys(r Ref) []string {
	key := r.Key()
	if k, ok := t.keys[key]; ok {
		return k
	}
	v, _ := t.Value(r)
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	k := make([]string, 0, len(m))
	for s := range m {
		k = append(k, s)
	}
	sort.Strings(k)
	t.keys[key] = k
	return k
}
func (t *Tree) Child(r Ref, last bool) (Ref, bool) {
	v, ok := t.Value(r)
	if !ok {
		return r, false
	}
	switch x := v.(type) {
	case []any:
		if len(x) > 0 {
			i := 0
			if last {
				i = len(x) - 1
			}
			return Ref{r.Result, r.Path.Append(query.Index(i))}, true
		}
	case map[string]any:
		k := t.Keys(r)
		if len(k) > 0 {
			i := 0
			if last {
				i = len(k) - 1
			}
			return Ref{r.Result, r.Path.Append(query.Field(k[i]))}, true
		}
	}
	return r, false
}
func (t *Tree) sibling(r Ref, d int) (Ref, bool) {
	if len(r.Path) == 0 {
		n := r.Result + d
		if n >= 0 && n < len(t.Roots) {
			return Ref{Result: n}, true
		}
		return r, false
	}
	p := Ref{r.Result, r.Path[:len(r.Path)-1]}
	s := r.Path[len(r.Path)-1]
	v, _ := t.Value(p)
	switch x := v.(type) {
	case []any:
		n := s.Index + d
		if n >= 0 && n < len(x) {
			return Ref{r.Result, p.Path.Append(query.Index(n))}, true
		}
	case map[string]any:
		k := t.Keys(p)
		i := sort.SearchStrings(k, s.Key) + d
		if i >= 0 && i < len(k) {
			return Ref{r.Result, p.Path.Append(query.Field(k[i]))}, true
		}
	}
	return r, false
}
func (t *Tree) Next(r Ref) (Ref, bool) {
	if t.IsExpanded(r) {
		if n, ok := t.Child(r, false); ok {
			return n, true
		}
	}
	for {
		if n, ok := t.sibling(r, 1); ok {
			return n, true
		}
		if len(r.Path) == 0 {
			return r, false
		}
		r.Path = r.Path[:len(r.Path)-1]
	}
}
func (t *Tree) Prev(r Ref) (Ref, bool) {
	if n, ok := t.sibling(r, -1); ok {
		for t.IsExpanded(n) {
			c, ok := t.Child(n, true)
			if !ok {
				break
			}
			n = c
		}
		return n, true
	}
	if len(r.Path) > 0 {
		return Ref{r.Result, r.Path[:len(r.Path)-1]}, true
	}
	return r, false
}
func (t *Tree) Move(d int) {
	for i := 0; i < abs(d); i++ {
		var n Ref
		var ok bool
		if d > 0 {
			n, ok = t.Next(t.Cursor)
		} else {
			n, ok = t.Prev(t.Cursor)
		}
		if !ok {
			break
		}
		t.Cursor = n
	}
}
func (t *Tree) Left() {
	if t.IsExpanded(t.Cursor) {
		t.Expanded[t.Cursor.Key()] = false
	} else if len(t.Cursor.Path) > 0 {
		t.Cursor.Path = t.Cursor.Path[:len(t.Cursor.Path)-1]
	}
}
func (t *Tree) Right() {
	if t.IsExpanded(t.Cursor) {
		if n, ok := t.Child(t.Cursor, false); ok {
			t.Cursor = n
		}
	} else {
		t.Expanded[t.Cursor.Key()] = true
	}
}
func (t *Tree) Toggle() { k := t.Cursor.Key(); t.Expanded[k] = !t.IsExpanded(t.Cursor) }
func (t *Tree) Jump(r Ref) {
	t.Cursor = r
	t.Top = r
	for i := 0; i < len(r.Path); i++ {
		t.Expanded[(Ref{r.Result, r.Path[:i]}).Key()] = true
	}
}
func (t *Tree) End() {
	if len(t.Roots) == 0 {
		return
	}
	r := Ref{Result: len(t.Roots) - 1}
	for t.IsExpanded(r) {
		n, ok := t.Child(r, true)
		if !ok {
			break
		}
		r = n
	}
	t.Jump(r)
}
func (t *Tree) Rows(height int) []Row {
	if len(t.Roots) == 0 || height <= 0 {
		return nil
	}
	r := t.Top
	found := false
	for i := 0; i < height; i++ {
		if r.Key() == t.Cursor.Key() {
			found = true
			break
		}
		n, ok := t.Next(r)
		if !ok {
			break
		}
		r = n
	}
	if !found {
		t.Top = t.Cursor
	}
	r = t.Top
	rows := make([]Row, 0, height)
	for i := 0; i < height; i++ {
		v, ok := t.Value(r)
		if !ok {
			break
		}
		label := "$"
		if t.Sequence {
			label = fmt.Sprintf("RESULT %d", r.Result+1)
		}
		if len(r.Path) > 0 {
			s := r.Path[len(r.Path)-1]
			if s.Kind == "field" {
				b, _ := json.Marshal(s.Key)
				label = string(b)
			} else {
				label = fmt.Sprintf("[%d]", s.Index)
			}
		}
		typ := query.TypeOf(v)
		rows = append(rows, Row{r, v, label, len(r.Path), typ == "object" || typ == "array", t.IsExpanded(r)})
		n, ok := t.Next(r)
		if !ok {
			break
		}
		r = n
	}
	return rows
}
func Summary(v any, limit int) string {
	switch x := v.(type) {
	case []any:
		return fmt.Sprintf("[%d items]", len(x))
	case map[string]any:
		return fmt.Sprintf("{%d keys}", len(x))
	case string:
		if len(x) > limit {
			x = x[:limit] + "…"
		}
		b, _ := json.Marshal(x)
		return string(b)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
func Match(text, pattern string, exact, caseSensitive bool) bool {
	if !caseSensitive {
		text = strings.ToLower(text)
		pattern = strings.ToLower(pattern)
	}
	if exact {
		return strings.Contains(text, pattern)
	}
	p := []rune(pattern)
	i := 0
	for _, r := range text {
		if i < len(p) && r == p[i] {
			i++
		}
	}
	return i == len(p)
}

type SearchOptions struct {
	Text, Target         string
	Exact, CaseSensitive bool
	Root                 *Ref
	Limit                int
}

func Search(ctx context.Context, roots []any, o SearchOptions) ([]Ref, error) {
	if o.Limit <= 0 {
		o.Limit = 1000
	}
	var results []Ref
	var walk func(Ref, any, string) error
	walk = func(r Ref, v any, key string) error {
		if len(results) >= o.Limit {
			return nil
		}
		if e := ctx.Err(); e != nil {
			return e
		}
		matched := o.Target != "values" && Match(key, o.Text, o.Exact, o.CaseSensitive)
		if o.Target != "keys" {
			switch v.(type) {
			case map[string]any, []any:
			default:
				matched = matched || Match(Summary(v, 1<<20), o.Text, o.Exact, o.CaseSensitive)
			}
		}
		if matched {
			results = append(results, r)
		}
		switch x := v.(type) {
		case []any:
			for i, v := range x {
				if e := walk(Ref{r.Result, r.Path.Append(query.Index(i))}, v, fmt.Sprint(i)); e != nil {
					return e
				}
				if len(results) >= o.Limit {
					break
				}
			}
		case map[string]any:
			ks := make([]string, 0, len(x))
			for k := range x {
				ks = append(ks, k)
			}
			sort.Strings(ks)
			for _, k := range ks {
				if e := walk(Ref{r.Result, r.Path.Append(query.Field(k))}, x[k], k); e != nil {
					return e
				}
				if len(results) >= o.Limit {
					break
				}
			}
		}
		return nil
	}
	if o.Root != nil {
		if o.Root.Result < len(roots) {
			v, ok := query.Get(roots[o.Root.Result], o.Root.Path)
			if ok {
				e := walk(*o.Root, v, "")
				return results, e
			}
		}
		return results, nil
	}
	for i, v := range roots {
		if e := walk(Ref{Result: i}, v, ""); e != nil {
			return results, e
		}
		if len(results) >= o.Limit {
			break
		}
	}
	return results, nil
}
func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}
