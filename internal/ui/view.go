package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// View implements tea.Model.
func (m *Model) View() string {
	out := m.viewRaw()
	if m.theme.ASCII {
		out = asciiReplace(out)
	}
	return out
}

func (m *Model) viewRaw() string {
	if m.width == 0 || m.height == 0 {
		return "diskscope\n\nPreparing scanner…"
	}
	if tooSmall(m.width, m.height) {
		return m.viewTooSmall()
	}
	if m.scanning {
		return m.viewProgress()
	}
	if m.fatalErr != nil {
		return m.viewFatal()
	}
	if m.mode == modeHelp {
		return m.viewHelp()
	}
	return m.viewMain()
}

func (m *Model) viewMain() string {
	l := computeLayout(m.width, m.height, m.wantDetails())

	var body string
	switch {
	case m.tab == tabOverview:
		lines := m.overviewLines(l)
		m.overviewLineCount = len(lines)
		body = strings.Join(clipLines(lines, l.BodyH, m.overviewOffset), "\n")
	case m.tab == tabBrowser:
		body = m.renderBrowser(l)
	case m.tab == tabCategories && m.mode == modeCategoryFiles:
		body = m.renderCategoryFiles(l)
	case m.tab == tabCategories:
		body = m.renderCategories(l)
	default:
		body = ""
	}

	parts := []string{m.renderHeader(l)}
	if l.ShowTabs {
		parts = append(parts, m.renderTabs(l))
	}
	parts = append(parts, padBlock(body, l.BodyH, l.Width, m.theme.ASCII))
	if l.ShowStatus {
		parts = append(parts, m.renderStatus(l))
	}
	if l.ShowFooter {
		parts = append(parts, m.renderFooter(l))
	}
	return strings.Join(parts, "\n")
}

func (m *Model) renderHeader(l layout) string {
	title := m.theme.Title().Render("diskscope")

	rootPath := m.opts.Root
	if m.result != nil {
		rootPath = m.result.RootPath
	}

	right := ""
	if m.result != nil && m.result.Disk.Ok && l.Width >= 86 {
		used := Percent(m.result.Disk.Used, m.result.Disk.Total)
		style := m.theme.Success()
		switch {
		case used >= 90:
			style = m.theme.Danger()
		case used >= 75:
			style = m.theme.Warning()
		}
		right = m.theme.Muted().Render(HumanBytes(m.result.Disk.Used)) +
			m.theme.Faint().Render(" / ") +
			m.theme.Muted().Render(HumanBytes(m.result.Disk.Total)) +
			m.theme.Faint().Render("  ") +
			style.Render(FormatPercent(used)+" used")
	} else if m.result != nil && l.Width >= 70 {
		mode := "allocated"
		if m.result.ApparentSize {
			mode = "apparent"
		}
		right = m.theme.Faint().Render(mode)
	}

	rightW := ansi.StringWidth(right)
	leftBudget := l.Width - rightW - 1
	left := title
	rootAvail := leftBudget - ansi.StringWidth(title) - 2
	if rootAvail >= 8 {
		left += "  " + m.theme.Muted().Render(TruncateLeft(rootPath, rootAvail, m.theme.ASCII))
	}

	gap := l.Width - ansi.StringWidth(left) - rightW
	if gap < 0 {
		left = Truncate(left, l.Width-rightW-1, m.theme.ASCII)
		gap = l.Width - ansi.StringWidth(left) - rightW
	}
	if gap < 0 {
		gap = 0
	}
	return Truncate(left+strings.Repeat(" ", gap)+right, l.Width, m.theme.ASCII)
}

func (m *Model) renderTabs(l layout) string {
	tabs := []string{"OVERVIEW", "BROWSER", "CATEGORIES"}
	rendered := make([]string, 0, len(tabs))
	for i, name := range tabs {
		if tab(i) == m.tab {
			rendered = append(rendered, m.theme.TabActive().Render(name))
		} else {
			rendered = append(rendered, m.theme.TabInactive().Render(name))
		}
	}
	left := strings.Join(rendered, "   ")
	if l.Width >= 100 && m.result != nil {
		mode := "allocated size"
		if m.result.ApparentSize {
			mode = "apparent size"
		}
		if m.tab == tabBrowser && m.browse.filter != "" {
			mode = "filter: " + m.browse.filter
		}
		right := m.theme.Faint().Render(mode)
		gap := l.Width - ansi.StringWidth(left) - ansi.StringWidth(right)
		if gap > 0 {
			left += strings.Repeat(" ", gap) + right
		}
	}
	return Truncate(left, l.Width, m.theme.ASCII)
}

func (m *Model) renderStatus(l layout) string {
	text := m.status
	if text == "" {
		if m.result != nil {
			text = fmt.Sprintf("%s in %s · %s · %s",
				HumanBytes(m.result.TotalSize),
				m.result.Elapsed.Round(10*time.Millisecond),
				CountNoun(m.result.TotalFiles, "file", "files"),
				CountNoun(m.result.TotalDirs, "dir", "dirs"))
			if m.result.ApparentSize {
				text = "apparent " + text
			}
		} else {
			text = "Ready"
		}
	}
	var style lipgloss.Style
	switch m.statusKind {
	case statusSuccess:
		style = m.theme.Success()
	case statusWarning:
		style = m.theme.Warning()
	case statusError:
		style = m.theme.Danger()
	default:
		style = m.theme.Muted()
	}
	left := style.Render(Truncate(text, l.Width, m.theme.ASCII))

	right := ""
	if m.result != nil && m.result.Errors > 0 {
		right = m.theme.Danger().Render(CountNoun(m.result.Errors, "error", "errors"))
	}
	if m.tab == tabBrowser && m.browse.filter != "" {
		f := "filter: " + m.browse.filter
		if right != "" {
			f += "  " + right
		}
		right = m.theme.Accent().Render(Truncate(f, l.Width/3, m.theme.ASCII))
	}
	if right == "" {
		return left
	}
	rw := ansi.StringWidth(right)
	if rw+10 > l.Width {
		return left
	}
	left = Truncate(left, l.Width-rw-1, m.theme.ASCII)
	gap := l.Width - ansi.StringWidth(left) - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m *Model) renderFooter(l layout) string {
	type hint struct{ key, desc string }
	var hints []hint
	switch {
	case m.tab == tabOverview:
		hints = []hint{{"j/k", "scroll"}, {"g/G", "top/end"}, {"tab", "view"}, {"1-3", "jump"}, {"r", "rescan"}, {"?", "help"}, {"q", "quit"}}
	case m.tab == tabBrowser:
		hints = []hint{{"j/k", "move"}, {"enter", "open"}, {"backspace", "up"}, {"s/S", "sort"}, {"/", "filter"}, {"c", "copy"}, {"tab", "view"}, {"?", "help"}, {"q", "quit"}}
	case m.mode == modeCategoryFiles:
		hints = []hint{{"j/k", "move"}, {"backspace", "back"}, {"c", "copy"}, {"o", "open"}, {"tab", "view"}, {"?", "help"}, {"q", "quit"}}
	default:
		hints = []hint{{"j/k", "move"}, {"enter", "largest files"}, {"tab", "view"}, {"r", "rescan"}, {"?", "help"}, {"q", "quit"}}
	}

	maxHints := len(hints)
	if l.Width < 110 {
		maxHints = 6
	}
	if l.Width < 80 {
		maxHints = 5
	}
	if l.Width < 60 {
		maxHints = 4
	}
	if maxHints > len(hints) {
		maxHints = len(hints)
	}
	hints = hints[:maxHints]

	var b strings.Builder
	for i, h := range hints {
		if i > 0 {
			b.WriteString(m.theme.Faint().Render("  ·  "))
		}
		b.WriteString(m.theme.Key().Render(h.key))
		b.WriteString(" ")
		b.WriteString(m.theme.Hint().Render(h.desc))
	}
	right := ""
	switch {
	case m.tab == tabOverview && m.overviewLineCount > l.BodyH:
		right = m.theme.Faint().Render(fmt.Sprintf("%d%%", scrollPercent(m.overviewOffset, m.overviewLineCount, l.BodyH)))
	case m.tab == tabBrowser:
		arrow := "↓"
		if !m.browse.sortDesc {
			arrow = "↑"
		}
		if m.theme.ASCII {
			arrow = "v"
			if !m.browse.sortDesc {
				arrow = "^"
			}
		}
		right = m.theme.Faint().Render("sort " + m.browse.sortKey.label() + " " + arrow)
	}
	line := b.String()
	if right != "" {
		gap := l.Width - ansi.StringWidth(line) - ansi.StringWidth(right)
		if gap > 1 {
			line += strings.Repeat(" ", gap) + right
		}
	}
	return Truncate(line, l.Width, m.theme.ASCII)
}

func (m *Model) viewTooSmall() string {
	msg := fmt.Sprintf("Terminal too small\n%d×%d · need at least 40×10\n\nResize the window to continue", m.width, m.height)
	block := lipgloss.NewStyle().
		Width(maxInt(1, m.width-2)).
		Height(maxInt(1, m.height-2)).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(m.theme.CategoryColor(13)).
		Render(msg)
	return block
}

func keyHints(theme Theme, pairs [][2]string) string {
	var b strings.Builder
	for i, p := range pairs {
		if i > 0 {
			b.WriteString(theme.Faint().Render("  ·  "))
		}
		b.WriteString(theme.Key().Render(p[0]))
		b.WriteString(" ")
		b.WriteString(theme.Hint().Render(p[1]))
	}
	return b.String()
}

// clipLines returns a window of lines.
func clipLines(lines []string, height, offset int) []string {
	if height < 1 {
		height = 1
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(lines) {
		offset = len(lines)
	}
	end := offset + height
	if end > len(lines) {
		end = len(lines)
	}
	out := append([]string(nil), lines[offset:end]...)
	for len(out) < height {
		out = append(out, "")
	}
	return out
}

// padBlock ensures a rendered block occupies exactly height lines and no line
// exceeds width terminal cells.
func padBlock(block string, height, width int, ascii bool) string {
	lines := strings.Split(block, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		lines[i] = Truncate(line, width, ascii)
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

var asciiReplacer = strings.NewReplacer(
	"·", "|", "↑", "^", "↓", "v", "…", "~", "█", "#", "░", "-",
	"●", "*", "▸", "+", "›", ">", "→", "->", "×", "x",
	"╭", "+", "╮", "+", "╰", "+", "╯", "+", "─", "-", "│", "|",
)

func asciiReplace(s string) string {
	return asciiReplacer.Replace(s)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
