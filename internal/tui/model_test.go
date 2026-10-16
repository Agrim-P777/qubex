package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"encoding/json"
	"github.com/Agrim-P777/qubex/internal/data"
	"github.com/Agrim-P777/qubex/internal/execute"
	"github.com/Agrim-P777/qubex/internal/history"
	"github.com/Agrim-P777/qubex/internal/query"
	"github.com/Agrim-P777/qubex/internal/tree"
	"strings"
	"testing"
)

const services = `{"services":[{"name":"api","language":"python","replicas":3},{"name":"worker","language":"python","replicas":8},{"name":"web","language":"go","replicas":2}]}`

func testModel(t *testing.T) *Model {
	t.Helper()
	d, e := data.Read(context.Background(), strings.NewReader(services), "services.json", 1<<20, nil)
	if e != nil {
		t.Fatal(e)
	}
	m := New(Options{Store: history.Store{Dir: t.TempDir()}})
	t.Cleanup(m.Close)
	_, cmd := m.Update(loaded{d, nil})
	if cmd != nil {
		m.Update(cmd())
	}
	return m
}
func executeCmd(m *Model, cmd tea.Cmd) {
	if cmd != nil {
		m.Update(cmd())
	}
}
func TestAcceptanceWorkflow(t *testing.T) {
	m := testModel(t)
	m.source.Jump(tree.Ref{Path: query.Path{query.Field("services")}})
	executeCmd(m, m.begin("iterate"))
	m.source.Jump(tree.Ref{Path: query.Path{query.Field("services"), query.Index(0), query.Field("language")}})
	m.begin("filter")
	if m.form == nil {
		t.Fatal("no filter")
	}
	executeCmd(m, m.applyForm())
	m.begin("project")
	m.form.Checks["name"] = true
	m.form.Checks["replicas"] = true
	executeCmd(m, m.applyForm())
	m.begin("sort")
	m.form.set("Key", "/replicas")
	m.form.set("Direction", "descending")
	executeCmd(m, m.applyForm())
	if len(m.diagnostics) > 0 {
		t.Fatal(m.diagnostics)
	}
	if len(m.timeline.Current.Operations) != 5 {
		t.Fatal(m.timeline.Current)
	}
	b, _ := json.Marshal(m.result.Values)
	if string(b) != `[[{"name":"worker","replicas":8},{"name":"api","replicas":3}]]` {
		t.Fatal(string(b), m.notice)
	}
	m.saveForm()
	m.form.set("Name", "pending services")
	executeCmd(m, m.applyForm())
	records, e := m.opts.Store.List()
	if e != nil || len(records) != 1 {
		t.Fatal(records, e)
	}
	m.timeline.Undo()
	if len(m.timeline.Current.Operations) != 3 {
		t.Fatal("collect and sort were not atomic")
	}
	m.timeline.Redo()
	if len(m.timeline.Current.Operations) != 5 {
		t.Fatal("redo")
	}
}
func TestNavigationAndCancelledForm(t *testing.T) {
	m := testModel(t)
	before := m.compiled.Text
	rev := m.timeline.Revision
	for _, r := range []rune{'j', 'l', 'k', 'h', 'g'} {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if m.compiled.Text != before || m.timeline.Revision != rev {
		t.Fatal("navigation changed query")
	}
	m.source.Jump(tree.Ref{Path: query.Path{query.Field("services")}})
	m.begin("map")
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if len(m.timeline.Current.Operations) != 0 || m.form != nil {
		t.Fatal("cancel committed")
	}
}
func TestStaleResultsAndInvalidDraft(t *testing.T) {
	m := testModel(t)
	token := m.runToken
	m.Update(executed{result: execute.ExecutionResult{Revision: token - 1, Values: []any{"stale"}}})
	if m.result.Revision != token {
		t.Fatal("accepted stale result")
	}
	q := query.QueryDocument{Version: 1, Operations: []query.Operation{{ID: "p", Kind: "project", Fields: []query.Projection{{Name: "a", Path: query.Path{query.Field("services")}}}}, {ID: "s", Kind: "select", Path: query.Path{query.Field("removed")}}}}
	cmd := m.commit(q)
	if cmd != nil || len(m.diagnostics) == 0 || len(m.timeline.Current.Operations) != 2 {
		t.Fatal("invalid draft not retained", m.diagnostics)
	}
}
func TestScopeChooserAndGuard(t *testing.T) {
	m := testModel(t)
	m.source.Jump(tree.Ref{Path: query.Path{query.Field("services"), query.Index(0), query.Field("language")}})
	m.begin("filter")
	if m.menu == nil || m.menu.Kind != "scope" {
		t.Fatal("scope not explicit")
	}
	m.menu = nil
	m.chooseScope("0")
	m.form.set("Guard", "type")
	executeCmd(m, m.applyForm())
	if m.result.Count != 2 {
		t.Fatal(m.result, m.notice)
	}
}
func TestRenderSizes(t *testing.T) {
	m := testModel(t)
	for _, width := range []int{30, 80, 120} {
		m.Update(tea.WindowSizeMsg{Width: width, Height: 24})
		v := m.View()
		if !strings.Contains(v.Content, "QUBEX") || !strings.Contains(v.Content, "JQ") {
			t.Fatal(v.Content)
		}
	}
}

func TestPasteAndInvalidPreviewInvalidateOldWork(t *testing.T) {
	m := testModel(t)
	m.saveForm()
	m.Update(tea.PasteMsg{Content: "pasted name"})
	if m.form.value("Name") != "pasted name" {
		t.Fatal(m.form.value("Name"))
	}
	m.form = nil
	m.source.Jump(tree.Ref{Path: query.Path{query.Field("services")}})
	executeCmd(m, m.begin("iterate"))
	m.source.Jump(tree.Ref{Path: query.Path{query.Field("services"), query.Index(0), query.Field("replicas")}})
	m.begin("filter")
	old := m.runToken
	m.form.set("Value", "")
	m.previewForm()
	if m.runToken == old || m.form.Error == "" {
		t.Fatal("incomplete form did not invalidate execution")
	}
	m.Update(executed{result: execute.ExecutionResult{Revision: old, Values: []any{"stale"}}, draft: true})
	if len(m.output.Roots) == 1 && m.output.Roots[0] == "stale" {
		t.Fatal("old preview accepted")
	}
}

func TestPipelineEditingKeepsFollowingStepsAndGuards(t *testing.T) {
	m := testModel(t)
	q := query.QueryDocument{Version: 1, Operations: []query.Operation{{ID: "i", Kind: "iterate", Path: query.Path{query.Field("services")}}, {ID: "f", Kind: "filter", Predicate: &query.Predicate{Kind: "and", Children: []query.Predicate{{Kind: "exists", Path: query.Path{query.Field("language")}}, {Kind: "eq", Path: query.Path{query.Field("language")}, Value: "python"}}}}, {ID: "p", Kind: "project", Fields: []query.Projection{{Name: "label", Path: query.Path{query.Field("name")}}}}}}
	executeCmd(m, m.commit(q))
	m.editOperation(1)
	if m.form.value("Guard") != "exists" {
		t.Fatal("guard lost")
	}
	m.form.set("Value", "go")
	executeCmd(m, m.applyForm())
	if len(m.timeline.Current.Operations) != 3 || m.timeline.Current.Operations[2].ID != "p" {
		t.Fatal("suffix lost")
	}
	m.editOperation(2)
	executeCmd(m, m.applyForm())
	b, _ := json.Marshal(m.result.Values)
	if string(b) != `[{"label":"web"}]` {
		t.Fatal(string(b), m.notice)
	}
}

func TestFieldSearchKeepsHiddenSelections(t *testing.T) {
	m := testModel(t)
	m.source.Jump(tree.Ref{Path: query.Path{query.Field("services")}})
	executeCmd(m, m.begin("iterate"))
	m.begin("project")
	m.form.Checks["name"] = true
	m.form.FieldSearch.SetValue("repl")
	if names := m.form.visibleNames(); len(names) != 1 || names[0] != "replicas" {
		t.Fatal(names)
	}
	m.updateForm(tea.KeyPressMsg{Code: ' ', Text: " "})
	executeCmd(m, m.applyForm())
	if len(m.timeline.Current.Operations[1].Fields) != 2 {
		t.Fatal("hidden selection was lost")
	}
}

func TestOpeningShortcutIsNotInsertedInForm(t *testing.T) {
	m := testModel(t)
	m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if m.form == nil || m.form.value("Name") != "" {
		t.Fatal("save shortcut leaked into name")
	}
	m.form = nil
	m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	if m.form == nil || m.form.value("Search") != "" {
		t.Fatal("search shortcut leaked into search")
	}
}
