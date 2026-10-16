package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"encoding/json"
	"fmt"
	engine "github.com/Agrim-P777/qubex/internal/context"
	"github.com/Agrim-P777/qubex/internal/tree"
	"github.com/charmbracelet/x/ansi"
	"math/big"
	"strings"
	"unicode"
)

var (
	accent     = lipgloss.NewStyle().Foreground(lipgloss.Color("#75d5cf"))
	muted      = lipgloss.NewStyle().Foreground(lipgloss.Color("#8794a8"))
	selected   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ecffff")).Background(lipgloss.Color("#234552"))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb1a9"))
	treeKey    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6840a5"))
	treeString = lipgloss.NewStyle().Foreground(lipgloss.Color("#8bdc9a"))
	treeBool   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e9a1dc"))
	treeInt    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f4c47c"))
	treeFloat  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f3ab82"))
	treeArray  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7fb9ef"))
	treeObject = lipgloss.NewStyle().Foreground(lipgloss.Color("#75d5cf"))
)

func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return '�'
		}
		return r
	}, s)
}
func line(s string, w int) string {
	if w < 1 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}
func (m *Model) View() tea.View {
	w, h := max(20, m.width), max(8, m.height)
	var out string
	header := accent.Bold(true).Render(" QUBEX ") + muted.Render(" / interactive query builder")
	if m.height < 12 {
		v := tea.NewView("QUBEX\nResize terminal to at least 12 rows\n" + line("JQ "+m.compiled.Text, w) + "\nq Quit")
		v.AltScreen = true
		return v
	}
	if m.loading {
		out = header + "\n\n Loading JSON… " + fmt.Sprintf("%.1f MiB", float64(m.progress.Load())/(1<<20)) + "\n\n Ctrl+C cancels"
	} else if m.doc == nil {
		out = header + "\n\n" + errorStyle.Render(clean(m.notice)) + "\n\nq Quit"
	} else {
		header += "  " + muted.Render(line(clean(m.doc.Source), max(10, w-45)))
		bodyH := max(3, h-9)
		var body string
		if m.form != nil {
			body = m.formView(w, bodyH)
		} else if m.menu != nil {
			body = m.menuView(w, bodyH)
		} else if m.pipeline {
			body = m.pipelineView(w, bodyH)
		} else {
			left := m.source
			title := "SOURCE"
			if m.view == "stage" {
				left = m.stage
				title = "STAGE"
			}
			if m.inspect >= 0 {
				title += fmt.Sprintf(" after step %d", m.inspect+1)
			}
			if m.panel == 2 && w >= 76 {
				lw := (w - 5) / 2
				rw := w - lw - 5
				a := m.treeView(left, title, lw, bodyH, !m.focusOutput)
				b := m.treeView(m.output, "OUTPUT · result sequence", rw, bodyH, m.focusOutput)
				body = lipgloss.JoinHorizontal(lipgloss.Top, a, paneDivider(bodyH), b)
			} else if m.focusOutput && m.panel == 2 {
				body = m.treeView(m.output, "OUTPUT · result sequence", w, bodyH, true)
			} else {
				body = m.treeView(left, title, w, bodyH, true)
			}
		}
		path := "."
		target := "unavailable"
		if m.active() != nil {
			path = m.cursor().Path.String()
			r := engine.Resolve(m.cursor(), m.currentStage())
			if r.Mapped {
				target = r.Relative.String()
			}
		}
		q := m.compiled.Text
		if q == "" {
			q = "(invalid draft — repair the pipeline)"
		}
		if m.form != nil && m.form.Preview != "" {
			q = m.form.Preview
		}
		runes := []rune(q)
		off := min(m.queryOffset, len(runes))
		q = string(runes[off:])
		if off > 0 {
			q = "…" + q
		}
		state := m.notice
		if m.panel == 1 {
			state = "OUTPUT collapsed · " + state
		}
		if len(m.diagnostics) > 0 {
			state = m.diagnostics[0].Error() + " · last successful output retained"
		}
		footer := "↑↓ Navigate  ? Actions  / Search  v Source/Stage  f Filter  s Save  c Copy  q Quit"
		if m.form != nil {
			footer = "Tab/↑↓ Fields  ←→ Options  Enter Apply  Esc Cancel"
		} else if m.pipeline {
			footer = "↑↓ Select  Enter Edit  x Delete  K/J Reorder  i Inspect stage  c Copy  Esc Close"
		} else if m.menu != nil {
			footer = "↑↓ Choose  / Search  Enter Select  ? Help  Esc Close"
		}
		out = header + "\n" + muted.Render(strings.Repeat("─", w)) + "\n" + body + "\n" + muted.Render(strings.Repeat("─", w)) + "\n" + line(accent.Render("PATH ")+clean(path)+muted.Render("  TARGET ")+clean(target), w) + "\n" + line(accent.Render("JQ   ")+q, w) + "\n" + line(muted.Render(clean(state)), w) + "\n" + line(footer, w)
	}
	lines := strings.Split(out, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for i := range lines {
		lines[i] = line(lines[i], w)
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.WindowTitle = "Qubex · qbx"
	return v
}

func paneDivider(h int) string {
	lines := make([]string, h)
	for i := range lines {
		lines[i] = muted.Render("  │  ")
	}
	if h >= 2 {
		middle := h / 2
		lines[middle-1] = accent.Bold(true).Render(" Tab ")
		lines[middle] = accent.Render("  ⇄  ")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) treeView(t *tree.Tree, title string, w, h int, active bool) string {
	style := muted
	if active {
		style = accent
	}
	lines := []string{style.Bold(true).Render(line(title, w))}
	if t == nil || len(t.Roots) == 0 {
		lines = append(lines, muted.Render("  No results"))
	} else {
		for _, r := range t.Rows(max(1, h-1)) {
			icon := " "
			if r.Container {
				icon = "▸"
				if r.Expanded {
					icon = "▾"
				}
			}
			indent := strings.Repeat("  ", min(r.Depth, 30))
			label := clean(r.Label)
			summary := clean(tree.Summary(r.Value, max(w*2, 80)))
			plain := indent + icon + " " + label + ": " + summary
			colored := indent + treeValueStyle(r.Value).Render(icon) + " " + treeKey.Render(label) + muted.Render(": ") + treeValueStyle(r.Value).Render(summary)
			if t.Sequence && !m.pretty && r.Depth == 0 {
				b, e := json.Marshal(r.Value)
				if e == nil {
					plain = string(b)
					colored = colorJSON(plain)
				}
			}
			isSelected := r.Ref.Key() == t.Cursor.Key() && active
			text := colored
			if isSelected {
				text = plain
			}
			if m.wrap {
				wrapped := strings.Split(ansi.Wrap(text, w, ""), "\n")
				for _, s := range wrapped {
					if len(lines) >= h {
						break
					}
					if isSelected {
						s = highlightTreeElement(s, indent)
					}
					lines = append(lines, s)
				}
			} else {
				text = line(text, w)
				if isSelected {
					text = highlightTreeElement(text, indent)
				}
				lines = append(lines, text)
			}
			if len(lines) >= h {
				break
			}
		}
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return lipgloss.NewStyle().Width(w).Render(strings.Join(lines, "\n"))
}

func highlightTreeElement(s, indent string) string {
	if strings.TrimSpace(s) == "" {
		return s
	}
	if strings.HasPrefix(s, indent) {
		return indent + selected.Render(strings.TrimPrefix(s, indent))
	}
	return selected.Render(s)
}

func treeValueStyle(v any) lipgloss.Style {
	switch x := v.(type) {
	case map[string]any:
		return treeObject
	case []any:
		return treeArray
	case string:
		return treeString
	case bool:
		return treeBool
	case nil:
		return muted
	case int, int64, *big.Int:
		return treeInt
	case float32, float64:
		return treeFloat
	case json.Number:
		if strings.ContainsAny(string(x), ".eE") {
			return treeFloat
		}
		return treeInt
	default:
		return muted
	}
}

// colorJSON highlights the compact result without changing its serialized text.
func colorJSON(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); {
		start := i
		switch s[i] {
		case '"':
			i++
			for i < len(s) {
				if s[i] == '\\' {
					i += min(2, len(s)-i)
				} else if s[i] == '"' {
					i++
					break
				} else {
					i++
				}
			}
			if i < len(s) && s[i] == ':' {
				out.WriteString(treeKey.Render(clean(s[start:i])))
			} else {
				out.WriteString(treeString.Render(clean(s[start:i])))
			}
		case '{', '}':
			i++
			out.WriteString(treeObject.Render(s[start:i]))
		case '[', ']':
			i++
			out.WriteString(treeArray.Render(s[start:i]))
		case 't', 'f':
			for i < len(s) && s[i] >= 'a' && s[i] <= 'z' {
				i++
			}
			out.WriteString(treeBool.Render(s[start:i]))
		case 'n':
			i += min(4, len(s)-i)
			out.WriteString(muted.Render(s[start:i]))
		default:
			if s[i] == '-' || s[i] >= '0' && s[i] <= '9' {
				for i < len(s) && (s[i] == '-' || s[i] == '+' || s[i] == '.' || s[i] == 'e' || s[i] == 'E' || s[i] >= '0' && s[i] <= '9') {
					i++
				}
				style := treeInt
				if strings.ContainsAny(s[start:i], ".eE") {
					style = treeFloat
				}
				out.WriteString(style.Render(s[start:i]))
			} else {
				i++
				out.WriteByte(s[start])
			}
		}
	}
	return out.String()
}
func (m *Model) menuView(w, h int) string {
	menu := m.menu
	items := m.filteredMenu()
	lines := []string{accent.Bold(true).Render(line(clean(menu.Title), w))}
	if menu.Searching {
		lines = append(lines, menu.Search.View())
	}
	start := max(0, menu.Cursor-max(1, h-5))
	for i := start; i < len(items) && len(lines) < h-2; i++ {
		s := "  " + clean(items[i].Title)
		if i == menu.Cursor {
			s = selected.Render("› " + clean(items[i].Title))
		}
		lines = append(lines, line(s, w))
	}
	if len(items) > 0 {
		lines = append(lines, muted.Render(line(clean(items[min(menu.Cursor, len(items)-1)].Detail), w)))
	}
	if menu.Kind == "restore" && len(lines) < h-2 {
		lines = append(lines, strings.Split(m.treeView(m.output, "SAVED QUERY PREVIEW", w, h-len(lines), false), "\n")...)
	}
	return pad(lines, h)
}
func (m *Model) formView(w, h int) string {
	f := m.form
	lines := []string{accent.Bold(true).Render(line(f.Title, w))}
	if len(f.Names) > 0 {
		names := f.visibleNames()
		lines = append(lines, f.FieldSearch.View())
		start := max(0, f.CheckCursor-max(1, h-6))
		for i := start; i < len(names) && len(lines) < h-3; i++ {
			name := names[i]
			box := "[ ]"
			if f.Checks[name] {
				box = "[✓]"
			}
			s := "  " + box + " " + clean(name)
			if i == f.CheckCursor {
				s = selected.Render(s)
			}
			lines = append(lines, line(s, w))
		}
	} else {
		start := max(0, f.Focus-max(1, h-6))
		for i := start; i < len(f.Fields) && len(lines) < h-3; i++ {
			v := f.Fields[i]
			label := v.Label
			if i == f.Focus {
				label = "› " + label
			} else {
				label = "  " + label
			}
			value := v.Input.View()
			if len(v.Choices) > 0 {
				value = "← " + v.Input.Value() + " →"
				if v.Label == "Boundary" {
					label := "Entire query"
					if v.Input.Value() == "@arrays" {
						label = "Each array value"
					}
					for i, op := range m.timeline.Current.Operations {
						if op.ID == v.Input.Value() {
							label = fmt.Sprintf("Before step %d: %s", i+1, op.Kind)
							break
						}
					}
					value = "← " + label + " →"
				}
			}
			lines = append(lines, line(fmt.Sprintf("%-22s %s", label, value), w))
		}
	}
	if f.Error != "" {
		lines = append(lines, errorStyle.Render(line(clean(f.Error), w)))
	} else if f.Preview != "" {
		lines = append(lines, muted.Render(line("Preview: "+f.Preview, w)))
	}
	lines = append(lines, muted.Render("Suggestions are sampled · Apply commits one undoable change"))
	return pad(lines, h)
}
func (m *Model) pipelineView(w, h int) string {
	lines := []string{accent.Bold(true).Render("Query pipeline · invalid edits remain repairable")}
	ops := m.timeline.Current.Operations
	start := max(0, m.pipelineCursor-max(1, h-4))
	for i := start; i < len(ops) && len(lines) < h-2; i++ {
		o := ops[i]
		s := fmt.Sprintf("%2d  %-12s %s", i+1, o.Kind, o.Path.String())
		for _, d := range m.diagnostics {
			if d.OperationID == o.ID {
				s += "  ! " + d.Message
			}
		}
		if i == m.pipelineCursor {
			s = selected.Render(s)
		}
		lines = append(lines, line(s, w))
	}
	if len(ops) == 0 {
		lines = append(lines, "Identity query · add an operation with ?")
	} else if fragment, ok := m.stepExpression(ops[min(m.pipelineCursor, len(ops)-1)].ID); ok {
		lines = append(lines, muted.Render(line("Step jq: "+fragment, w)))
	}
	return pad(lines, h)
}
func pad(lines []string, h int) string {
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines[:min(h, len(lines))], "\n")
}
