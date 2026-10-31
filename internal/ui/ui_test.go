package ui

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"diskscope/internal/categories"
	"diskscope/internal/model"
	"diskscope/internal/scan"
)

func catSizes(values map[categories.Category]int64) [categories.Count]int64 {
	var out [categories.Count]int64
	for cat, size := range values {
		out[cat] = size
	}
	return out
}

func fixtureResult() *scan.Result {
	now := time.Now().Add(-2 * time.Hour)
	const gi = int64(1 << 30)

	root := &model.Node{
		Name:      "u",
		Path:      "/home/u",
		Size:      10 * gi,
		Count:     1200,
		DirCount:  14,
		Errors:    2,
		SmallSize: 200 << 20,
		SmallNum:  42,
		ModTime:   now,
		Mode:      os.ModeDir | 0o755,
	}
	root.Categories = catSizes(map[categories.Category]int64{
		categories.Video:      4 * gi,
		categories.SourceCode: 2 * gi,
		categories.Caches:     3 * gi,
		categories.Archives:   1 * gi,
	})
	root.Files = []model.File{{
		Name: "backup.tar", Path: "/home/u/backup.tar", Size: gi,
		Category: categories.Archives, ModTime: now, Mode: 0o644,
	}}

	videos := &model.Node{
		Name: "Videos", Path: "/home/u/Videos", Parent: root, Depth: 1,
		Size: 4 * gi, Count: 400, DirCount: 3, ModTime: now,
		Mode: os.ModeDir | 0o755,
	}
	videos.Categories = catSizes(map[categories.Category]int64{categories.Video: 4 * gi})
	videos.Files = []model.File{
		{Name: "huge.mp4", Path: "/home/u/Videos/huge.mp4", Size: 3 * gi, Category: categories.Video, ModTime: now, Mode: 0o644},
		{Name: "clip.mkv", Path: "/home/u/Videos/clip.mkv", Size: gi, Category: categories.Video, ModTime: now, Mode: 0o644},
	}

	projects := &model.Node{
		Name: "projects", Path: "/home/u/projects", Parent: root, Depth: 1,
		Size: 5 * gi, Count: 700, DirCount: 8, ModTime: now,
		Mode: os.ModeDir | 0o755,
	}
	projects.Categories = catSizes(map[categories.Category]int64{
		categories.SourceCode: 2 * gi,
		categories.Caches:     3 * gi,
	})
	projects.Files = []model.File{
		{Name: "app.go", Path: "/home/u/projects/app.go", Size: 2 * gi, Category: categories.SourceCode, ModTime: now, Mode: 0o644},
		{Name: "cache.db", Path: "/home/u/projects/cache.db", Size: 3 * gi, Category: categories.Caches, ModTime: now, Mode: 0o644},
	}

	root.Children = []*model.Node{videos, projects}

	global := []model.File{
		{Name: "huge.mp4", Path: "/home/u/Videos/huge.mp4", Size: 3 * gi, Category: categories.Video, ModTime: now, Mode: 0o644},
		{Name: "cache.db", Path: "/home/u/projects/cache.db", Size: 3 * gi, Category: categories.Caches, ModTime: now, Mode: 0o644},
		{Name: "app.go", Path: "/home/u/projects/app.go", Size: 2 * gi, Category: categories.SourceCode, ModTime: now, Mode: 0o644},
		{Name: "backup.tar", Path: "/home/u/backup.tar", Size: gi, Category: categories.Archives, ModTime: now, Mode: 0o644},
		{Name: "clip.mkv", Path: "/home/u/Videos/clip.mkv", Size: gi, Category: categories.Video, ModTime: now, Mode: 0o644},
	}

	var counts [categories.Count]int64
	counts[categories.Video] = 100
	counts[categories.SourceCode] = 300
	counts[categories.Caches] = 500
	counts[categories.Archives] = 1

	return &scan.Result{
		Root:         root,
		RootPath:     "/home/u",
		TotalSize:    10 * gi,
		TotalFiles:   1200,
		TotalDirs:    14,
		Errors:       2,
		ErrorSamples: []string{"/home/u/locked: permission denied"},
		Elapsed:      8 * time.Second,
		MinFileSize:  1 << 20,
		GlobalTop:    global,
		CategoryTop: map[categories.Category][]model.File{
			categories.Video:      {videos.Files[0], videos.Files[1]},
			categories.SourceCode: {projects.Files[0]},
			categories.Caches:     {projects.Files[1]},
		},
		CategoryCount: counts,
		TopDirs:       []*model.Node{projects, videos},
		Disk:          scan.DiskInfo{Total: 150 << 30, Used: 42 << 30, Free: 108 << 30, Ok: true},
	}
}

func testModel(t *testing.T) *Model {
	t.Helper()
	m := NewModel(scan.DefaultOptions("/home/u"), NewTheme(true, false))
	m.width, m.height = 120, 40
	m.result = fixtureResult()
	m.browse = newBrowserState(m.result.Root)
	m.cats = newCategoryState(m.result)
	m.mode = modeMain
	m.tab = tabOverview
	m.clampViews()
	return m
}

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func assertViewFits(t *testing.T, view string, width, height int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) != height {
		t.Errorf("view has %d lines, want %d", len(lines), height)
	}
	for i, line := range lines {
		if got := ansi.StringWidth(line); got > width {
			t.Errorf("line %d width %d exceeds %d: %q", i, got, width, line)
		}
	}
}

func TestViewFitsAllSizesAndViews(t *testing.T) {
	sizes := []struct{ w, h int }{
		{40, 10}, {55, 15}, {72, 22}, {80, 24}, {100, 30}, {120, 40}, {160, 50}, {200, 60},
	}
	for _, sz := range sizes {
		m := testModel(t)
		m.width, m.height = sz.w, sz.h
		m.clampViews()

		for _, tab := range []tab{tabOverview, tabBrowser, tabCategories} {
			m.tab = tab
			m.mode = modeMain
			m.clampViews()
			assertViewFits(t, m.View(), sz.w, sz.h)
		}

		m.tab = tabCategories
		if m.cats.openDetail(m.result) {
			m.mode = modeCategoryFiles
			m.clampViews()
			assertViewFits(t, m.View(), sz.w, sz.h)
		}
	}
}

func TestOverviewContent(t *testing.T) {
	m := testModel(t)
	view := m.View()
	for _, want := range []string{
		"diskscope", "OVERVIEW", "Drive capacity", "Storage by category",
		"Largest files", "Largest directories", "Insights", "huge.mp4",
		"Caches & temp", "28% used",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("overview missing %q", want)
		}
	}
}

func TestBrowserNavigationSortAndFilter(t *testing.T) {
	m := testModel(t)
	m.tab = tabBrowser
	m.mode = modeMain
	m.clampViews()

	target := -1
	for i, it := range m.browse.items {
		if it.Name == "Videos" {
			target = i
		}
	}
	if target < 0 {
		t.Fatal("Videos entry missing from browser")
	}
	m.browse.selected = target

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cur := m.browse.current(); cur == nil || cur.Name != "Videos" {
		t.Fatalf("current directory = %#v, want Videos", cur)
	}
	if view := m.View(); !strings.Contains(view, "huge.mp4") {
		t.Error("drilled view should show huge.mp4")
	}
	if m.mode != modeMain {
		t.Errorf("mode = %v, want main", m.mode)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if cur := m.browse.current(); cur == nil || cur.Name != "u" {
		t.Fatalf("after back current = %#v, want u", cur)
	}

	m.Update(keyRunes("s"))
	if m.browse.sortKey != sortName || m.browse.sortDesc {
		t.Errorf("sort cycle = %v desc=%v, want name ascending", m.browse.sortKey, m.browse.sortDesc)
	}
	m.Update(keyRunes("S"))
	if !m.browse.sortDesc {
		t.Error("S should reverse the sort")
	}

	m.Update(keyRunes("/"))
	if !m.filtering {
		t.Fatal("expected filter input mode")
	}
	for _, r := range "proj" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if m.browse.filter != "proj" {
		t.Errorf("filter = %q, want proj", m.browse.filter)
	}
	if len(m.browse.items) != 1 || m.browse.items[0].Name != "projects" {
		t.Errorf("filtered items = %#v", m.browse.items)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.filtering {
		t.Error("enter should leave filter input mode")
	}
	if m.browse.filter != "proj" {
		t.Errorf("filter should persist after enter, got %q", m.browse.filter)
	}
	view := m.View()
	if !strings.Contains(view, "filter: proj") {
		t.Error("status bar should show active filter")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.browse.filter != "" {
		t.Errorf("esc should clear filter, got %q", m.browse.filter)
	}
	if len(m.browse.items) < 3 {
		t.Errorf("expected all entries after clearing filter, got %d", len(m.browse.items))
	}
}

func TestCategoryDetail(t *testing.T) {
	m := testModel(t)
	m.tab = tabCategories
	m.mode = modeMain
	m.clampViews()
	_ = m.View()

	first := m.cats.rows[0].Cat
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeCategoryFiles {
		t.Fatalf("mode = %v, want category files", m.mode)
	}
	if m.cats.detailCat != first {
		t.Errorf("detail category = %s, want %s", m.cats.detailCat, first)
	}
	view := m.View()
	for _, want := range []string{"largest files", "huge.mp4", "DETAILS"} {
		if !strings.Contains(view, want) {
			t.Errorf("category detail missing %q", want)
		}
	}
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.mode != modeMain {
		t.Errorf("mode after back = %v, want main", m.mode)
	}
}

func TestHelpAndSmallTerminal(t *testing.T) {
	m := testModel(t)
	m.Update(keyRunes("?"))
	if m.mode != modeHelp {
		t.Fatal("? should open help")
	}
	view := m.View()
	for _, want := range []string{"diskscope help", "Navigation", "Scanner behavior", "Categories"} {
		if !strings.Contains(view, want) {
			t.Errorf("help missing %q", want)
		}
	}
	assertViewFits(t, view, 120, 40)
	m.Update(keyRunes("?"))
	if m.mode != modeMain {
		t.Error("? should close help")
	}

	small := testModel(t)
	small.width, small.height = 30, 8
	if view := small.View(); !strings.Contains(view, "Terminal too small") {
		t.Error("small terminal should show resize message")
	}
}

func TestProgressAndFatalViews(t *testing.T) {
	m := NewModel(scan.DefaultOptions("/home/u"), NewTheme(true, false))
	m.width, m.height = 100, 30
	m.scanning = true
	m.prog = scan.NewProgress()
	m.cancel = func() {}
	m.tick = 5
	view := m.View()
	for _, want := range []string{"Scanning filesystem", "Files seen", "CURRENT PATH", "cancel scan"} {
		if !strings.Contains(view, want) {
			t.Errorf("progress view missing %q", want)
		}
	}
	assertViewFits(t, view, 100, 30)

	fatal := NewModel(scan.DefaultOptions("/root"), NewTheme(true, false))
	fatal.width, fatal.height = 80, 24
	fatal.fatalErr = errors.New("permission denied opening /root")
	view = fatal.View()
	for _, want := range []string{"Scan failed", "permission denied", "retry"} {
		if !strings.Contains(view, want) {
			t.Errorf("fatal view missing %q", want)
		}
	}
	assertViewFits(t, view, 80, 24)
}

func TestASCIIModeIsPureASCII(t *testing.T) {
	m := testModel(t)
	m.theme = NewTheme(true, true)
	m.width, m.height = 120, 40
	m.clampViews()
	for _, tab := range []tab{tabOverview, tabBrowser, tabCategories} {
		m.tab = tab
		m.mode = modeMain
		m.clampViews()
		assertASCII(t, m.View())
	}
	if m.cats.openDetail(m.result) {
		m.mode = modeCategoryFiles
		assertASCII(t, m.View())
	}
	m.mode = modeHelp
	assertASCII(t, m.View())

	m.width, m.height = 30, 8
	assertASCII(t, m.View())

	scanning := NewModel(scan.DefaultOptions("/home/u"), NewTheme(true, true))
	scanning.width, scanning.height = 100, 30
	scanning.scanning = true
	scanning.prog = scan.NewProgress()
	assertASCII(t, scanning.View())
}

func assertASCII(t *testing.T, view string) {
	t.Helper()
	for _, r := range view {
		if r > 127 {
			t.Fatalf("non-ASCII rune %q in ASCII mode: %q", r, view)
		}
	}
}

func TestWindowResize(t *testing.T) {
	m := testModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.width != 80 || m.height != 24 {
		t.Fatalf("size = %dx%d", m.width, m.height)
	}
	assertViewFits(t, m.View(), 80, 24)
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	assertViewFits(t, m.View(), 160, 50)
}

func TestOverviewScrolling(t *testing.T) {
	m := testModel(t)
	m.width, m.height = 80, 24
	m.clampViews()
	_ = m.View()
	if m.overviewLineCount == 0 {
		t.Fatal("overview line count not populated")
	}
	m.Update(keyRunes("j"))
	if m.overviewOffset != 1 {
		t.Errorf("overview offset = %d, want 1", m.overviewOffset)
	}
	m.Update(keyRunes("G"))
	if m.overviewOffset == 0 {
		t.Error("G should scroll to the end")
	}
	m.Update(keyRunes("g"))
	if m.overviewOffset != 0 {
		t.Errorf("g should reset scroll, got %d", m.overviewOffset)
	}
}

func TestQuitDuringScanCancels(t *testing.T) {
	m := NewModel(scan.DefaultOptions("/home/u"), NewTheme(true, false))
	m.width, m.height = 80, 24
	m.scanning = true
	m.prog = scan.NewProgress()
	m.beginScan()
	m.scanning = true
	m.prog = scan.NewProgress()

	_, cmd := m.Update(keyRunes("q"))
	if cmd != nil {
		t.Error("q during scan should cancel, not quit")
	}
	if m.cancel != nil {
		t.Error("cancel function should be cleared after cancellation")
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("ctrl+c should quit")
	}
}

func TestListRowWidthsAreExact(t *testing.T) {
	m := testModel(t)
	for _, width := range []int{40, 46, 60, 78, 92, 100, 120, 160} {
		cc := categoryColumns(width)
		if h := m.categoryHeader(cc); ansi.StringWidth(h) != width {
			t.Errorf("category header width at %d = %d (%q)", width, ansi.StringWidth(h), h)
		}
		for _, row := range m.cats.rows {
			for _, selected := range []bool{false, true} {
				line := m.categoryRow(row, selected, cc, m.result.TotalSize)
				if !selected && !strings.HasPrefix(line, "  "+m.theme.Dot()) {
					t.Errorf("category row at %d has bad marker: %q", width, line)
				}
				if got := ansi.StringWidth(line); got != width {
					t.Errorf("category row width at %d selected=%v = %d (%q)", width, selected, got, line)
				}
			}
		}

		bc := browserColumns(width)
		if h := m.browserHeader(bc); ansi.StringWidth(h) != width {
			t.Errorf("browser header width at %d = %d (%q)", width, ansi.StringWidth(h), h)
		}
		for _, it := range m.browse.items {
			for _, selected := range []bool{false, true} {
				line := m.browserRow(it, selected, width, bc, m.result.TotalSize)
				if got := ansi.StringWidth(line); got != width {
					t.Errorf("browser row width at %d selected=%v = %d (%q)", width, selected, got, line)
				}
			}
		}
	}
}
