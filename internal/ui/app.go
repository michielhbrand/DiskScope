package ui

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"diskscope/internal/clipboard"
	"diskscope/internal/scan"
)

type tab int

const (
	tabOverview tab = iota
	tabBrowser
	tabCategories
)

func (t tab) String() string {
	switch t {
	case tabBrowser:
		return "BROWSER"
	case tabCategories:
		return "CATEGORIES"
	default:
		return "OVERVIEW"
	}
}

type mode int

const (
	modeMain mode = iota
	modeHelp
	modeCategoryFiles
)

type statusKind int

const (
	statusInfo statusKind = iota
	statusSuccess
	statusWarning
	statusError
)

type scanDoneMsg struct {
	res *scan.Result
	err error
}

type statusMsg struct {
	text string
	kind statusKind
}

type copiedMsg struct {
	osc string
	err error
}

// Model is the root Bubbletea model.
type Model struct {
	opts  scan.Options
	theme Theme

	ctx      context.Context
	cancel   context.CancelFunc
	prog     *scan.Progress
	scanning bool
	result   *scan.Result
	fatalErr error

	width, height int
	tab           tab
	mode          mode
	helpReturn    mode

	browse         browserState
	cats           categoryState
	overviewOffset int

	filter    textinput.Model
	filtering bool

	spinner spinner.Model

	status     string
	statusKind statusKind

	overviewLineCount int
	helpOffset        int
	helpLineCount     int
	tick              int
}

// NewModel creates the TUI model. It does not start scanning until Init.
func NewModel(opts scan.Options, theme Theme) *Model {
	opts.Normalize()
	ti := textinput.New()
	ti.Placeholder = "filter by name or path"
	ti.Prompt = "/ "
	ti.CharLimit = 256
	ti.PromptStyle = theme.Accent()
	ti.TextStyle = theme.Text()
	ti.PlaceholderStyle = theme.Faint()
	ti.Cursor.Style = theme.Accent()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	if theme.ASCII {
		sp.Spinner = spinner.Line
	}
	sp.Style = theme.Accent()

	return &Model{
		opts:    opts,
		theme:   theme,
		tab:     tabOverview,
		filter:  ti,
		spinner: sp,
	}
}

// Init starts the initial scan.
func (m *Model) Init() tea.Cmd {
	m.beginScan()
	return tea.Batch(m.spinner.Tick, m.scanCmd())
}

func (m *Model) beginScan() {
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel
	m.prog = scan.NewProgress()
	m.scanning = true
	m.result = nil
	m.fatalErr = nil
	m.tab = tabOverview
	m.mode = modeMain
	m.filtering = false
	m.filter.Blur()
	m.status = ""
	m.overviewOffset = 0
}

func (m *Model) scanCmd() tea.Cmd {
	ctx, prog, opts := m.ctx, m.prog, m.opts
	return func() tea.Msg {
		res, err := scan.Scan(ctx, opts, prog)
		return scanDoneMsg{res: res, err: err}
	}
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.clampViews()
		return m, nil

	case tea.KeyMsg:
		return m, m.handleKey(msg)

	case spinner.TickMsg:
		if !m.scanning {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.tick++
		return m, cmd

	case scanDoneMsg:
		m.scanning = false
		if m.cancel != nil {
			m.cancel()
			m.cancel = nil
		}
		if msg.err != nil {
			m.fatalErr = msg.err
			m.status = msg.err.Error()
			m.statusKind = statusError
			return m, nil
		}
		m.result = msg.res
		m.browse = newBrowserState(msg.res.Root)
		m.cats = newCategoryState(msg.res)
		m.mode = modeMain
		m.tab = tabOverview
		m.overviewOffset = 0
		if msg.res.Canceled {
			m.status = "Scan canceled - showing partial results"
			m.statusKind = statusWarning
		} else {
			m.status = fmt.Sprintf("Scan complete in %s · %s · %s",
				msg.res.Elapsed.Round(10*time.Millisecond), HumanBytes(msg.res.TotalSize),
				CountNoun(msg.res.TotalFiles, "file", "files"))
			m.statusKind = statusSuccess
		}
		m.clampViews()
		return m, nil

	case statusMsg:
		m.status = msg.text
		m.statusKind = msg.kind
		return m, nil

	case copiedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusKind = statusError
			return m, nil
		}
		m.status = "Path copied to clipboard"
		m.statusKind = statusSuccess
		if msg.osc != "" {
			return m, tea.Println(msg.osc)
		}
		return m, nil
	}

	if m.filtering {
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.browse.filter = m.filter.Value()
		m.browse.rebuild()
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()

	if key == "ctrl+c" {
		if m.cancel != nil {
			m.cancel()
		}
		return tea.Quit
	}

	if m.scanning {
		switch key {
		case "q", "esc":
			if m.cancel != nil {
				m.cancel()
				m.cancel = nil
				m.status = "Canceling scan…"
				m.statusKind = statusWarning
			}
		}
		return nil
	}

	if m.fatalErr != nil {
		switch key {
		case "q", "esc":
			return tea.Quit
		case "r":
			return m.rescan()
		}
		return nil
	}

	if m.mode == modeHelp {
		l := computeLayout(m.width, m.height, false)
		page := l.BodyH - 2
		if page < 1 {
			page = 1
		}
		switch key {
		case "?", "esc", "q":
			m.mode = m.helpReturn
		case "j", "down":
			m.helpOffset = scrollOffset(m.helpOffset, 1, m.helpLineCount, l.BodyH)
		case "k", "up":
			m.helpOffset = scrollOffset(m.helpOffset, -1, m.helpLineCount, l.BodyH)
		case "pgdown", "ctrl+f":
			m.helpOffset = scrollOffset(m.helpOffset, page, m.helpLineCount, l.BodyH)
		case "pgup", "ctrl+b":
			m.helpOffset = scrollOffset(m.helpOffset, -page, m.helpLineCount, l.BodyH)
		case "g", "home":
			m.helpOffset = 0
		case "G", "end":
			m.helpOffset = scrollOffset(m.helpLineCount, m.helpLineCount, m.helpLineCount, l.BodyH)
		}
		return nil
	}

	if m.filtering {
		switch key {
		case "esc":
			m.filter.SetValue("")
			m.filter.Blur()
			m.filtering = false
			m.browse.filter = ""
			m.browse.rebuild()
			return nil
		case "enter":
			m.filter.Blur()
			m.filtering = false
			m.browse.filter = m.filter.Value()
			m.browse.rebuild()
			return nil
		default:
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			m.browse.filter = m.filter.Value()
			m.browse.rebuild()
			return cmd
		}
	}

	switch key {
	case "?":
		if m.mode != modeHelp {
			m.helpReturn = m.mode
			m.helpOffset = 0
			m.mode = modeHelp
		}
		return nil
	case "q":
		return tea.Quit
	case "r":
		return m.rescan()
	case "tab":
		m.switchTab((m.tab + 1) % 3)
		return nil
	case "shift+tab":
		m.switchTab((m.tab + 2) % 3)
		return nil
	case "1":
		m.switchTab(tabOverview)
		return nil
	case "2":
		m.switchTab(tabBrowser)
		return nil
	case "3":
		m.switchTab(tabCategories)
		return nil
	}

	if m.mode == modeCategoryFiles {
		return m.handleCategoryDetailKey(key)
	}

	switch m.tab {
	case tabOverview:
		return m.handleOverviewKey(key)
	case tabCategories:
		return m.handleCategoriesKey(key)
	default:
		return m.handleBrowserKey(key)
	}
}

func (m *Model) switchTab(t tab) {
	m.tab = t
	m.mode = modeMain
	m.filtering = false
	m.filter.Blur()
	m.clampViews()
}

func (m *Model) rescan() tea.Cmd {
	if m.cancel != nil {
		m.cancel()
	}
	m.beginScan()
	return tea.Batch(m.spinner.Tick, m.scanCmd())
}

func (m *Model) handleOverviewKey(key string) tea.Cmd {
	page := m.bodyHeight() - 2
	if page < 1 {
		page = 1
	}
	switch key {
	case "j", "down":
		m.overviewOffset = scrollOffset(m.overviewOffset, 1, m.overviewLineCount, m.bodyHeight())
	case "k", "up":
		m.overviewOffset = scrollOffset(m.overviewOffset, -1, m.overviewLineCount, m.bodyHeight())
	case "pgdown", "ctrl+f":
		m.overviewOffset = scrollOffset(m.overviewOffset, page, m.overviewLineCount, m.bodyHeight())
	case "pgup", "ctrl+b":
		m.overviewOffset = scrollOffset(m.overviewOffset, -page, m.overviewLineCount, m.bodyHeight())
	case "g", "home":
		m.overviewOffset = 0
	case "G", "end":
		m.overviewOffset = scrollOffset(m.overviewLineCount, m.overviewLineCount, m.overviewLineCount, m.bodyHeight())
	}
	return nil
}

func (m *Model) handleBrowserKey(key string) tea.Cmd {
	rows := m.browseRows()
	switch key {
	case "j", "down":
		m.browse.move(1)
	case "k", "up":
		m.browse.move(-1)
	case "pgdown", "ctrl+f":
		m.browse.move(rows)
	case "pgup", "ctrl+b":
		m.browse.move(-rows)
	case "g", "home":
		m.browse.selected = 0
	case "G", "end":
		m.browse.selected = len(m.browse.items) - 1
	case "enter", "l", "right":
		m.browse.enter()
	case "backspace", "h", "left":
		m.browse.up()
	case "s":
		m.browse.cycleSort()
	case "S":
		m.browse.reverseSort()
	case "/":
		m.filtering = true
		m.filter.Focus()
		m.filter.SetValue(m.browse.filter)
		return textinput.Blink
	case "esc":
		if m.browse.filter != "" {
			m.browse.filter = ""
			m.filter.SetValue("")
			m.browse.rebuild()
		}
	case "c":
		return m.copySelection()
	case "o":
		return m.openSelection()
	}
	m.browse.offset = ensureVisible(m.browse.selected, m.browse.offset, rows, len(m.browse.items))
	return nil
}

func (m *Model) handleCategoriesKey(key string) tea.Cmd {
	rows := m.bodyHeight() - 3
	if rows < 1 {
		rows = 1
	}
	switch key {
	case "j", "down":
		m.cats.move(1)
	case "k", "up":
		m.cats.move(-1)
	case "pgdown", "ctrl+f":
		m.cats.move(rows)
	case "pgup", "ctrl+b":
		m.cats.move(-rows)
	case "g", "home":
		m.cats.selected = 0
	case "G", "end":
		m.cats.selected = len(m.cats.rows) - 1
	case "enter", "l", "right":
		if m.cats.openDetail(m.result) {
			m.mode = modeCategoryFiles
			m.clampViews()
		}
	}
	m.cats.offset = ensureVisible(m.cats.selected, m.cats.offset, rows, len(m.cats.rows))
	return nil
}

func (m *Model) handleCategoryDetailKey(key string) tea.Cmd {
	rows := m.bodyHeight() - 3
	if rows < 1 {
		rows = 1
	}
	switch key {
	case "j", "down":
		m.cats.moveDetail(1)
	case "k", "up":
		m.cats.moveDetail(-1)
	case "pgdown", "ctrl+f":
		m.cats.moveDetail(rows)
	case "pgup", "ctrl+b":
		m.cats.moveDetail(-rows)
	case "g", "home":
		m.cats.detailSel = 0
	case "G", "end":
		m.cats.detailSel = len(m.cats.detailItems) - 1
	case "backspace", "h", "left", "esc":
		m.mode = modeMain
	case "c":
		return m.copySelection()
	case "o":
		return m.openSelection()
	}
	m.cats.detailOffset = ensureVisible(m.cats.detailSel, m.cats.detailOffset, rows, len(m.cats.detailItems))
	return nil
}

func (m *Model) copySelection() tea.Cmd {
	var path string
	switch {
	case m.mode == modeCategoryFiles:
		if it, ok := m.cats.selectedDetail(); ok {
			path = it.Path
		}
	case m.tab == tabBrowser:
		if it, ok := m.browse.selectedItem(); ok && !it.Rollup {
			path = it.Path
		} else if cur := m.browse.current(); cur != nil {
			path = cur.Path
		}
	}
	if path == "" {
		return func() tea.Msg {
			return statusMsg{text: "Nothing selected to copy", kind: statusWarning}
		}
	}
	return func() tea.Msg {
		osc, err := clipboard.Copy(path)
		return copiedMsg{osc: osc, err: err}
	}
}

func (m *Model) openSelection() tea.Cmd {
	path := ""
	switch {
	case m.mode == modeCategoryFiles:
		if it, ok := m.cats.selectedDetail(); ok {
			path = filepath.Dir(it.Path)
		}
	case m.tab == tabBrowser:
		if it, ok := m.browse.selectedItem(); ok && !it.Rollup {
			if it.IsDir {
				path = it.Path
			} else {
				path = filepath.Dir(it.Path)
			}
		} else if cur := m.browse.current(); cur != nil {
			path = cur.Path
		}
	}
	if path == "" {
		return func() tea.Msg {
			return statusMsg{text: "Nothing selected to open", kind: statusWarning}
		}
	}
	return openLocationCmd(path)
}

func openLocationCmd(path string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", path)
		case "windows":
			cmd = exec.Command("explorer", path)
		default:
			cmd = exec.Command("xdg-open", path)
		}
		if err := cmd.Start(); err != nil {
			return statusMsg{text: "Could not open location: " + err.Error(), kind: statusError}
		}
		if cmd.Process != nil {
			_ = cmd.Process.Release()
		}
		return statusMsg{text: "Opened " + path, kind: statusInfo}
	}
}

func (m *Model) clampViews() {
	l := computeLayout(m.width, m.height, m.wantDetails())
	rows := l.BodyH - 2
	if rows < 1 {
		rows = 1
	}
	m.browse.offset = ensureVisible(m.browse.selected, m.browse.offset, rows, len(m.browse.items))
	catRows := l.BodyH - 3
	if catRows < 1 {
		catRows = 1
	}
	m.cats.offset = ensureVisible(m.cats.selected, m.cats.offset, catRows, len(m.cats.rows))
	m.cats.detailOffset = ensureVisible(m.cats.detailSel, m.cats.detailOffset, catRows, len(m.cats.detailItems))
	m.overviewOffset = scrollOffset(m.overviewOffset, 0, m.overviewLineCount, l.BodyH)
}

func (m *Model) wantDetails() bool {
	return (m.mode == modeMain && m.tab == tabBrowser) || m.mode == modeCategoryFiles
}

func (m *Model) bodyHeight() int {
	return computeLayout(m.width, m.height, m.wantDetails()).BodyH
}

func (m *Model) browseRows() int {
	rows := m.bodyHeight() - 2
	if rows < 1 {
		rows = 1
	}
	return rows
}
