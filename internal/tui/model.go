package tui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Agrim-P777/qubex/internal/compiler"
	engine "github.com/Agrim-P777/qubex/internal/context"
	"github.com/Agrim-P777/qubex/internal/data"
	"github.com/Agrim-P777/qubex/internal/execute"
	"github.com/Agrim-P777/qubex/internal/history"
	"github.com/Agrim-P777/qubex/internal/query"
	"github.com/Agrim-P777/qubex/internal/tree"
	"io"
	"strings"
	"sync/atomic"
	"time"
)

type Options struct {
	Reader     io.Reader
	Source     string
	InputLimit int64
	Limits     execute.Limits
	Store      history.Store
	Query      query.QueryDocument
	Keys       map[string]string
}
type loaded struct {
	doc *data.Document
	err error
}
type tick struct{}
type executed struct {
	result execute.ExecutionResult
	draft  bool
	stage  int
}
type debounced struct{ token uint64 }
type searched struct {
	refs   []tree.Ref
	err    error
	token  uint64
	target *tree.Tree
}
type saved struct {
	record history.Record
	err    error
}
type recordsMsg struct {
	records []history.Record
	err     error
}
type menuItem struct{ ID, Title, Detail string }
type menu struct {
	Title, Kind string
	Items       []menuItem
	Cursor      int
	Search      textinput.Model
	Searching   bool
}
type field struct {
	Label   string
	Input   textinput.Model
	Choices []string
}
type form struct {
	Kind, Title     string
	Fields          []field
	Focus           int
	Base            query.QueryDocument
	Prefix          []query.Operation
	Path            query.Path
	Edit            int
	Checks          map[string]bool
	Names           []string
	CheckCursor     int
	Error           string
	Preview         string
	Original        query.Operation
	FieldSearch     textinput.Model
	SearchingFields bool
}
type Model struct {
	opts                         Options
	doc                          *data.Document
	timeline                     query.Timeline
	runtime                      *execute.Runtime
	source, stage, output        *tree.Tree
	view                         string
	focusOutput                  bool
	panel                        int
	width, height                int
	stages                       []query.StageContext
	diagnostics                  []query.Diagnostic
	compiled                     compiler.CompiledQuery
	result                       execute.ExecutionResult
	lastGood                     execute.ExecutionResult
	notice                       string
	menu                         *menu
	form                         *form
	history                      []history.Record
	pipeline                     bool
	pipelineCursor               int
	inspect                      int
	ctx                          context.Context
	cancel                       context.CancelFunc
	runCancel                    context.CancelFunc
	searchCancel                 context.CancelFunc
	progress                     *atomic.Int64
	loading                      bool
	token, runToken, searchToken uint64
	seq                          uint64
	matches                      []tree.Ref
	matchIndex                   int
	searchTree                   *tree.Tree
	pretty, wrap                 bool
	queryOffset                  int
	Interrupted                  bool
	InputError                   bool
	pendingScopes                []query.Path
	pendingCursor                engine.Cursor
}

func New(o Options) *Model {
	if o.Query.Version == 0 {
		o.Query = query.New()
	}
	if o.InputLimit == 0 {
		o.InputLimit = data.DefaultInputLimit
	}
	if o.Limits.Timeout == 0 {
		o.Limits = execute.PreviewLimits()
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Model{opts: o, timeline: query.NewTimeline(o.Query), runtime: execute.New(), view: "source", panel: 2, width: 100, height: 30, inspect: -1, ctx: ctx, cancel: cancel, progress: &atomic.Int64{}, loading: true, pretty: true}
}
func (m *Model) Init() tea.Cmd {
	return tea.Batch(func() tea.Msg {
		d, e := data.Read(m.ctx, m.opts.Reader, m.opts.Source, m.opts.InputLimit, func(n int64) { m.progress.Store(n) })
		return loaded{d, e}
	}, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tick{} }))
}
func (m *Model) Close() {
	m.cancel()
	if m.runCancel != nil {
		m.runCancel()
	}
	if m.searchCancel != nil {
		m.searchCancel()
	}
}
func (m *Model) id() string { m.seq++; return fmt.Sprintf("op-%d-%d", time.Now().UnixNano(), m.seq) }
func (m *Model) active() *tree.Tree {
	if m.focusOutput {
		return m.output
	}
	if m.view == "stage" {
		return m.stage
	}
	return m.source
}
func (m *Model) currentStage() query.StageContext {
	if len(m.stages) == 0 {
		return query.StageContext{Shape: query.Unknown()}
	}
	i := len(m.stages) - 1
	if m.inspect >= 0 {
		i = min(m.inspect+1, len(m.stages)-1)
	}
	return m.stages[i]
}
func (m *Model) cursor() engine.Cursor {
	t := m.active()
	if t == nil {
		return engine.Cursor{View: "stage"}
	}
	v, _ := t.Value(t.Cursor)
	view := m.view
	if m.focusOutput {
		view = "stage"
	}
	return engine.Cursor{View: view, Path: t.Cursor.Path, Value: v}
}
func (m *Model) base() query.QueryDocument {
	q := m.timeline.Current.Clone()
	if m.inspect >= 0 {
		q.Operations = q.Operations[:min(m.inspect+1, len(q.Operations))]
	}
	return q
}
func (m *Model) evaluate(q query.QueryDocument, draft bool, stage int) tea.Cmd {
	if m.runCancel != nil {
		m.runCancel()
	}
	m.runToken++
	rev := m.runToken
	ctx, cancel := context.WithCancel(m.ctx)
	m.runCancel = cancel
	if !draft {
		m.stages, m.diagnostics = query.Analyze(m.doc.Shape, m.timeline.Current)
		c, e := compiler.Compile(m.timeline.Current)
		if e != nil {
			m.compiled = compiler.CompiledQuery{}
			m.output = tree.New(m.lastGood.Values, true)
			m.notice = e.Error()
			return nil
		}
		m.compiled = c
		if stage < 0 {
			m.stage = tree.New(nil, true)
		}
		if len(m.diagnostics) > 0 {
			m.notice = "Invalid draft — repair highlighted steps or undo. Output is from an earlier revision."
			if stage < 0 {
				m.output = tree.New(m.lastGood.Values, true)
				return nil
			}
		}
	}
	_, ds := query.Analyze(m.doc.Shape, q)
	if len(ds) > 0 {
		if m.form != nil {
			m.form.Error = ds[0].Error()
		}
		return nil
	}
	rt, input, limits := m.runtime, m.doc.Value, m.opts.Limits
	m.notice = "Evaluating…"
	return func() tea.Msg { return executed{rt.Run(ctx, q, input, rev, limits, nil), draft, stage} }
}
func (m *Model) commit(q query.QueryDocument) tea.Cmd {
	if m.inspect >= 0 {
		old := m.timeline.Current.Clone()
		end := min(m.inspect+1, len(old.Operations))
		suffix := append([]query.Operation{}, old.Operations[end:]...)
		q.Operations = append(q.Operations, suffix...)
		m.inspect = -1
	}
	m.timeline.Commit(q)
	m.form = nil
	m.menu = nil
	m.pipeline = false
	return m.evaluate(q, false, -1)
}
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch x := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = x.Width
		m.height = x.Height
		return m, nil
	case loaded:
		m.loading = false
		if x.err != nil {
			m.InputError = true
			m.notice = x.err.Error()
			return m, nil
		}
		m.doc = x.doc
		m.source = tree.New([]any{x.doc.Value}, false)
		m.stage = tree.New(nil, true)
		m.output = tree.New(nil, true)
		return m, m.evaluate(m.timeline.Current, false, -1)
	case tick:
		if m.loading {
			return m, tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return tick{} })
		}
		return m, nil
	case executed:
		if x.result.Revision != m.runToken {
			return m, nil
		}
		m.result = x.result
		m.notice = x.result.Summary()
		if x.draft {
			if m.form != nil && x.result.Error != nil {
				m.form.Error = x.result.Error.Error()
			}
		} else if x.stage < 0 {
			m.stage = tree.New(x.result.Values, true)
			if x.result.Status == "complete" || x.result.Status == "empty" || x.result.Status == "truncated" {
				m.lastGood = x.result
			}
		} else {
			m.stage = tree.New(x.result.Values, true)
		}
		m.output = tree.New(x.result.Values, true)
		return m, nil
	case debounced:
		if x.token != m.token || m.form == nil {
			return m, nil
		}
		q, e := m.formQuery()
		if e != nil {
			m.form.Error = e.Error()
			return m, nil
		}
		return m, m.evaluate(q, true, -1)
	case searched:
		if x.token != m.searchToken {
			return m, nil
		}
		if x.err != nil {
			m.notice = x.err.Error()
			return m, nil
		}
		m.matches = x.refs
		m.matchIndex = 0
		m.searchTree = x.target
		if len(x.refs) > 0 {
			x.target.Jump(x.refs[0])
		}
		m.notice = fmt.Sprintf("%d matches (up to 1000) · n/N next/previous", len(x.refs))
		return m, nil
	case saved:
		if x.err != nil {
			m.notice = x.err.Error()
		} else {
			m.notice = fmt.Sprintf("Saved #%d · %s", x.record.ID, x.record.Name)
		}
		return m, nil
	case recordsMsg:
		if x.err != nil {
			m.notice = x.err.Error()
			return m, nil
		}
		m.history = x.records
		items := []menuItem{}
		for _, r := range x.records {
			items = append(items, menuItem{fmt.Sprint(r.ID), fmt.Sprintf("#%d %s", r.ID, r.Name), r.JQ})
		}
		m.openMenu("Saved queries · Enter previews, then restore", "history", items)
		return m, nil
	case tea.KeyPressMsg:
		key := x.String()
		if key == "ctrl+c" {
			m.Interrupted = true
			m.Close()
			return m, tea.Quit
		}
		if m.form != nil {
			return m, m.updateForm(x)
		}
		if m.menu != nil {
			return m, m.updateMenu(x)
		}
		if m.pipeline {
			return m, m.updatePipeline(key)
		}
		if m.loading || m.doc == nil {
			if key == "q" {
				m.Close()
				return m, tea.Quit
			}
			return m, nil
		}
		if mapped, ok := m.opts.Keys[key]; ok {
			key = mapped
		}
		switch key {
		case "q":
			m.Close()
			return m, tea.Quit
		case "up", "k":
			if t := m.active(); t != nil {
				t.Move(-1)
			}
		case "down", "j":
			if t := m.active(); t != nil {
				t.Move(1)
			}
		case "left", "h":
			if t := m.active(); t != nil {
				t.Left()
			}
		case "right", "l":
			if t := m.active(); t != nil {
				t.Right()
			}
		case "enter":
			if t := m.active(); t != nil {
				t.Toggle()
			}
		case "home", "g":
			if t := m.active(); t != nil {
				t.Jump(tree.Ref{})
			}
		case "end", "G":
			if t := m.active(); t != nil {
				t.End()
			}
		case "pgup":
			if t := m.active(); t != nil {
				t.Move(-max(1, m.height-10))
			}
		case "pgdown":
			if t := m.active(); t != nil {
				t.Move(max(1, m.height-10))
			}
		case "tab":
			m.focusOutput = !m.focusOutput
			if m.focusOutput {
				m.panel = 2
			}
		case "v":
			m.focusOutput = false
			if m.view == "source" {
				m.view = "stage"
			} else {
				m.view = "source"
			}
		case "ctrl+o":
			m.panel = (m.panel + 1) % 3
			if m.panel != 2 {
				m.focusOutput = false
			}
		case "?":
			m.palette()
		case "f":
			return m, m.begin("filter")
		case "m":
			return m, m.begin("extract_transform")
		case "o":
			return m, m.begin("sort")
		case "p":
			m.openMenu("Path actions", "action", []menuItem{{"select", "Extract current path", ""}, {"iterate", "Iterate current array", ""}, {"copy_path", "Copy jq path", ""}, {"copy_pointer", "Copy JSON Pointer", ""}})
		case "s", "ctrl+s":
			m.saveForm()
		case "c":
			return m, m.copyCurrent()
		case "u", "ctrl+u":
			if m.timeline.Undo() {
				m.inspect = -1
				return m, m.evaluate(m.timeline.Current, false, -1)
			}
		case "ctrl+r":
			if m.timeline.Redo() {
				m.inspect = -1
				return m, m.evaluate(m.timeline.Current, false, -1)
			}
		case "ctrl+h":
			return m, m.loadHistory()
		case "ctrl+p":
			m.pipeline = true
			m.pipelineCursor = 0
		case "/":
			m.searchForm()
		case "n", "N":
			if len(m.matches) > 0 {
				d := 1
				if key == "N" {
					d = -1
				}
				m.matchIndex = (m.matchIndex + d + len(m.matches)) % len(m.matches)
				m.searchTree.Jump(m.matches[m.matchIndex])
			}
		case "w":
			m.wrap = !m.wrap
		case "J":
			m.pretty = !m.pretty
		case "[":
			m.queryOffset = max(0, m.queryOffset-20)
		case "]":
			m.queryOffset += 20
		case "esc":
			if m.inspect >= 0 {
				m.inspect = -1
				return m, m.evaluate(m.timeline.Current, false, -1)
			}
		}
	}
	// Forward paste and cursor messages as well as key events to Bubbles.
	// Recompute previews only when a value changed, never on blink ticks.
	if _, handledKey := msg.(tea.KeyPressMsg); handledKey {
		return m, nil
	}
	if m.form != nil && m.form.SearchingFields {
		var cmd tea.Cmd
		m.form.FieldSearch, cmd = m.form.FieldSearch.Update(msg)
		m.form.CheckCursor = 0
		return m, cmd
	}
	if m.form != nil && len(m.form.Fields) > 0 && len(m.form.Names) == 0 {
		f := &m.form.Fields[m.form.Focus]
		if len(f.Choices) == 0 {
			before := f.Input.Value()
			var cmd tea.Cmd
			f.Input, cmd = f.Input.Update(msg)
			if before != f.Input.Value() {
				return m, tea.Batch(cmd, m.previewForm())
			}
			return m, cmd
		}
	}
	if m.menu != nil && m.menu.Searching {
		var cmd tea.Cmd
		m.menu.Search, cmd = m.menu.Search.Update(msg)
		return m, cmd
	}
	return m, nil
}
func (m *Model) copyCurrent() tea.Cmd {
	s := m.compiled.Text
	if s == "" && !m.focusOutput {
		m.notice = "Repair the invalid query before copying jq"
		return nil
	}
	if m.focusOutput {
		if t := m.active(); t != nil {
			v, ok := t.Value(t.Cursor)
			if ok {
				b, e := json.Marshal(v)
				if e != nil {
					m.notice = e.Error()
					return nil
				}
				s = string(b)
			}
		}
	}
	m.notice = "Copied to clipboard"
	return writeClipboard(s)
}
func (m *Model) loadHistory() tea.Cmd {
	s := m.opts.Store
	return func() tea.Msg { r, e := s.List(); return recordsMsg{r, e} }
}
func (m *Model) openMenu(title, kind string, items []menuItem) {
	i := textinput.New()
	i.Placeholder = "Search actions…"
	i.SetWidth(50)
	m.menu = &menu{Title: title, Kind: kind, Items: items, Search: i}
}
func (m *Model) palette() {
	items := []menuItem{}
	items = append(items, menuItem{"expand", "Expand all", "Expand the active tree lazily"})
	for _, a := range engine.Actions(m.cursor(), m.currentStage()) {
		items = append(items, menuItem{a.ID, a.Name, a.Description})
	}
	items = append(items, menuItem{"collect", "Collect results", "Make the result sequence an array"}, menuItem{"history", "Saved query history", "Browse and restore saved definitions"}, menuItem{"pipeline", "Query pipeline", "Edit, delete, move or inspect steps"}, menuItem{"limits", "Preview limits", "Increase timeout and result display limits"}, menuItem{"switch", "Switch Source / Stage", "Choose the data context for operations"}, menuItem{"rerun", "Rerun query", "Evaluate with current limits"}, menuItem{"collapse", "Collapse all", "Collapse the active tree"}, menuItem{"help", "Help", "Keyboard and query-building guidance"})
	m.openMenu("Operations · "+m.cursor().Path.String(), "action", items)
}
func (m *Model) filteredMenu() []menuItem {
	if m.menu == nil {
		return nil
	}
	var r []menuItem
	for _, i := range m.menu.Items {
		if tree.Match(i.Title+" "+i.Detail, m.menu.Search.Value(), false, false) {
			r = append(r, i)
		}
	}
	return r
}
func (m *Model) updateMenu(k tea.KeyPressMsg) tea.Cmd {
	key := k.String()
	menu := m.menu
	if key == "esc" {
		m.menu = nil
		if menu.Kind == "restore" {
			return m.evaluate(m.base(), false, m.inspect)
		}
		return nil
	}
	items := m.filteredMenu()
	switch key {
	case "up":
		menu.Cursor = max(0, menu.Cursor-1)
		return nil
	case "down":
		menu.Cursor = min(max(0, len(items)-1), menu.Cursor+1)
		return nil
	case "/":
		menu.Searching = true
		return menu.Search.Focus()
	case "?":
		if len(items) > 0 {
			m.notice = items[min(menu.Cursor, len(items)-1)].Detail
		}
		return nil
	case "enter":
		if len(items) == 0 {
			return nil
		}
		item := items[min(menu.Cursor, len(items)-1)]
		kind := menu.Kind
		m.menu = nil
		switch kind {
		case "action":
			return m.begin(item.ID)
		case "scope":
			return m.chooseScope(item.ID)
		case "history":
			for _, r := range m.history {
				if fmt.Sprint(r.ID) == item.ID {
					m.openMenu(fmt.Sprintf("#%d %s · %s", r.ID, r.Name, r.Source), "restore", []menuItem{{item.ID, "Restore this query", r.JQ}, {"cancel", "Cancel", ""}})
					m.notice = "History preview: " + r.CreatedAt.Format(time.RFC3339)
					return m.evaluate(r.Query, true, -1)
				}
			}
		case "restore":
			if item.ID != "cancel" {
				for _, r := range m.history {
					if fmt.Sprint(r.ID) == item.ID {
						if e := m.opts.Store.Touch(r.ID); e != nil {
							m.notice = e.Error()
						}
						m.inspect = -1
						return m.commit(r.Query)
					}
				}
			}
			return m.evaluate(m.base(), false, m.inspect)
		case "help":
			return nil
		}
		return nil
	}
	if menu.Searching {
		var cmd tea.Cmd
		menu.Search, cmd = menu.Search.Update(k)
		menu.Cursor = 0
		return cmd
	}
	if len(key) == 1 {
		for i, item := range items {
			if strings.HasPrefix(strings.ToLower(item.Title), strings.ToLower(key)) {
				menu.Cursor = i
				break
			}
		}
	}
	return nil
}
func (m *Model) updatePipeline(key string) tea.Cmd {
	ops := m.timeline.Current.Operations
	if key == "esc" {
		m.pipeline = false
		return nil
	}
	switch key {
	case "up", "k":
		m.pipelineCursor = max(0, m.pipelineCursor-1)
	case "down", "j":
		m.pipelineCursor = min(max(0, len(ops)-1), m.pipelineCursor+1)
	case "x", "delete":
		if len(ops) > 0 {
			q := m.timeline.Current.Clone()
			i := m.pipelineCursor
			q.Operations = append(q.Operations[:i], q.Operations[i+1:]...)
			m.timeline.Commit(q)
			m.pipelineCursor = min(i, max(0, len(q.Operations)-1))
			return m.evaluate(q, false, -1)
		}
	case "K", "J":
		if len(ops) > 1 {
			i := m.pipelineCursor
			j := i - 1
			if key == "J" {
				j = i + 1
			}
			if j >= 0 && j < len(ops) {
				q := m.timeline.Current.Clone()
				q.Operations[i], q.Operations[j] = q.Operations[j], q.Operations[i]
				m.timeline.Commit(q)
				m.pipelineCursor = j
				return m.evaluate(q, false, -1)
			}
		}
	case "i":
		if len(ops) > 0 {
			m.inspect = m.pipelineCursor
			m.pipeline = false
			m.view = "stage"
			q := m.base()
			return m.evaluate(q, false, m.inspect)
		}
	case "c":
		if len(ops) > 0 {
			if fragment, ok := m.stepExpression(ops[m.pipelineCursor].ID); ok {
				m.notice = "Copied to clipboard"
				return writeClipboard(fragment)
			}
			c, e := compiler.Compile(query.QueryDocument{Version: 1, Operations: []query.Operation{ops[m.pipelineCursor]}})
			if e == nil {
				return writeClipboard(c.Text)
			}
			m.notice = e.Error()
		}
	case "enter":
		if len(ops) > 0 {
			return m.editOperation(m.pipelineCursor)
		}
	}
	return nil
}

func (m *Model) stepExpression(id string) (string, bool) {
	l, ok := m.compiled.Locations[id]
	if !ok || l.Start < 0 || l.End > len(m.compiled.Text) {
		return "", false
	}
	return m.compiled.Text[l.Start:l.End], true
}
