package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

type helpEntry struct {
	key  string
	desc string
}

func (m *Model) viewHelp() string {
	l := computeLayout(m.width, m.height, false)
	bodyH := m.height - 2
	if bodyH < 1 {
		bodyH = 1
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, m.theme.Title().Render("diskscope help"))
	lines = append(lines, m.theme.Faint().Render("Read-only disk usage explorer for large filesystems."))
	lines = append(lines, "")

	addGroup := func(title string, entries []helpEntry) {
		lines = append(lines, m.theme.Section().Render(title))
		keyW := 16
		if l.Width < 54 {
			keyW = 12
		}
		for _, e := range entries {
			key := m.theme.Key().Render(PadRight(e.key, keyW, m.theme.ASCII))
			descW := l.Width - keyW
			if descW < 6 {
				descW = 6
			}
			lines = append(lines, "  "+key+m.theme.Hint().Render(Truncate(e.desc, descW, m.theme.ASCII)))
		}
		lines = append(lines, "")
	}

	addGroup("Navigation", []helpEntry{
		{"j/k, ↑/↓", "move selection or scroll"},
		{"pgup/pgdn", "page up / page down"},
		{"g/G", "jump to first / last"},
		{"enter/l", "open directory, category, or file list"},
		{"backspace/h", "go to parent or previous view"},
		{"tab/shift+tab", "switch between views"},
		{"1/2/3", "jump to overview, browser, categories"},
	})
	addGroup("Sorting and filtering", []helpEntry{
		{"s", "cycle sort: size, name, files, modified"},
		{"S", "reverse sort direction"},
		{"/", "filter entries in the current directory"},
		{"esc", "clear filter or leave current view"},
	})
	addGroup("Actions", []helpEntry{
		{"c", "copy selected path to the clipboard"},
		{"o", "open selected directory or file location"},
		{"r", "rescan with the same options"},
		{"?", "toggle this help"},
		{"q", "quit (cancels an active scan)"},
	})
	addGroup("Scanner behavior", []helpEntry{
		{"allocated", "sizes use st_blocks; --apparent-size uses st_size"},
		{"symlinks", "never followed"},
		{"hard links", "counted once per filesystem"},
		{"retention", "all large files plus the largest N per directory; the rest are rolled up"},
		{"mounts", "one filesystem by default; disable with --one-file-system=false"},
		{"errors", "unreadable paths are counted and skipped, never fatal"},
	})
	addGroup("Categories", []helpEntry{
		{"groups", "video, images, audio, archives, documents, source code"},
		{"", "databases, data & models, binaries, caches/temp, logs, fonts"},
		{"", "VMs/images, containers, system, other"},
		{"colors", "each group has a distinct muted hue and a text label"},
	})

	m.helpLineCount = len(lines)
	body := strings.Join(clipLines(lines, bodyH, m.helpOffset), "\n")
	footer := keyHints(m.theme, [][2]string{{"j/k", "scroll"}, {"?", "close"}, {"q", "close"}})
	if m.helpLineCount > bodyH {
		right := m.theme.Faint().Render(fmt.Sprintf("%d%%", scrollPercent(m.helpOffset, m.helpLineCount, bodyH)))
		gap := l.Width - ansi.StringWidth(footer) - ansi.StringWidth(right)
		if gap > 1 {
			footer += strings.Repeat(" ", gap) + right
		}
	}
	if ansi.StringWidth(footer) > l.Width {
		footer = Truncate(footer, l.Width, m.theme.ASCII)
	}
	return m.renderHeader(l) + "\n" + padBlock(body, bodyH, l.Width, m.theme.ASCII) + "\n" + footer
}
