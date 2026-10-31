package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type categoryCols struct {
	name  int
	bar   int
	size  int
	pct   int
	files int

	showBar   bool
	showPct   bool
	showFiles bool
}

func categoryColumns(width int) categoryCols {
	c := categoryCols{
		size:      10,
		pct:       6,
		files:     8,
		showBar:   width >= 46,
		showPct:   width >= 40,
		showFiles: width >= 64,
	}
	if c.showBar {
		c.bar = width / 10
		if c.bar < 8 {
			c.bar = 8
		}
		if c.bar > 22 {
			c.bar = 22
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
		if c.showFiles {
			fixed += 1 + c.files
		}
		c.name = width - fixed
		if c.name >= 10 {
			return c
		}
		switch {
		case c.showFiles:
			c.showFiles = false
		case c.showBar:
			c.showBar = false
		case c.showPct:
			c.showPct = false
		default:
			c.name = maxInt(4, width-fixed)
			return c
		}
	}
}

func (m *Model) renderCategories(l layout) string {
	lines := make([]string, 0, l.BodyH)
	title := m.theme.Section().Render("Storage by category")
	total := m.theme.Faint().Render(fmt.Sprintf("%s total · %d groups", HumanBytes(m.result.TotalSize), len(m.cats.rows)))
	gap := l.Width - ansi.StringWidth(title) - ansi.StringWidth(total)
	if gap > 2 {
		lines = append(lines, title+strings.Repeat(" ", gap)+total)
	} else {
		lines = append(lines, Truncate(title, l.Width, m.theme.ASCII))
	}

	cols := categoryColumns(l.Width)
	rows := l.BodyH - 2
	if rows < 1 {
		rows = 1
	}
	if l.BodyH >= 6 {
		lines = append(lines, m.categoryHeader(cols))
	} else {
		rows = l.BodyH - 1
	}

	m.cats.offset = ensureVisible(m.cats.selected, m.cats.offset, rows, len(m.cats.rows))
	if len(m.cats.rows) == 0 {
		lines = append(lines, m.theme.Muted().Render("  No categories found."))
	}
	end := m.cats.offset + rows
	if end > len(m.cats.rows) {
		end = len(m.cats.rows)
	}
	for i := m.cats.offset; i < end; i++ {
		lines = append(lines, m.categoryRow(m.cats.rows[i], i == m.cats.selected, cols, m.result.TotalSize))
	}
	return strings.Join(clipLines(lines, l.BodyH, 0), "\n")
}

func (m *Model) categoryHeader(c categoryCols) string {
	parts := []string{"  "}
	parts = append(parts, m.theme.Muted().Bold(true).Render(PadRight("CATEGORY", c.name, m.theme.ASCII)))
	if c.showBar {
		parts = append(parts, " ", m.theme.Muted().Bold(true).Render(strings.Repeat(" ", c.bar)))
	}
	parts = append(parts, " ", m.theme.Muted().Bold(true).Render(PadLeft("SIZE", c.size, m.theme.ASCII)))
	if c.showPct {
		parts = append(parts, " ", m.theme.Muted().Bold(true).Render(PadLeft("%", c.pct, m.theme.ASCII)))
	}
	if c.showFiles {
		parts = append(parts, " ", m.theme.Muted().Bold(true).Render(PadLeft("FILES", c.files, m.theme.ASCII)))
	}
	return strings.Join(parts, "")
}

func (m *Model) categoryRow(row categoryRow, selected bool, c categoryCols, total int64) string {
	fraction := float64(row.Size) / float64(total)
	bar := Bar(fraction, c.bar, m.theme.ASCII)
	pct := fmt.Sprintf("%.1f%%", Percent(row.Size, total))
	name := m.theme.Dot() + " " + row.Cat.String()

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
		b.WriteString(PadLeft(HumanBytes(row.Size), c.size, m.theme.ASCII))
		if c.showPct {
			b.WriteString(" ")
			b.WriteString(PadLeft(pct, c.pct, m.theme.ASCII))
		}
		if c.showFiles {
			b.WriteString(" ")
			b.WriteString(PadLeft(FormatCount(row.Files), c.files, m.theme.ASCII))
		}
		line := PadRight(b.String(), c.name+categoryFixed(c), m.theme.ASCII)
		return m.theme.Selected().Render(line)
	}

	var b strings.Builder
	b.WriteString(m.theme.Accent().Render(" "))
	b.WriteString(" ")
	nameText := PadRight(name, c.name, m.theme.ASCII)
	dot := m.theme.Dot()
	rest := strings.TrimPrefix(nameText, dot)
	b.WriteString(m.theme.Category(row.Cat).Render(dot))
	b.WriteString(m.theme.Text().Render(rest))
	if c.showBar {
		b.WriteString(" ")
		b.WriteString(m.theme.Category(row.Cat).Render(bar))
	}
	b.WriteString(" ")
	b.WriteString(m.theme.Text().Render(PadLeft(HumanBytes(row.Size), c.size, m.theme.ASCII)))
	if c.showPct {
		b.WriteString(" ")
		b.WriteString(m.theme.Muted().Render(PadLeft(pct, c.pct, m.theme.ASCII)))
	}
	if c.showFiles {
		b.WriteString(" ")
		b.WriteString(m.theme.Muted().Render(PadLeft(FormatCount(row.Files), c.files, m.theme.ASCII)))
	}
	return Truncate(b.String(), c.name+categoryFixed(c), m.theme.ASCII)
}

func categoryFixed(c categoryCols) int {
	n := 2 + 1 + c.size
	if c.showBar {
		n += 1 + c.bar
	}
	if c.showPct {
		n += 1 + c.pct
	}
	if c.showFiles {
		n += 1 + c.files
	}
	return n
}
