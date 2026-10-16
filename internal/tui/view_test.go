package tui

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/Agrim-P777/qubex/internal/query"
	"github.com/Agrim-P777/qubex/internal/tree"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
)

func TestPaneDividerShowsTabFocusSwitch(t *testing.T) {
	m := testModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, " Tab ") || !strings.Contains(view, "  ⇄  ") {
		t.Fatalf("missing Tab focus hint:\n%s", view)
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !m.focusOutput {
		t.Fatal("Tab did not move focus to output")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.focusOutput {
		t.Fatal("Tab did not move focus back to source")
	}
}

func TestTreeSyntaxColors(t *testing.T) {
	styles := []struct {
		name  string
		value any
	}{
		{"object", map[string]any{}},
		{"array", []any{}},
		{"string", "value"},
		{"boolean", true},
		{"integer", 42},
		{"decimal", 4.25},
		{"null", nil},
	}
	seen := map[string]string{}
	for _, tc := range styles {
		r, g, b, _ := treeValueStyle(tc.value).GetForeground().RGBA()
		id := fmt.Sprintf("%d,%d,%d", r, g, b)
		if old, ok := seen[id]; ok {
			t.Fatalf("%s and %s share a color", old, tc.name)
		}
		seen[id] = tc.name
	}

	jsonText := `{"key":"a\"b","active":true,"count":12,"rate":1.25,"items":[null,{"ok":false}]}`
	if got := ansi.Strip(colorJSON(jsonText)); got != jsonText {
		t.Fatalf("compact JSON changed: %q", got)
	}

	treeView := tree.New([]any{map[string]any{"active": true, "count": 12, "items": []any{nil}, "name": "hello", "rate": 1.25}}, false)
	m := &Model{pretty: true}
	view := ansi.Strip(m.treeView(treeView, "SOURCE", 80, 10, true))
	for _, want := range []string{`"active": true`, `"count": 12`, `"items": [1 items]`, `"name": "hello"`, `"rate": 1.25`} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q in tree:\n%s", want, view)
		}
	}
}

func TestTreeSelectionHighlightsElement(t *testing.T) {
	treeView := tree.New([]any{map[string]any{"replicas": 3}}, false)
	treeView.Jump(tree.Ref{Path: query.Path{query.Field("replicas")}})
	m := &Model{pretty: true}
	view := m.treeView(treeView, "SOURCE", 40, 4, true)
	want := "  " + selected.Render(`  "replicas": 3`)
	if !strings.Contains(view, want) {
		t.Fatalf("selection did not cover the element:\n%q\nwant %q", view, want)
	}
	if strings.Contains(view, selected.Render("  ")) {
		t.Fatal("selection highlighted only the side indentation")
	}
}
