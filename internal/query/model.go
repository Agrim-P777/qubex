// Package query defines Qubex's language-neutral, versioned query representation.
package query

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const SchemaVersion = 1

type Segment struct {
	Kind  string `json:"kind"`
	Key   string `json:"key,omitempty"`
	Index int    `json:"index,omitempty"`
}
type Path []Segment

func Field(k string) Segment         { return Segment{Kind: "field", Key: k} }
func Index(i int) Segment            { return Segment{Kind: "index", Index: i} }
func Each() Segment                  { return Segment{Kind: "each"} }
func (p Path) Append(s Segment) Path { return append(append(Path{}, p...), s) }
func (p Path) Pointer() string {
	var b strings.Builder
	for _, s := range p {
		b.WriteByte('/')
		switch s.Kind {
		case "field":
			b.WriteString(strings.ReplaceAll(strings.ReplaceAll(s.Key, "~", "~0"), "/", "~1"))
		case "index":
			b.WriteString(strconv.Itoa(s.Index))
		case "each":
			b.WriteByte('*')
		}
	}
	return b.String()
}
func (p Path) String() string {
	if len(p) == 0 {
		return "."
	}
	var b strings.Builder
	for _, s := range p {
		switch s.Kind {
		case "field":
			if identifier(s.Key) {
				b.WriteByte('.')
				b.WriteString(s.Key)
			} else {
				if b.Len() == 0 {
					b.WriteByte('.')
				}
				q, _ := json.Marshal(s.Key)
				b.WriteByte('[')
				b.Write(q)
				b.WriteByte(']')
			}
		case "index":
			if b.Len() == 0 {
				b.WriteByte('.')
			}
			fmt.Fprintf(&b, "[%d]", s.Index)
		case "each":
			if b.Len() == 0 {
				b.WriteByte('.')
			}
			b.WriteString("[]")
		}
	}
	return b.String()
}
func identifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		letter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
		digit := i > 0 && r >= '0' && r <= '9'
		if r != '_' && !letter && !digit {
			return false
		}
	}
	return true
}

// ParsePath accepts a JSON Pointer. Empty or "." means the current value.
// Field names are resolved by the context engine; numeric segments are fields
// until ResolvePointer observes an array. This avoids guessing numeric keys.
func ParsePath(s string) (Path, error) {
	if s == "" || s == "." {
		return Path{}, nil
	}
	if !strings.HasPrefix(s, "/") {
		return nil, fmt.Errorf("use a JSON Pointer such as /metadata/name")
	}
	var p Path
	for _, k := range strings.Split(s[1:], "/") {
		for i := 0; i < len(k); i++ {
			if k[i] == '~' {
				if i+1 == len(k) || (k[i+1] != '0' && k[i+1] != '1') {
					return nil, fmt.Errorf("invalid JSON Pointer escape")
				}
				i++
			}
		}
		p = append(p, Field(strings.ReplaceAll(strings.ReplaceAll(k, "~1", "/"), "~0", "~")))
	}
	return p, nil
}
func Get(v any, p Path) (any, bool) {
	for _, s := range p {
		switch s.Kind {
		case "field":
			m, ok := v.(map[string]any)
			if !ok {
				return nil, false
			}
			v, ok = m[s.Key]
			if !ok {
				return nil, false
			}
		case "index":
			a, ok := v.([]any)
			if !ok || s.Index < 0 || s.Index >= len(a) {
				return nil, false
			}
			v = a[s.Index]
		default:
			return nil, false
		}
	}
	return v, true
}

type QueryDocument struct {
	Version    int         `json:"version"`
	Operations []Operation `json:"operations"`
}

func New() QueryDocument { return QueryDocument{Version: SchemaVersion, Operations: []Operation{}} }
func (q QueryDocument) Clone() QueryDocument {
	b, _ := json.Marshal(q)
	var c QueryDocument
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.UseNumber()
	_ = d.Decode(&c)
	return c
}

type Operation struct {
	ID         string       `json:"id"`
	Kind       string       `json:"kind"`
	Path       Path         `json:"path,omitempty"`
	Fields     []Projection `json:"fields,omitempty"`
	Predicate  *Predicate   `json:"predicate,omitempty"`
	Transform  *Transform   `json:"transform,omitempty"`
	Body       []Operation  `json:"body,omitempty"`
	Descending bool         `json:"descending,omitempty"`
	// From identifies the first operation enclosed by Collect; empty means root.
	From string `json:"from,omitempty"`
}
type Projection struct {
	Name string `json:"name"`
	Path Path   `json:"path"`
}
type Predicate struct {
	Kind     string      `json:"kind"`
	Path     Path        `json:"path,omitempty"`
	Value    any         `json:"value,omitempty"`
	Children []Predicate `json:"children,omitempty"`
}
type Transform struct {
	Kind        string `json:"kind"`
	Value       any    `json:"value,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}
type Diagnostic struct {
	OperationID string `json:"operation_id"`
	Message     string `json:"message"`
}

func (d Diagnostic) Error() string { return d.OperationID + ": " + d.Message }

// Timeline snapshots only query definitions, never input data.
type Timeline struct {
	Current      QueryDocument
	Past, Future []QueryDocument
	Revision     uint64
}

func NewTimeline(q QueryDocument) Timeline { return Timeline{Current: q.Clone(), Revision: 1} }
func (t *Timeline) Commit(q QueryDocument) {
	t.Past = append(t.Past, t.Current.Clone())
	t.Current = q.Clone()
	t.Future = nil
	t.Revision++
}
func (t *Timeline) Undo() bool {
	if len(t.Past) == 0 {
		return false
	}
	t.Future = append(t.Future, t.Current)
	t.Current = t.Past[len(t.Past)-1]
	t.Past = t.Past[:len(t.Past)-1]
	t.Revision++
	return true
}
func (t *Timeline) Redo() bool {
	if len(t.Future) == 0 {
		return false
	}
	t.Past = append(t.Past, t.Current)
	t.Current = t.Future[len(t.Future)-1]
	t.Future = t.Future[:len(t.Future)-1]
	t.Revision++
	return true
}
