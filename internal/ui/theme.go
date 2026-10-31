package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"diskscope/internal/categories"
)

// Theme holds the visual configuration for the UI. Colors are adaptive so the
// interface stays legible on both light and dark terminals.
type Theme struct {
	NoColor bool
	ASCII   bool
}

// NewTheme constructs a theme. When noColor is true the global lipgloss
// renderer is switched to an ASCII profile, stripping all escape sequences.
func NewTheme(noColor, ascii bool) Theme {
	if noColor {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
	return Theme{NoColor: noColor, ASCII: ascii}
}

func adaptive(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}

func (t Theme) Text() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(adaptive("#1F2328", "#C9D1D9"))
}

func (t Theme) Muted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(adaptive("#57606A", "#8B949E"))
}

func (t Theme) Faint() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(adaptive("#8C959F", "#6E7681"))
}

func (t Theme) Accent() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(adaptive("#0969DA", "#58A6FF"))
}

func (t Theme) Success() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(adaptive("#1A7F37", "#3FB950"))
}

func (t Theme) Warning() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(adaptive("#9A6700", "#D29922"))
}

func (t Theme) Danger() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(adaptive("#CF222E", "#F85149"))
}

func (t Theme) Bold() lipgloss.Style {
	return t.Text().Bold(true)
}

func (t Theme) Title() lipgloss.Style {
	return t.Accent().Bold(true)
}

func (t Theme) Section() lipgloss.Style {
	return t.Bold().Underline(true)
}

func (t Theme) Selected() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(adaptive("#0B1F33", "#E6EDF3")).
		Background(adaptive("#DDEBFA", "#172B45")).
		Bold(true)
}

func (t Theme) TabActive() lipgloss.Style {
	return t.Accent().Bold(true).Underline(true)
}

func (t Theme) TabInactive() lipgloss.Style {
	return t.Muted()
}

func (t Theme) Key() lipgloss.Style {
	return t.Accent().Bold(true)
}

func (t Theme) Hint() lipgloss.Style {
	return t.Faint()
}

func (t Theme) BorderColor() lipgloss.TerminalColor {
	return lipgloss.AdaptiveColor{Light: "#D0D7DE", Dark: "#30363D"}
}

// CategoryColor returns the category's hue.
func (t Theme) CategoryColor(c categories.Category) lipgloss.TerminalColor {
	if !c.Valid() {
		c = categories.Other
	}
	pair := categoryColors[int(c)]
	return adaptive(pair[0], pair[1])
}

func (t Theme) Category(c categories.Category) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.CategoryColor(c))
}

func (t Theme) CategoryBold(c categories.Category) lipgloss.Style {
	return t.Category(c).Bold(true)
}

var categoryColors = [categories.Count][2]string{
	categories.Video:           {"#6639BA", "#A371F7"},
	categories.Images:          {"#0969DA", "#58A6FF"},
	categories.Audio:           {"#0F7B83", "#39C5CF"},
	categories.Archives:        {"#7D4E00", "#D29922"},
	categories.Documents:       {"#1A7F37", "#7EE787"},
	categories.SourceCode:      {"#0550AE", "#79C0FF"},
	categories.Databases:       {"#CF222E", "#FF7B72"},
	categories.DataModels:      {"#3B4CCA", "#7C8CF8"},
	categories.Binaries:        {"#57606A", "#A5ADB8"},
	categories.Caches:          {"#BC4C00", "#F0883E"},
	categories.Logs:            {"#953800", "#DB6D28"},
	categories.Fonts:           {"#8250DF", "#D2A8FF"},
	categories.VirtualMachines: {"#BF3989", "#FF9BCE"},
	categories.Containers:      {"#1B7C83", "#56D4DD"},
	categories.System:          {"#6E7681", "#6E7681"},
	categories.Other:           {"#8C959F", "#8B949E"},
}

func (t Theme) DirGlyph() string {
	if t.ASCII {
		return "+"
	}
	return "▸"
}

func (t Theme) FileGlyph() string {
	if t.ASCII {
		return "."
	}
	return "·"
}

func (t Theme) Selector() string {
	if t.ASCII {
		return ">"
	}
	return "›"
}

func (t Theme) Dot() string {
	if t.ASCII {
		return "*"
	}
	return "●"
}

func (t Theme) Arrow() string {
	if t.ASCII {
		return "->"
	}
	return "→"
}
