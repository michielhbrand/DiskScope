package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"diskscope/internal/categories"
	"diskscope/internal/scan"
)

func (m *Model) overviewLines(l layout) []string {
	res := m.result
	if res == nil {
		return []string{m.theme.Muted().Render("No scan data available.")}
	}
	var lines []string

	if res.Disk.Ok && res.Disk.Total > 0 {
		lines = append(lines, m.theme.Section().Render("Drive capacity"))
		usedPct := Percent(res.Disk.Used, res.Disk.Total)
		barW := l.Width - 10
		if barW > 56 {
			barW = 56
		}
		if barW < 10 {
			barW = 10
		}
		bar := Bar(usedPct/100, barW, m.theme.ASCII)
		barStyle := m.theme.Success()
		switch {
		case usedPct >= 90:
			barStyle = m.theme.Danger()
		case usedPct >= 75:
			barStyle = m.theme.Warning()
		}
		lines = append(lines, barStyle.Render(bar)+"  "+barStyle.Bold(true).Render(FormatPercent(usedPct)+" used"))
		lines = append(lines, m.theme.Muted().Render(fmt.Sprintf("%s used of %s · %s available",
			HumanBytes(res.Disk.Used), HumanBytes(res.Disk.Total), HumanBytes(res.Disk.Free))))
		lines = append(lines, "")
	}

	method := "allocated"
	if res.ApparentSize {
		method = "apparent"
	}
	summary := fmt.Sprintf("Scanned %s (%s) · %s · %s · %s",
		HumanBytes(res.TotalSize), method,
		CountNoun(res.TotalFiles, "file", "files"),
		CountNoun(res.TotalDirs, "directory", "directories"),
		res.Elapsed.Round(10*time.Millisecond))
	lines = append(lines, m.theme.Section().Render("Scan summary"))
	lines = append(lines, m.theme.Muted().Render(Truncate(summary, l.Width, m.theme.ASCII)))
	if res.Errors > 0 {
		lines = append(lines, m.theme.Danger().Render(fmt.Sprintf("%s unreadable paths were skipped", FormatCount(res.Errors))))
	}
	lines = append(lines, "")

	rows := m.cats.rows
	if len(rows) > 0 {
		lines = append(lines, m.theme.Section().Render("Storage by category"))
		lines = append(lines, m.stackedCategories(rows, res.TotalSize, l.Width))
		lines = append(lines, m.categoryLegend(rows, res.TotalSize, l.Width)...)
		lines = append(lines, "")
	}

	if len(res.GlobalTop) > 0 {
		lines = append(lines, m.theme.Section().Render("Largest files"))
		limit := minInt(10, len(res.GlobalTop))
		sizeW := 10
		for i := 0; i < limit; i++ {
			f := res.GlobalTop[i]
			path := relativePath(res.RootPath, f.Path)
			nameW := l.Width - sizeW - 6
			if nameW < 10 {
				nameW = 10
			}
			line := m.theme.Faint().Render(fmt.Sprintf("%2d", i+1)) + "  " +
				m.theme.Text().Render(TruncateLeft(path, nameW, m.theme.ASCII)) + "  " +
				m.theme.Muted().Render(PadLeft(HumanBytes(f.Size), sizeW, m.theme.ASCII))
			lines = append(lines, Truncate(line, l.Width, m.theme.ASCII))
		}
		lines = append(lines, "")
	}

	if len(res.TopDirs) > 0 {
		lines = append(lines, m.theme.Section().Render("Largest directories"))
		limit := minInt(10, len(res.TopDirs))
		sizeW := 10
		for i := 0; i < limit; i++ {
			d := res.TopDirs[i]
			path := relativePath(res.RootPath, d.Path)
			nameW := l.Width - sizeW - 6
			if nameW < 10 {
				nameW = 10
			}
			line := m.theme.Faint().Render(fmt.Sprintf("%2d", i+1)) + "  " +
				m.theme.Text().Render(TruncateLeft(path, nameW, m.theme.ASCII)) + "  " +
				m.theme.Muted().Render(PadLeft(HumanBytes(d.Size), sizeW, m.theme.ASCII))
			lines = append(lines, Truncate(line, l.Width, m.theme.ASCII))
		}
		lines = append(lines, "")
	}

	insights := m.insightLines(res, l.Width)
	if len(insights) > 0 {
		lines = append(lines, m.theme.Section().Render("Insights"))
		lines = append(lines, insights...)
	}
	return lines
}

func (m *Model) stackedCategories(rows []categoryRow, total int64, width int) string {
	if total <= 0 || width < 8 {
		return ""
	}
	usable := width
	if usable > 120 {
		usable = 120
	}
	type alloc struct {
		row  categoryRow
		cell int
		rem  float64
	}
	allocs := make([]alloc, 0, len(rows))
	used := 0
	for _, row := range rows {
		exact := float64(row.Size) / float64(total) * float64(usable)
		cell := int(exact)
		allocs = append(allocs, alloc{row: row, cell: cell, rem: exact - float64(cell)})
		used += cell
	}
	for used < usable {
		best := -1
		for i := range allocs {
			if allocs[i].cell == 0 && allocs[i].row.Size == 0 {
				continue
			}
			if best == -1 || allocs[i].rem > allocs[best].rem {
				best = i
			}
		}
		if best == -1 {
			break
		}
		allocs[best].cell++
		allocs[best].rem -= 1
		used++
	}
	full := "█"
	if m.theme.ASCII {
		full = "#"
	}
	var b strings.Builder
	for _, a := range allocs {
		if a.cell <= 0 {
			continue
		}
		b.WriteString(m.theme.Category(a.row.Cat).Render(strings.Repeat(full, a.cell)))
	}
	return Truncate(b.String(), usable, m.theme.ASCII)
}

func (m *Model) categoryLegend(rows []categoryRow, total int64, width int) []string {
	if len(rows) == 0 || total <= 0 {
		return nil
	}
	twoCols := width >= 92
	colW := width
	if twoCols {
		colW = (width - 3) / 2
	}
	left := make([]string, 0, (len(rows)+1)/2)
	right := make([]string, 0, len(rows)/2)
	for i, row := range rows {
		entry := m.legendEntry(row, colW, total)
		if twoCols && i%2 == 1 {
			right = append(right, entry)
		} else {
			left = append(left, entry)
		}
	}
	if !twoCols {
		return left
	}
	out := make([]string, 0, maxInt(len(left), len(right)))
	for i := 0; i < maxInt(len(left), len(right)); i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		gap := width - ansi.StringWidth(l) - ansi.StringWidth(r)
		if gap < 1 {
			gap = 1
		}
		out = append(out, l+strings.Repeat(" ", gap)+r)
	}
	return out
}

func (m *Model) legendEntry(row categoryRow, width int, total int64) string {
	if width < 20 {
		return ""
	}
	sizeW := 10
	pctW := 6
	showPct := width >= 36
	nameW := width - 2 - 1 - 1 - sizeW
	if showPct {
		nameW -= 1 + pctW
	}
	if nameW < 6 {
		showPct = false
		nameW = width - 2 - 1 - 1 - sizeW
	}
	if nameW < 6 {
		nameW = 6
	}
	swatch := m.theme.Category(row.Cat).Render("██")
	name := m.theme.Text().Render(PadRight(row.Cat.String(), nameW, m.theme.ASCII))
	size := m.theme.Muted().Render(PadLeft(HumanBytes(row.Size), sizeW, m.theme.ASCII))
	line := swatch + " " + name + " " + size
	if showPct {
		line += " " + m.theme.Faint().Render(PadLeft(fmt.Sprintf("%.1f%%", Percent(row.Size, total)), pctW, m.theme.ASCII))
	}
	return line
}

func (m *Model) insightLines(res *scan.Result, width int) []string {
	var lines []string
	add := func(label, value string) {
		line := m.theme.Accent().Render(m.theme.Dot()) + " " +
			m.theme.Muted().Render(label+": ") +
			m.theme.Text().Render(value)
		lines = append(lines, Truncate(line, width, m.theme.ASCII))
	}
	if len(res.GlobalTop) > 0 {
		f := res.GlobalTop[0]
		add("Largest file", fmt.Sprintf("%s (%s)", relativePath(res.RootPath, f.Path), HumanBytes(f.Size)))
	}
	if len(res.TopDirs) > 0 {
		d := res.TopDirs[0]
		add("Largest directory", fmt.Sprintf("%s (%s)", relativePath(res.RootPath, d.Path), HumanBytes(d.Size)))
	}
	if len(m.cats.rows) > 0 {
		top := m.cats.rows[0]
		add("Dominant category", fmt.Sprintf("%s (%s, %.1f%%)", top.Cat.String(), HumanBytes(top.Size), Percent(top.Size, res.TotalSize)))
	}
	if res.Root != nil {
		cacheSize := res.Root.Categories[categories.Caches]
		if cacheSize > 0 {
			add("Caches & temp", fmt.Sprintf("%s across %s", HumanBytes(cacheSize), CountNoun(res.CategoryCount[categories.Caches], "file", "files")))
		}
	}
	if res.Errors > 0 {
		lines = append(lines, Truncate(m.theme.Danger().Render(fmt.Sprintf("%s %s skipped because of permission or I/O errors",
			m.theme.Dot(), FormatCount(res.Errors))), width, m.theme.ASCII))
	}
	return lines
}

func relativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return path
	}
	return filepath.ToSlash(rel)
}
