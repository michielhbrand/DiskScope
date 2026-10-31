package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"diskscope/internal/categories"
)

type browserCols struct {
	name  int
	bar   int
	size  int
	pct   int
	items int
	cat   int
	mod   int

	showBar   bool
	showPct   bool
	showItems bool
	showCat   bool
	showMod   bool
}

func browserColumns(width int) browserCols {
	c := browserCols{
		size:      10,
		pct:       6,
		items:     7,
		cat:       13,
		mod:       12,
		showBar:   width >= 60,
		showPct:   width >= 44,
		showItems: width >= 78,
		showCat:   width >= 96,
		showMod:   width >= 124,
	}
	if c.showBar {
		c.bar = width / 9
		if c.bar < 6 {
			c.bar = 6
		}
		if c.bar > 16 {
			c.bar = 16
		}
	}
	for {
		fixed := 2 + 1 + c.size
		if c.showBar {
			fixed += 1 + c.bar
		}
		if c.showPct {
			fixed += 1 + c.pct
		}
		if c.showItems {
			fixed += 1 + c.items
		}
		if c.showCat {
			fixed += 1 + c.cat
		}
		if c.showMod {
			fixed += 1 + c.mod
		}
		c.name = width - fixed
		if c.name >= 8 {
			break
		}
		switch {
		case c.showMod:
			c.showMod = false
		case c.showCat:
			c.showCat = false
		case c.showItems:
			c.showItems = false
		case c.showBar:
			c.showBar = false
		case c.showPct:
			c.showPct = false
		default:
			c.name = maxInt(3, width-fixed)
			return c
		}
	}
	return c
}

func (m *Model) renderBrowser(l layout) string {
	cur := m.browse.current()
	if cur == nil {
		return padBlock(m.theme.Muted().Render("No scan data available."), l.BodyH, l.Width, m.theme.ASCII)
	}

	lines := make([]string, 0, l.BodyH)
	lines = append(lines, m.browserTopLine(l))
	cols := browserColumns(l.LeftW)

	rows := l.BodyH - 2
	if rows < 1 {
		rows = 1
	}
	if l.BodyH >= 6 {
		lines = append(lines, m.browserHeader(cols))
	} else {
		rows = l.BodyH - 1
	}
	m.browse.offset = ensureVisible(m.browse.selected, m.browse.offset, rows, len(m.browse.items))
	lines = append(lines, m.itemRows(m.browse.items, m.browse.selected, m.browse.offset, rows, l.LeftW, cols, m.browseDenominator())...)

	left := strings.Join(clipLines(lines, l.BodyH, 0), "\n")
	if !l.ShowDetails {
		return left
	}
	details := m.renderDetails(l.DetailW, l.BodyH)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", details)
}

func (m *Model) browserTopLine(l layout) string {
	if m.filtering {
		m.filter.Width = maxInt(8, l.LeftW-6)
		return m.theme.Accent().Render("filter: ") + m.filter.View()
	}
	cur := m.browse.current()
	path := ""
	if cur != nil {
		path = cur.Path
	}
	left := m.theme.Bold().Render(TruncateLeft(path, l.LeftW, m.theme.ASCII))
	if len(m.browse.items) > 0 {
		right := m.theme.Faint().Render(CountNoun(int64(len(m.browse.items)), "entry", "entries"))
		gap := l.LeftW - ansi.StringWidth(left) - ansi.StringWidth(right)
		if gap > 2 {
			return left + strings.Repeat(" ", gap) + right
		}
	}
	return Truncate(left, l.LeftW, m.theme.ASCII)
}

func (m *Model) browserHeader(c browserCols) string {
	parts := []string{"  "}
	parts = append(parts, m.theme.Muted().Bold(true).Render(PadRight("NAME", c.name, m.theme.ASCII)))
	if c.showBar {
		parts = append(parts, " ", m.theme.Muted().Bold(true).Render(strings.Repeat(" ", c.bar)))
	}
	parts = append(parts, " ", m.theme.Muted().Bold(true).Render(PadLeft("SIZE", c.size, m.theme.ASCII)))
	if c.showPct {
		parts = append(parts, " ", m.theme.Muted().Bold(true).Render(PadLeft("%", c.pct, m.theme.ASCII)))
	}
	if c.showItems {
		parts = append(parts, " ", m.theme.Muted().Bold(true).Render(PadLeft("FILES", c.items, m.theme.ASCII)))
	}
	if c.showCat {
		parts = append(parts, " ", m.theme.Muted().Bold(true).Render(PadRight("CATEGORY", c.cat, m.theme.ASCII)))
	}
	if c.showMod {
		parts = append(parts, " ", m.theme.Muted().Bold(true).Render(PadRight("MODIFIED", c.mod, m.theme.ASCII)))
	}
	return Truncate(strings.Join(parts, ""), c.name+sumCols(c), m.theme.ASCII)
}

func sumCols(c browserCols) int {
	n := 2 + 1 + c.size
	if c.showBar {
		n += 1 + c.bar
	}
	if c.showPct {
		n += 1 + c.pct
	}
	if c.showItems {
		n += 1 + c.items
	}
	if c.showCat {
		n += 1 + c.cat
	}
	if c.showMod {
		n += 1 + c.mod
	}
	return n
}

func (m *Model) itemRows(items []item, selected, offset, rows, width int, cols browserCols, denom int64) []string {
	lines := make([]string, 0, rows)
	if len(items) == 0 {
		lines = append(lines, m.theme.Muted().Render("  No matching entries in this directory."))
		return lines
	}
	end := offset + rows
	if end > len(items) {
		end = len(items)
	}
	for i := offset; i < end; i++ {
		lines = append(lines, m.browserRow(items[i], i == selected, width, cols, denom))
	}
	return lines
}

func (m *Model) browserRow(it item, selected bool, width int, c browserCols, denom int64) string {
	glyph := m.theme.FileGlyph()
	nameStyle := m.theme.Text()
	switch {
	case it.Rollup:
		glyph = m.theme.FileGlyph()
		nameStyle = m.theme.Faint()
	case it.IsDir:
		glyph = m.theme.DirGlyph()
		nameStyle = m.theme.Bold()
	}

	name := glyph + " " + it.Name
	fraction := 0.0
	if denom > 0 {
		fraction = float64(it.Size) / float64(denom)
	}
	bar := Bar(fraction, c.bar, m.theme.ASCII)
	barStyle := m.theme.Category(it.Category)
	pct := fmt.Sprintf("%.1f%%", Percent(it.Size, denom))
	cat := it.Category.String()
	mod := FormatTime(it.ModTime)
	if it.Rollup {
		pct = "-"
		cat = "aggregated"
		mod = "-"
	}

	if selected {
		var b strings.Builder
		b.WriteString(m.theme.Selector())
		b.WriteString(" ")
		b.WriteString(PadRight(name, c.name, m.theme.ASCII))
		if c.showBar {
			b.WriteString(" ")
			b.WriteString(bar)
		}
		b.WriteString(" ")
		b.WriteString(PadLeft(HumanBytes(it.Size), c.size, m.theme.ASCII))
		if c.showPct {
			b.WriteString(" ")
			b.WriteString(PadLeft(pct, c.pct, m.theme.ASCII))
		}
		if c.showItems {
			b.WriteString(" ")
			b.WriteString(PadLeft(FormatCount(it.Count), c.items, m.theme.ASCII))
		}
		if c.showCat {
			b.WriteString(" ")
			b.WriteString(PadRight(cat, c.cat, m.theme.ASCII))
		}
		if c.showMod {
			b.WriteString(" ")
			b.WriteString(PadRight(mod, c.mod, m.theme.ASCII))
		}
		line := PadRight(b.String(), width, m.theme.ASCII)
		return m.theme.Selected().Render(line)
	}

	var b strings.Builder
	b.WriteString(m.theme.Accent().Render(" "))
	b.WriteString(" ")
	b.WriteString(nameStyle.Render(PadRight(name, c.name, m.theme.ASCII)))
	if c.showBar {
		b.WriteString(" ")
		b.WriteString(barStyle.Render(bar))
	}
	b.WriteString(" ")
	b.WriteString(m.theme.Text().Render(PadLeft(HumanBytes(it.Size), c.size, m.theme.ASCII)))
	if c.showPct {
		b.WriteString(" ")
		b.WriteString(m.theme.Muted().Render(PadLeft(pct, c.pct, m.theme.ASCII)))
	}
	if c.showItems {
		b.WriteString(" ")
		b.WriteString(m.theme.Muted().Render(PadLeft(FormatCount(it.Count), c.items, m.theme.ASCII)))
	}
	if c.showCat {
		b.WriteString(" ")
		b.WriteString(m.theme.Category(it.Category).Render(PadRight(cat, c.cat, m.theme.ASCII)))
	}
	if c.showMod {
		b.WriteString(" ")
		b.WriteString(m.theme.Faint().Render(PadRight(mod, c.mod, m.theme.ASCII)))
	}
	return Truncate(b.String(), width, m.theme.ASCII)
}

func (m *Model) browseDenominator() int64 {
	if m.mode == modeCategoryFiles {
		if row, ok := m.cats.selectedRow(); ok && row.Cat == m.cats.detailCat && row.Size > 0 {
			return row.Size
		}
	}
	if cur := m.browse.current(); cur != nil && cur.Size > 0 {
		return cur.Size
	}
	if m.result != nil {
		return m.result.TotalSize
	}
	return 0
}

func (m *Model) renderDetails(width, height int) string {
	innerW := width - 4
	if innerW < 4 {
		innerW = 4
	}
	lines := make([]string, 0, height)
	lines = append(lines, m.theme.Section().Render(Truncate("DETAILS", innerW, m.theme.ASCII)))
	lines = append(lines, "")

	var it item
	var ok bool
	if m.mode == modeCategoryFiles {
		it, ok = m.cats.selectedDetail()
	} else {
		it, ok = m.browse.selectedItem()
	}

	if ok {
		lines = append(lines, m.theme.Bold().Render(Truncate(it.Name, innerW, m.theme.ASCII)))
		lines = append(lines, m.theme.Faint().Render(TruncateLeft(it.Path, innerW, m.theme.ASCII)))
		lines = append(lines, "")
		lines = append(lines, detailField(m.theme, innerW, "Type", itemType(it)))
		lines = append(lines, detailField(m.theme, innerW, "Size", HumanBytes(it.Size)))
		scope := "directory"
		if m.mode == modeCategoryFiles {
			scope = "category"
		}
		lines = append(lines, detailField(m.theme, innerW, "Share", fmt.Sprintf("%.1f%% of %s", Percent(it.Size, m.browseDenominator()), scope)))
		lines = append(lines, detailFieldStyled(m.theme, innerW, "Category", it.Category.String(), m.theme.Category(it.Category)))
		if it.IsDir {
			lines = append(lines, detailField(m.theme, innerW, "Files", FormatCount(it.Count)))
			if it.Node != nil {
				lines = append(lines, detailField(m.theme, innerW, "Dirs", FormatCount(it.Node.DirCount)))
			}
		} else if it.Rollup {
			lines = append(lines, detailField(m.theme, innerW, "Files", FormatCount(it.Count)))
			lines = append(lines, m.theme.Faint().Render(Truncate("Below retention threshold", innerW, m.theme.ASCII)))
		}
		if !it.ModTime.IsZero() {
			lines = append(lines, detailField(m.theme, innerW, "Modified", FormatTime(it.ModTime)))
		}
		if !it.IsDir && !it.Rollup {
			lines = append(lines, detailField(m.theme, innerW, "Mode", FileModeShort(it.Mode)))
		}
		if it.IsDir && it.Node != nil && len(it.Node.Files) > 0 {
			lines = append(lines, "")
			lines = append(lines, m.theme.Muted().Render("Largest files"))
			limit := 5
			if height < 24 {
				limit = 3
			}
			for i, f := range it.Node.Files {
				if i >= limit {
					break
				}
				line := m.theme.Category(f.Category).Render(m.theme.Dot()) + " " +
					m.theme.Text().Render(Truncate(f.Name, innerW-2, m.theme.ASCII)) +
					" " + m.theme.Faint().Render(HumanBytes(f.Size))
				lines = append(lines, Truncate(line, innerW, m.theme.ASCII))
			}
		}
		lines = append(lines, "")
		lines = append(lines, m.theme.Faint().Render(Truncate("c copy path   o open location", innerW, m.theme.ASCII)))
	} else if cur := m.browse.current(); cur != nil && m.tab == tabBrowser {
		lines = append(lines, m.theme.Bold().Render(Truncate(cur.Name, innerW, m.theme.ASCII)))
		lines = append(lines, m.theme.Faint().Render(TruncateLeft(cur.Path, innerW, m.theme.ASCII)))
		lines = append(lines, "")
		lines = append(lines, detailField(m.theme, innerW, "Type", "Directory"))
		lines = append(lines, detailField(m.theme, innerW, "Size", HumanBytes(cur.Size)))
		lines = append(lines, detailField(m.theme, innerW, "Files", FormatCount(cur.Count)))
		lines = append(lines, detailField(m.theme, innerW, "Dirs", FormatCount(cur.DirCount)))
		lines = append(lines, detailField(m.theme, innerW, "Modified", FormatTime(cur.ModTime)))
	} else {
		lines = append(lines, m.theme.Muted().Render(Truncate("Nothing selected.", innerW, m.theme.ASCII)))
	}

	for len(lines) < height-2 {
		lines = append(lines, "")
	}
	if len(lines) > height-2 {
		lines = lines[:height-2]
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, innerW, "")
	}

	top, mid, bottom := "╭", "│", "╰"
	topRight, bottomRight := "╮", "╯"
	if m.theme.ASCII {
		top, mid, bottom, topRight, bottomRight = "+", "|", "+", "+", "+"
	}
	horiz := strings.Repeat("─", width-2)
	if m.theme.ASCII {
		horiz = strings.Repeat("-", width-2)
	}
	var out []string
	out = append(out, top+horiz+topRight)
	for _, line := range lines {
		pad := innerW - ansi.StringWidth(line)
		if pad < 0 {
			pad = 0
		}
		out = append(out, mid+" "+line+strings.Repeat(" ", pad)+" "+mid)
	}
	out = append(out, bottom+horiz+bottomRight)
	for len(out) < height {
		out = append(out, "")
	}
	if len(out) > height {
		out = out[:height]
	}
	return strings.Join(out, "\n")
}

func detailField(theme Theme, width int, label, value string) string {
	return detailFieldStyled(theme, width, label, value, theme.Text())
}

func detailFieldStyled(theme Theme, width int, label, value string, style lipgloss.Style) string {
	labelW := 10
	if width < 24 {
		labelW = 7
	}
	labelText := theme.Muted().Render(PadRight(label, labelW, theme.ASCII))
	avail := width - labelW
	if avail < 1 {
		avail = 1
	}
	return labelText + style.Render(Truncate(value, avail, theme.ASCII))
}

func itemType(it item) string {
	switch {
	case it.Rollup:
		return "Aggregated files"
	case it.IsDir:
		return "Directory"
	case it.Mode&os.ModeSymlink != 0:
		return "Symlink"
	case it.Mode&os.ModeNamedPipe != 0:
		return "FIFO"
	default:
		return "File"
	}
}

func (m *Model) renderCategoryFiles(l layout) string {
	row, ok := m.cats.selectedRow()
	title := "Category files"
	if ok {
		title = row.Cat.String() + " · largest files"
	}
	lines := make([]string, 0, l.BodyH)
	cat := categories.Other
	if ok {
		cat = row.Cat
	}
	left := m.theme.Category(cat).Bold(true).Render(Truncate(title, l.LeftW, m.theme.ASCII))
	if ok {
		right := m.theme.Faint().Render(HumanBytes(row.Size) + " · " + CountNoun(row.Files, "file", "files"))
		gap := l.LeftW - ansi.StringWidth(left) - ansi.StringWidth(right)
		if gap > 2 {
			left += strings.Repeat(" ", gap) + right
		}
	}
	lines = append(lines, Truncate(left, l.LeftW, m.theme.ASCII))

	cols := browserColumns(l.LeftW)
	rows := l.BodyH - 2
	if rows < 1 {
		rows = 1
	}
	if l.BodyH >= 6 {
		lines = append(lines, m.browserHeader(cols))
	} else {
		rows = l.BodyH - 1
	}
	m.cats.detailOffset = ensureVisible(m.cats.detailSel, m.cats.detailOffset, rows, len(m.cats.detailItems))
	lines = append(lines, m.itemRows(m.cats.detailItems, m.cats.detailSel, m.cats.detailOffset, rows, l.LeftW, cols, m.browseDenominator())...)
	if len(m.cats.detailItems) == 0 {
		lines = append(lines, m.theme.Muted().Render("  No retained files in this category (all files are below the size threshold)."))
	}

	leftBlock := strings.Join(clipLines(lines, l.BodyH, 0), "\n")
	if !l.ShowDetails {
		return leftBlock
	}
	details := m.renderDetails(l.DetailW, l.BodyH)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, " ", details)
}
