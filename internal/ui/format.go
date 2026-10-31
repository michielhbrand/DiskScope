package ui

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// HumanBytes formats a byte count using binary units.
func HumanBytes(n int64) string {
	if n < 0 {
		return "-" + HumanBytes(-n)
	}
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	value := float64(n)
	unit := "B"
	for _, u := range units {
		value /= 1024
		unit = u
		if value < 1024 {
			break
		}
	}
	if value >= 100 {
		return fmt.Sprintf("%.0f %s", value, unit)
	}
	return fmt.Sprintf("%.1f %s", value, unit)
}

// HumanRate formats a byte-per-second rate.
func HumanRate(bytes int64, elapsed time.Duration) string {
	seconds := elapsed.Seconds()
	if seconds <= 0 {
		return "0 B/s"
	}
	return HumanBytes(int64(float64(bytes)/seconds)) + "/s"
}

// FormatCount inserts thousands separators.
func FormatCount(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// Percent returns part/total as a percentage, clamped to [0, 100].
func Percent(part, total int64) float64 {
	if total <= 0 || part <= 0 {
		return 0
	}
	p := float64(part) / float64(total) * 100
	if p > 100 {
		return 100
	}
	return p
}

// CountNoun formats a count with a correctly pluralised noun.
func CountNoun(n int64, singular, plural string) string {
	if n == 1 {
		return FormatCount(n) + " " + singular
	}
	return FormatCount(n) + " " + plural
}

// FormatPercent renders a percentage without false precision.
func FormatPercent(p float64) string {
	if p <= 0 {
		return "0%"
	}
	if p < 0.1 {
		return "<0.1%"
	}
	if p < 10 {
		return fmt.Sprintf("%.1f%%", p)
	}
	return fmt.Sprintf("%.0f%%", p)
}

// Bar renders a fixed-width bar of filled and empty cells.
func Bar(fraction float64, width int, ascii bool) string {
	if width <= 0 {
		return ""
	}
	if math.IsNaN(fraction) || fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	filled := int(math.Round(float64(width) * fraction))
	if fraction > 0 && filled == 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	full, empty := "█", "░"
	if ascii {
		full, empty = "#", "-"
	}
	return strings.Repeat(full, filled) + strings.Repeat(empty, width-filled)
}

// Truncate shortens text to width cells, appending tail when needed.
func Truncate(s string, width int, ascii bool) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= width {
		return s
	}
	tail := "…"
	if ascii {
		tail = "~"
	}
	return ansi.Truncate(s, width, tail)
}

// TruncateLeft keeps the tail of a path and prefixes an ellipsis.
func TruncateLeft(s string, width int, ascii bool) string {
	if width <= 0 {
		return ""
	}
	w := ansi.StringWidth(s)
	if w <= width {
		return s
	}
	tail := "…"
	if ascii {
		tail = "~"
	}
	keep := width - ansi.StringWidth(tail)
	if keep <= 0 {
		return ansi.Truncate(tail, width, "")
	}
	return tail + ansi.TruncateLeft(s, w-keep, "")
}

// PadRight pads or truncates s to exactly width cells.
func PadRight(s string, width int, ascii bool) string {
	s = Truncate(s, width, ascii)
	if gap := width - ansi.StringWidth(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

// PadLeft right-aligns s within width cells.
func PadLeft(s string, width int, ascii bool) string {
	s = Truncate(s, width, ascii)
	if gap := width - ansi.StringWidth(s); gap > 0 {
		return strings.Repeat(" ", gap) + s
	}
	return s
}

// FormatTime renders a compact timestamp.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	now := time.Now()
	if t.Year() == now.Year() {
		return t.Format("Jan 02 15:04")
	}
	return t.Format("2006-01-02")
}

// FileModeShort renders Unix-style permission bits.
func FileModeShort(mode os.FileMode) string {
	p := mode.Perm()
	b := []byte("---------")
	for i := 0; i < 9; i++ {
		if p&(1<<uint(8-i)) != 0 {
			switch i % 3 {
			case 0:
				b[i] = 'r'
			case 1:
				b[i] = 'w'
			case 2:
				b[i] = 'x'
			}
		}
	}
	return string(b)
}
