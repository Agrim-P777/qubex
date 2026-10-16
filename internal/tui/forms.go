package tui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Agrim-P777/qubex/internal/compiler"
	engine "github.com/Agrim-P777/qubex/internal/context"
	"github.com/Agrim-P777/qubex/internal/execute"
	"github.com/Agrim-P777/qubex/internal/query"
	"github.com/Agrim-P777/qubex/internal/tree"
	"io"
	"strconv"
	"strings"
	"time"
)

func newField(label, value string, choices ...string) field {
	i := textinput.New()
	i.SetVirtualCursor(true)
	i.SetValue(value)
	i.SetWidth(52)
	i.CharLimit = 4096
	return field{label, i, choices}
}
func (f *form) value(name string) string {
	for _, v := range f.Fields {
		if v.Label == name {
			return v.Input.Value()
		}
	}
	return ""
}
func (f *form) set(name, value string) {
	for i := range f.Fields {
		if f.Fields[i].Label == name {
			f.Fields[i].Input.SetValue(value)
		}
	}
}
func (m *Model) showForm(f *form) {
	f.Edit = -1
	f.FieldSearch = textinput.New()
	f.FieldSearch.Placeholder = "Search fields · Ctrl+N adds an exact field name"
	f.FieldSearch.SetWidth(52)
	f.FieldSearch.SetVirtualCursor(true)
	m.form = f
	if len(f.Fields) > 0 {
		f.Fields[0].Input.Focus()
	}
}
func (m *Model) begin(action string) tea.Cmd {
	switch action {
	case "history":
		return m.loadHistory()
	case "pipeline":
		m.pipeline = true
		return nil
	case "switch":
		m.focusOutput = false
		if m.view == "source" {
			m.view = "stage"
		} else {
			m.view = "source"
		}
		return nil
	case "rerun":
		return m.evaluate(m.timeline.Current, false, -1)
	case "collapse":
		if t := m.active(); t != nil {
			t.Expanded = map[string]bool{}
			t.AllExpanded = false
			t.Jump(tree.Ref{})
		}
		return nil
	case "expand":
		if t := m.active(); t != nil {
			t.Expanded = map[string]bool{}
			t.AllExpanded = true
		}
		return nil
	case "help":
		m.openMenu("Qubex help", "help", []menuItem{{"", "Navigation", "↑↓/jk move · ←→/hl collapse/expand · g root · v source/stage"}, {"", "Build a query", "? actions · f filter · m transform · o sort · Enter only expands"}, {"", "Refine", "Ctrl+P pipeline · u undo · Ctrl+R redo · s save · Ctrl+H history"}, {"", "Inspect", "Tab output focus · Ctrl+O panel · / search · n/N matches · w wrap · J compact"}, {"", "Export", "c copies query (selected result when output focused) · [/] scroll query"}, {"", "Forms", "Tab switches fields · ←→ chooses options · Enter applies · Esc cancels"}, {"", "Scopes", "Source paths are mapped only when safe. Use Stage for transformed fields."}, {"", "Data", "Field suggestions are sampled; strict jq behavior applies to all records."}})
		return nil
	case "limits":
		m.showForm(&form{Kind: "limits", Title: "Preview limits", Fields: []field{newField("Timeout", m.opts.Limits.Timeout.String()), newField("Results", strconv.Itoa(m.opts.Limits.Results)), newField("Bytes", strconv.FormatInt(m.opts.Limits.Bytes, 10))}})
		return nil
	case "copy_path":
		m.notice = "Copied to clipboard"
		return writeClipboard(m.cursor().Path.String())
	case "copy_pointer":
		m.notice = "Copied to clipboard"
		return writeClipboard(m.cursor().Path.Pointer())
	case "copy_value":
		v := m.cursor().Value
		b, e := json.Marshal(v)
		if e != nil {
			m.notice = e.Error()
			return nil
		}
		m.notice = "Copied to clipboard"
		return writeClipboard(string(b))
	}
	s := m.currentStage()
	if len(m.diagnostics) > 0 {
		m.notice = "Repair the invalid pipeline step or undo before adding operations"
		return nil
	}
	c := m.cursor()
	r := engine.Resolve(c, s)
	stageAction := action == "sort" || action == "unique" || action == "project" || action == "collect" || action == "count"
	if !r.Mapped && !stageAction {
		m.notice = r.Reason
		return nil
	}
	if action == "filter" && !s.Stream && m.inspect < 0 {
		// Choosing a collection is necessary before interpreting a source field
		// as a predicate on elements rather than a predicate on the whole root.
		var roots any
		if c.View == "source" {
			roots = m.doc.Value
		} else if t := m.active(); t != nil && t.Cursor.Result < len(t.Roots) {
			roots = t.Roots[t.Cursor.Result]
		}
		scopes := engine.AncestorArrays(roots, c.Path)
		if len(scopes) > 0 {
			items := []menuItem{{"root", "Filter current query values", "Retain current values; nested array tests use explicit Any/All"}}
			for i, p := range scopes {
				items = append(items, menuItem{strconv.Itoa(i), "Filter elements of " + p.String(), "Iterate this collection, then filter its elements"})
			}
			m.pendingScopes = scopes
			m.pendingCursor = c
			m.openMenu("Choose filter scope", "scope", items)
			return nil
		}
	}
	return m.actionForm(action, r.Relative, nil)
}
func (m *Model) chooseScope(id string) tea.Cmd {
	if id == "root" {
		r := engine.Resolve(m.pendingCursor, m.currentStage())
		return m.actionForm("filter", r.Relative, nil)
	}
	i, e := strconv.Atoi(id)
	if e != nil || i < 0 || i >= len(m.pendingScopes) {
		return nil
	}
	scope := m.pendingScopes[i]
	r := engine.Resolve(engine.Cursor{View: m.pendingCursor.View, Path: scope}, m.currentStage())
	if !r.Mapped {
		m.notice = r.Reason
		return nil
	}
	prefix := []query.Operation{{ID: m.id(), Kind: "iterate", Path: r.Relative}}
	p := m.pendingCursor.Path
	n := len(scope)
	if n < len(p) && (p[n].Kind == "index" || p[n].Kind == "each") {
		n++
	}
	return m.actionForm("filter", p[n:], prefix)
}
func (m *Model) actionForm(action string, p query.Path, prefix []query.Operation) tea.Cmd {
	q := m.base()
	s := m.currentStage()
	base := q.Clone()
	q.Operations = append(q.Operations, prefix...)
	if len(prefix) > 0 {
		st, _ := query.Analyze(m.doc.Shape, q)
		s = st[len(st)-1]
	}
	f := &form{Kind: action, Title: action, Base: base, Prefix: prefix, Path: p, Edit: -1}
	v := m.cursor().Value
	switch action {
	case "select", "iterate", "length", "collect", "count":
		if action == "length" && len(p) > 0 {
			q.Operations = append(q.Operations, query.Operation{ID: m.id(), Kind: "select", Path: p})
		}
		op := query.Operation{ID: m.id(), Kind: action}
		if action == "select" || action == "iterate" {
			op.Path = p
		}
		q.Operations = append(q.Operations, engine.Propose(s, op, m.id()).Operations...)
		return m.commit(q)
	case "filter":
		f.Title = "Filter values · strict jq semantics"
		val, typ := typedDefault(v)
		f.Fields = []field{newField("Path", p.Pointer()), newField("Operator", "eq", "eq", "ne", "gt", "ge", "lt", "le", "contains", "startswith", "endswith", "exists", "type"), newField("Value type", typ, "string", "number", "boolean", "null", "json"), newField("Value", val), newField("Guard", "strict", "strict", "exists", "type")}
		for i, seg := range p {
			if seg.Kind == "index" || seg.Kind == "each" {
				f.Fields = append(f.Fields, newField("Array "+p[:i].String(), "choose", "choose", "any", "all"))
			}
		}
	case "project":
		f.Title = "Select fields · Space toggles · Enter applies"
		f.Checks = map[string]bool{}
		f.Names = s.Shape.Names()
		if len(f.Names) == 0 {
			f.Fields = []field{newField("Fields JSON", `["name"]`)}
		} else {
			for _, name := range f.Names {
				f.Checks[name] = false
			}
		}
	case "sort", "unique":
		f.Title = "Collection " + action
		names := s.Shape.Names()
		if s.Shape.Allows("array") && s.Shape.Element != nil {
			names = s.Shape.Element.Names()
		}
		key := ""
		if len(p) > 0 && p[len(p)-1].Kind == "field" {
			key = query.Path{p[len(p)-1]}.Pointer()
		}
		f.Fields = []field{newField("Key", key)}
		if len(names) > 0 {
			var sug []string
			for _, k := range names {
				sug = append(sug, query.Path{query.Field(k)}.Pointer())
			}
			f.Fields[0].Input.SetSuggestions(sug)
			f.Fields[0].Input.ShowSuggestions = true
		}
		if action == "sort" {
			f.Fields = append(f.Fields, newField("Direction", "ascending", "ascending", "descending"))
		}
		if s.Stream {
			choices := []string{""}
			if s.Shape.Types["array"] {
				choices = append(choices, "@arrays")
			}
			iterations := 0
			for _, previous := range q.Operations {
				if previous.Kind == "iterate" {
					iterations++
					if iterations > 1 {
						choices = append(choices, previous.ID)
					}
				}
			}
			if len(choices) > 1 {
				f.Fields = append(f.Fields, newField("Boundary", "", choices...))
			}
		}
	case "extract_transform", "update":
		f.Title = map[string]string{"extract_transform": "Extract and transform", "update": "Update field in object"}[action]
		kind := "upper"
		if query.TypeOf(v) == "number" {
			kind = "add"
		}
		if query.TypeOf(v) == "array" {
			kind = "join"
		}
		f.Fields = []field{newField("Path", p.Pointer()), newField("Transform", kind, "upper", "lower", "trim", "replace", "split", "join", "add", "subtract", "multiply", "divide", "floor", "ceil", "round"), newField("Argument", ""), newField("Replacement", "")}
	case "map":
		f.Title = "Map array elements"
		f.Fields = []field{newField("Array", p.Pointer()), newField("Mode", "extract", "extract", "project", "transform", "update"), newField("Element path", ""), newField("Fields JSON", `["name"]`), newField("Transform", "upper", "upper", "lower", "trim", "replace", "split", "join", "add", "subtract", "multiply", "divide", "floor", "ceil", "round"), newField("Argument", ""), newField("Replacement", "")}
	default:
		m.notice = "Unknown action: " + action
		return nil
	}
	m.showForm(f)
	if f.Kind == "filter" {
		names := s.Shape.Names()
		var suggestions []string
		for _, k := range names {
			suggestions = append(suggestions, query.Path{query.Field(k)}.Pointer())
		}
		f.Fields[0].Input.SetSuggestions(suggestions)
		f.Fields[0].Input.ShowSuggestions = true
	}
	return m.previewForm()
}
func typedDefault(v any) (string, string) {
	switch x := v.(type) {
	case string:
		return x, "string"
	case nil:
		return "", "null"
	default:
		b, _ := json.Marshal(v)
		t := query.TypeOf(v)
		if t != "number" && t != "boolean" {
			t = "json"
		}
		return string(b), t
	}
}
func parseValue(value, typ string) (any, error) {
	if typ == "string" {
		return value, nil
	}
	if typ == "null" {
		return nil, nil
	}
	d := json.NewDecoder(strings.NewReader(value))
	d.UseNumber()
	var v any
	if e := d.Decode(&v); e != nil {
		return nil, fmt.Errorf("invalid %s value: %w", typ, e)
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return nil, fmt.Errorf("enter exactly one JSON value")
	}
	if typ != "json" && query.TypeOf(v) != typ {
		return nil, fmt.Errorf("expected %s value", typ)
	}
	return v, nil
}
func (m *Model) formQuery() (query.QueryDocument, error) {
	f := m.form
	q := f.Base.Clone()
	q.Operations = append(q.Operations, f.Prefix...)
	st, _ := query.Analyze(m.doc.Shape, q)
	s := st[len(st)-1]
	readPath := func(label string, shape *query.Shape) (query.Path, error) {
		p, e := query.ParsePath(f.value(label))
		if e != nil {
			return nil, e
		}
		return query.ResolvePointer(shape, p), nil
	}
	op := query.Operation{ID: "draft-operation", Kind: f.Kind}
	if f.Edit >= 0 && f.Edit < len(m.timeline.Current.Operations) {
		op.ID = m.timeline.Current.Operations[f.Edit].ID
	}
	switch f.Kind {
	case "filter":
		p, e := readPath("Path", s.Shape)
		if e != nil {
			return q, e
		}
		if f.value("Path") == f.Path.Pointer() {
			p = f.Path
		}
		v, e := parseValue(f.value("Value"), f.value("Value type"))
		if e != nil && f.value("Operator") != "exists" {
			return q, e
		}
		var quant []string
		for _, field := range f.Fields {
			if strings.HasPrefix(field.Label, "Array ") {
				quant = append(quant, field.Input.Value())
			}
		}
		pred, e := engine.RelativePredicate(p, f.value("Operator"), v, quant)
		if e != nil {
			return q, e
		}
		if g := f.value("Guard"); g != "strict" {
			guardType := query.TypeOf(v)
			if f.value("Operator") == "type" {
				if value, ok := v.(string); ok {
					guardType = value
				}
			}
			pred = guardLeaf(pred, g, guardType)
		}
		op.Predicate = &pred
	case "project":
		var names []string
		if len(f.Names) > 0 {
			for _, k := range f.Names {
				if f.Checks[k] {
					names = append(names, k)
				}
			}
		} else {
			if e := json.Unmarshal([]byte(f.value("Fields JSON")), &names); e != nil {
				return q, e
			}
		}
		if len(names) == 0 {
			return q, fmt.Errorf("select at least one field")
		}
		for _, k := range names {
			projection := query.Projection{Name: k, Path: query.Path{query.Field(k)}}
			if f.Edit >= 0 {
				for _, previous := range m.timeline.Current.Operations[f.Edit].Fields {
					if previous.Name == k {
						projection = previous
						break
					}
				}
			}
			op.Fields = append(op.Fields, projection)
		}
	case "sort", "unique":
		shape := s.Shape
		if shape.Allows("array") && !s.Stream {
			shape = shape.Element
		}
		p, e := readPath("Key", shape)
		if e != nil {
			return q, e
		}
		op.Path = p
		op.Descending = f.value("Direction") == "descending"
		if boundary := f.value("Boundary"); boundary == "@arrays" {
			s.Stream = false
		} else {
			op.From = boundary
		}
	case "extract_transform", "update":
		p, e := readPath("Path", s.Shape)
		if e != nil {
			return q, e
		}
		t, e := readTransform(f)
		if e != nil {
			return q, e
		}
		op.Path = p
		op.Transform = &t
	case "map":
		p, e := readPath("Array", s.Shape)
		if e != nil {
			return q, e
		}
		a, e := query.At(s.Shape, p)
		if e != nil {
			return q, e
		}
		if len(p) > 0 {
			q.Operations = append(q.Operations, query.Operation{ID: "draft-select", Kind: "select", Path: p})
		}
		ep, e := readPath("Element path", a.Element)
		if e != nil {
			return q, e
		}
		child := query.Operation{ID: "draft-child"}
		switch f.value("Mode") {
		case "extract":
			child.Kind = "select"
			child.Path = ep
		case "project":
			child.Kind = "project"
			var names []string
			if e := json.Unmarshal([]byte(f.value("Fields JSON")), &names); e != nil {
				return q, e
			}
			for _, k := range names {
				child.Fields = append(child.Fields, query.Projection{Name: k, Path: query.Path{query.Field(k)}})
			}
		case "transform", "update":
			t, e := readTransform(f)
			if e != nil {
				return q, e
			}
			child.Transform = &t
			child.Kind = f.value("Mode")
			if child.Kind == "update" {
				child.Path = ep
			} else if len(ep) > 0 {
				op.Body = append(op.Body, query.Operation{ID: "draft-child-select", Kind: "select", Path: ep})
			}
		}
		op.Body = append(op.Body, child)
	case "edit_path":
		op = f.Original
		var e error
		op.Path, e = readPath("Path", s.Shape)
		if e != nil {
			return q, e
		}
		op.From = f.value("Boundary")
	default:
		return q, fmt.Errorf("not a query form")
	}
	q.Operations = append(q.Operations, engine.Propose(s, op, "draft-auxiliary").Operations...)
	if f.Edit >= 0 {
		q.Operations = append(q.Operations, m.timeline.Current.Clone().Operations[f.Edit+1:]...)
	}
	return q, nil
}
func guardLeaf(p query.Predicate, kind, typ string) query.Predicate {
	if p.Kind == "any" || p.Kind == "all" {
		p.Children[0] = guardLeaf(p.Children[0], kind, typ)
		return p
	}
	g := query.Predicate{Kind: kind, Path: p.Path}
	if kind == "type" {
		g.Value = typ
	}
	return query.Predicate{Kind: "and", Children: []query.Predicate{g, p}}
}
func readTransform(f *form) (query.Transform, error) {
	t := query.Transform{Kind: f.value("Transform"), Replacement: f.value("Replacement")}
	switch t.Kind {
	case "add", "subtract", "multiply", "divide":
		v, e := parseValue(f.value("Argument"), "number")
		if e != nil {
			return t, e
		}
		t.Value = v
	case "replace", "split", "join":
		t.Value = f.value("Argument")
	}
	return t, nil
}
func (m *Model) previewForm() tea.Cmd {
	m.token++
	f := m.form
	if f == nil || f.Kind == "save" || f.Kind == "search" || f.Kind == "limits" {
		return nil
	}
	if f.Kind == "filter" {
		m.syncFilterArrays()
	}
	// Invalidate even when the new text is incomplete, before the debounce.
	if m.runCancel != nil {
		m.runCancel()
	}
	m.runToken++
	m.output = tree.New(m.lastGood.Values, true)
	m.notice = "Editing draft · output shows the last committed result until preview completes"
	q, e := m.formQuery()
	if e != nil {
		f.Error = e.Error()
		f.Preview = ""
		return nil
	}
	c, e := compiler.Compile(q)
	if e != nil {
		f.Error = e.Error()
		return nil
	}
	f.Error = ""
	f.Preview = c.Text
	token := m.token
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return debounced{token} })
}

// A manually changed pointer can introduce another collection boundary.
// Keep explicit choices for unchanged boundaries and ask about new ones.
func (m *Model) syncFilterArrays() {
	f := m.form
	if f == nil || f.Kind != "filter" || len(f.Fields) < 5 {
		return
	}
	p, e := query.ParsePath(f.value("Path"))
	if e != nil {
		return
	}
	if f.value("Path") == f.Path.Pointer() {
		p = f.Path
	} else {
		q := f.Base.Clone()
		q.Operations = append(q.Operations, f.Prefix...)
		st, _ := query.Analyze(m.doc.Shape, q)
		p = query.ResolvePointer(st[len(st)-1].Shape, p)
	}
	choices := map[string]string{}
	for _, field := range f.Fields[5:] {
		choices[field.Label] = field.Input.Value()
	}
	fields := append([]field{}, f.Fields[:5]...)
	for i, segment := range p {
		if segment.Kind == "index" || segment.Kind == "each" {
			label := "Array " + p[:i].String()
			v := choices[label]
			if v == "" {
				v = "choose"
			}
			fields = append(fields, newField(label, v, "choose", "any", "all"))
		}
	}
	f.Fields = fields
	f.Path = p
	f.Focus = min(f.Focus, len(fields)-1)
}
func (m *Model) updateForm(k tea.KeyPressMsg) tea.Cmd {
	f := m.form
	key := k.String()
	if f.SearchingFields {
		switch key {
		case "esc":
			f.SearchingFields = false
			f.FieldSearch.Blur()
			return nil
		case "down", "up":
			f.SearchingFields = false
			f.FieldSearch.Blur()
		case "ctrl+n":
			name := f.FieldSearch.Value()
			if !contains(f.Names, name) {
				f.Names = append(f.Names, name)
			}
			f.Checks[name] = true
			f.SearchingFields = false
			f.FieldSearch.Blur()
			return m.previewForm()
		case "enter":
			f.SearchingFields = false
			f.FieldSearch.Blur()
			return nil
		default:
			var cmd tea.Cmd
			f.FieldSearch, cmd = f.FieldSearch.Update(k)
			f.CheckCursor = 0
			return cmd
		}
	}
	if key == "esc" {
		m.form = nil
		m.token++
		return m.evaluate(m.base(), false, m.inspect)
	}
	if key == "enter" {
		return m.applyForm()
	}
	if len(f.Names) > 0 {
		names := f.visibleNames()
		switch key {
		case "/":
			f.SearchingFields = true
			return f.FieldSearch.Focus()
		case "up", "k":
			f.CheckCursor = max(0, f.CheckCursor-1)
		case "down", "j":
			f.CheckCursor = min(max(0, len(names)-1), f.CheckCursor+1)
		case "space":
			if len(names) > 0 {
				f.Checks[names[f.CheckCursor]] = !f.Checks[names[f.CheckCursor]]
				return m.previewForm()
			}
		}
		return nil
	}
	if len(f.Fields) == 0 {
		return nil
	}
	switch key {
	case "tab", "down", "shift+tab", "up":
		f.Fields[f.Focus].Input.Blur()
		d := 1
		if key == "shift+tab" || key == "up" {
			d = -1
		}
		f.Focus = (f.Focus + d + len(f.Fields)) % len(f.Fields)
		return f.Fields[f.Focus].Input.Focus()
	case "left", "right":
		field := &f.Fields[f.Focus]
		if len(field.Choices) > 0 {
			i := 0
			for j, v := range field.Choices {
				if v == field.Input.Value() {
					i = j
					break
				}
			}
			d := 1
			if key == "left" {
				d = -1
			}
			field.Input.SetValue(field.Choices[(i+d+len(field.Choices))%len(field.Choices)])
			return m.previewForm()
		}
	}
	if len(f.Fields[f.Focus].Choices) > 0 {
		return nil
	}
	var cmd tea.Cmd
	f.Fields[f.Focus].Input, cmd = f.Fields[f.Focus].Input.Update(k)
	return tea.Batch(cmd, m.previewForm())
}

func (f *form) visibleNames() []string {
	var names []string
	for _, name := range f.Names {
		if tree.Match(name, f.FieldSearch.Value(), false, false) {
			names = append(names, name)
		}
	}
	return names
}
func (m *Model) applyForm() tea.Cmd {
	f := m.form
	switch f.Kind {
	case "save":
		name := f.value("Name")
		q := m.timeline.Current.Clone()
		source := "stdin"
		if m.doc.Source != "stdin" {
			source = m.doc.Source
		}
		store := m.opts.Store
		m.form = nil
		return func() tea.Msg { r, e := store.Save(name, q, source); return saved{r, e} }
	case "limits":
		d, e := time.ParseDuration(f.value("Timeout"))
		if e != nil || d <= 0 {
			f.Error = "Timeout must be positive, e.g. 10s"
			return nil
		}
		n, e := strconv.Atoi(f.value("Results"))
		if e != nil || n < 1 {
			f.Error = "Results must be positive"
			return nil
		}
		b, e := strconv.ParseInt(f.value("Bytes"), 10, 64)
		if e != nil || b < 1 {
			f.Error = "Bytes must be positive"
			return nil
		}
		m.opts.Limits = execute.Limits{Timeout: d, Results: n, Bytes: b}
		m.form = nil
		return m.evaluate(m.timeline.Current, false, -1)
	case "search":
		t := m.active()
		if t == nil {
			return nil
		}
		o := tree.SearchOptions{Text: f.value("Search"), Target: f.value("Target"), Exact: f.value("Mode") == "exact", CaseSensitive: f.value("Case") == "sensitive", Limit: 1000}
		if f.value("Scope") == "subtree" {
			r := t.Cursor
			o.Root = &r
		}
		m.form = nil
		if m.searchCancel != nil {
			m.searchCancel()
		}
		ctx, cancel := context.WithCancel(m.ctx)
		m.searchCancel = cancel
		m.searchToken++
		token := m.searchToken
		m.notice = "Searching…"
		roots := t.Roots
		return func() tea.Msg { r, e := tree.Search(ctx, roots, o); return searched{r, e, token, t} }
	}
	q, e := m.formQuery()
	if e != nil {
		f.Error = e.Error()
		return nil
	}
	if _, e := compiler.Compile(q); e != nil {
		f.Error = e.Error()
		return nil
	}
	assignDraftIDs(q.Operations, m.id)
	if f.Edit >= 0 {
		m.inspect = -1
	}
	return m.commit(q)
}
func assignDraftIDs(ops []query.Operation, id func() string) {
	for i := range ops {
		if strings.HasPrefix(ops[i].ID, "draft-") {
			ops[i].ID = id()
		}
		assignDraftIDs(ops[i].Body, id)
	}
}
func (m *Model) saveForm() {
	m.showForm(&form{Kind: "save", Title: "Save query definition · input data is never stored", Fields: []field{newField("Name", "")}})
}
func (m *Model) searchForm() {
	m.showForm(&form{Kind: "search", Title: "Search JSON · bounded to 1000 matches", Fields: []field{newField("Search", ""), newField("Mode", "fuzzy", "fuzzy", "exact"), newField("Target", "both", "both", "keys", "values"), newField("Scope", "document", "document", "subtree"), newField("Case", "insensitive", "insensitive", "sensitive")}})
}
func (m *Model) editOperation(i int) tea.Cmd {
	op := m.timeline.Current.Operations[i]
	m.pipeline = false
	oldInspect := m.inspect
	m.inspect = i - 1
	if i == 0 {
		m.inspect = -1
	}
	base := m.timeline.Current.Clone()
	base.Operations = base.Operations[:i]
	// Forms reuse semantic builders; retain all downstream steps on Apply.
	switch op.Kind {
	case "filter":
		if op.Predicate != nil {
			p := *op.Predicate
			var qs []string
			path := query.Path{}
			for p.Kind == "any" || p.Kind == "all" {
				path = append(path, p.Path...)
				path = append(path, query.Each())
				qs = append(qs, p.Kind)
				p = p.Children[0]
			}
			guard := "strict"
			if p.Kind == "and" && len(p.Children) == 2 {
				guard = p.Children[0].Kind
				p = p.Children[1]
			}
			path = append(path, p.Path...)
			m.actionForm("filter", path, nil)
			m.form.set("Operator", p.Kind)
			v, t := typedDefault(p.Value)
			m.form.set("Value", v)
			m.form.set("Value type", t)
			m.form.set("Guard", guard)
			j := 0
			for n := range m.form.Fields {
				if strings.HasPrefix(m.form.Fields[n].Label, "Array ") && j < len(qs) {
					m.form.Fields[n].Input.SetValue(qs[j])
					j++
				}
			}
		}
	case "project":
		m.actionForm("project", nil, nil)
		if m.form.Checks == nil {
			m.form.Checks = map[string]bool{}
		}
		for _, f := range op.Fields {
			if !contains(m.form.Names, f.Name) {
				m.form.Names = append(m.form.Names, f.Name)
			}
			m.form.Checks[f.Name] = true
		}
	case "sort", "unique":
		m.actionForm(op.Kind, op.Path, nil)
		m.form.set("Key", op.Path.Pointer())
		if op.Descending {
			m.form.set("Direction", "descending")
		}
	case "transform", "update":
		if op.Transform != nil {
			action := "extract_transform"
			if op.Kind == "update" {
				action = "update"
			}
			m.actionForm(action, op.Path, nil)
			m.form.set("Transform", op.Transform.Kind)
			v, _ := typedDefault(op.Transform.Value)
			m.form.set("Argument", v)
			m.form.set("Replacement", op.Transform.Replacement)
		}
	case "map":
		m.actionForm("map", nil, nil)
		m.form.set("Array", "")
		if len(op.Body) > 0 {
			child := op.Body[len(op.Body)-1]
			switch child.Kind {
			case "select":
				m.form.set("Mode", "extract")
				m.form.set("Element path", child.Path.Pointer())
			case "project":
				m.form.set("Mode", "project")
				names := []string{}
				for _, f := range child.Fields {
					names = append(names, f.Name)
				}
				b, _ := json.Marshal(names)
				m.form.set("Fields JSON", string(b))
			case "transform", "update":
				m.form.set("Mode", child.Kind)
				m.form.set("Element path", child.Path.Pointer())
				if child.Kind == "transform" && len(op.Body) > 1 && op.Body[0].Kind == "select" {
					m.form.set("Element path", op.Body[0].Path.Pointer())
				}
				if child.Transform != nil {
					m.form.set("Transform", child.Transform.Kind)
					v, _ := typedDefault(child.Transform.Value)
					m.form.set("Argument", v)
					m.form.set("Replacement", child.Transform.Replacement)
				}
			}
		}
	default:
		m.showForm(&form{Kind: "edit_path", Title: "Edit " + op.Kind, Fields: []field{newField("Path", op.Path.Pointer())}, Original: op})
		if op.Kind == "collect" || op.Kind == "count" {
			choices := []string{""}
			for _, previous := range base.Operations {
				if previous.Kind == "iterate" {
					choices = append(choices, previous.ID)
				}
			}
			m.form.Fields = append(m.form.Fields, newField("Boundary", op.From, choices...))
		}
	}
	m.inspect = oldInspect
	if m.form == nil {
		m.notice = "Inspect this operation's input to rebuild its body"
		return nil
	}
	m.form.Base = base
	m.form.Prefix = nil
	m.form.Edit = i
	if m.form.Kind == "project" {
		st, _ := query.Analyze(m.doc.Shape, base)
		names := st[len(st)-1].Shape.Names()
		for _, field := range op.Fields {
			if !contains(names, field.Name) {
				names = append(names, field.Name)
			}
		}
		m.form.Names = names
	}
	return m.previewForm()
}
func contains(a []string, s string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
