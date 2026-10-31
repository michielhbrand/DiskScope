package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func (m *Model) viewProgress() string {
	l := computeLayout(m.width, m.height, false)
	header := m.renderHeader(l)
	bodyH := m.height - 2
	if bodyH < 1 {
		bodyH = 1
	}

	snap := m.prog.Snapshot()
	title := "Scanning filesystem"
	titleStyle := m.theme.Title()
	if m.cancel == nil {
		title = "Canceling scan..."
		titleStyle = m.theme.Warning().Bold(true)
	}
	var lines []string
	lines = append(lines, "")
	lines = append(lines, m.spinner.View()+" "+titleStyle.Render(title))
	lines = append(lines, "")
	lines = append(lines, m.theme.Faint().Render(TruncateLeft(m.opts.Root, l.Width, m.theme.ASCII)))
	lines = append(lines, "")
	lines = append(lines, m.pulseBar(l.Width))
	lines = append(lines, "")

	errorStyle := m.theme.Muted()
	if snap.Errors > 0 {
		errorStyle = m.theme.Danger()
	}
	stats := [][2]string{
		{"Files seen", FormatCount(snap.Files)},
		{"Directories", FormatCount(snap.Dirs)},
		{"Bytes indexed", HumanBytes(snap.Bytes)},
		{"Throughput", HumanRate(snap.Bytes, snap.Elapsed)},
		{"Elapsed", snap.Elapsed.Round(100 * time.Millisecond).String()},
		{"Unreadable", FormatCount(snap.Errors)},
	}
	if l.Width >= 76 {
		colW := l.Width / 2
		for i := 0; i < len(stats); i += 2 {
			left := progressStat(m.theme, stats[i][0], stats[i][1], colW-1, errorStyle)
			right := ""
			if i+1 < len(stats) {
				right = progressStat(m.theme, stats[i+1][0], stats[i+1][1], colW-1, errorStyle)
			}
			lines = append(lines, left+"  "+right)
		}
	} else {
		for _, s := range stats {
			lines = append(lines, progressStat(m.theme, s[0], s[1], l.Width, errorStyle))
		}
	}
	lines = append(lines, "")
	lines = append(lines, m.theme.Muted().Render("CURRENT PATH"))
	cur := snap.Current
	if cur == "" {
		cur = m.opts.Root
	}
	lines = append(lines, m.theme.Text().Render(TruncateLeft(cur, l.Width, m.theme.ASCII)))
	lines = append(lines, "")
	lines = append(lines, m.theme.Faint().Render("Files below the retention threshold are aggregated; hard links are counted once."))

	body := strings.Join(clipLines(lines, bodyH, 0), "\n")
	footer := keyHints(m.theme, [][2]string{{"esc/q", "cancel scan"}, {"ctrl+c", "quit"}})
	return header + "\n" + padBlock(body, bodyH, l.Width, m.theme.ASCII) + "\n" + Truncate(footer, l.Width, m.theme.ASCII)
}

func progressStat(theme Theme, label, value string, width int, valueStyle interface{ Render(...string) string }) string {
	labelW := 15
	if width < 34 {
		labelW = 12
	}
	l := theme.Muted().Render(PadRight(label, labelW, theme.ASCII))
	avail := width - labelW
	if avail < 1 {
		avail = 1
	}
	v := Truncate(value, avail, theme.ASCII)
	return l + valueStyle.Render(v)
}

func (m *Model) pulseBar(width int) string {
	if width < 12 {
		width = 12
	}
	barW := width - 2
	if barW > 72 {
		barW = 72
	}
	band := barW / 6
	if band < 3 {
		band = 3
	}
	pos := (m.tick / 2) % (barW + band)
	pos -= band
	full, empty := "█", "░"
	if m.theme.ASCII {
		full, empty = "#", "-"
	}
	var b strings.Builder
	for i := 0; i < barW; i++ {
		if i >= pos && i < pos+band {
			b.WriteString(m.theme.Accent().Render(full))
		} else {
			b.WriteString(m.theme.Faint().Render(empty))
		}
	}
	return b.String()
}

func (m *Model) viewFatal() string {
	l := computeLayout(m.width, m.height, false)
	bodyH := m.height - 2
	if bodyH < 1 {
		bodyH = 1
	}
	lines := []string{
		"",
		m.theme.Danger().Bold(true).Render("Scan failed"),
		"",
	}
	for _, line := range wrapText(m.fatalErr.Error(), l.Width) {
		lines = append(lines, m.theme.Text().Render(Truncate(line, l.Width, m.theme.ASCII)))
	}
	lines = append(lines, "")
	lines = append(lines, m.theme.Muted().Render("Check that the path exists, is a directory, and is readable."))
	lines = append(lines, m.theme.Muted().Render("Press r to retry or q to quit."))

	body := strings.Join(clipLines(lines, bodyH, 0), "\n")
	footer := keyHints(m.theme, [][2]string{{"r", "retry"}, {"q", "quit"}})
	return m.renderHeader(l) + "\n" + padBlock(body, bodyH, l.Width, m.theme.ASCII) + "\n" + Truncate(footer, l.Width, m.theme.ASCII)
}

func wrapText(text string, width int) []string {
	if width < 8 {
		width = 8
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var out []string
	line := ""
	for _, w := range words {
		if line == "" {
			line = w
			continue
		}
		if ansi.StringWidth(line)+1+ansi.StringWidth(w) <= width {
			line += " " + w
		} else {
			out = append(out, line)
			line = w
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}
