package scan

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"diskscope/internal/categories"
	"diskscope/internal/model"
)

func fixtureRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", "fixture-")
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("absolute fixture path: %v", err)
	}
	return abs
}

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func findNode(root *model.Node, name string) *model.Node {
	if root.Name == name {
		return root
	}
	for _, ch := range root.Children {
		if n := findNode(ch, name); n != nil {
			return n
		}
	}
	return nil
}

func collectFiles(n *model.Node, out map[string]bool) {
	for _, f := range n.Files {
		out[f.Path] = true
	}
	for _, ch := range n.Children {
		collectFiles(ch, out)
	}
}

func TestScanAggregatesAndRetains(t *testing.T) {
	root := fixtureRoot(t)
	writeFile(t, filepath.Join(root, "a.bin"), 1500)
	writeFile(t, filepath.Join(root, "ignore.tmp"), 5000)
	writeFile(t, filepath.Join(root, "media", "movie.mp4"), 5000)
	writeFile(t, filepath.Join(root, "media", "pic.png"), 200)
	writeFile(t, filepath.Join(root, "code", "main.go"), 300)
	for i := 0; i < 5; i++ {
		writeFile(t, filepath.Join(root, "small", "s"+string(rune('a'+i))), 10)
	}
	writeFile(t, filepath.Join(root, "cache", "junk.bin"), 700)
	writeFile(t, filepath.Join(root, "hard-one.dat"), 1000)
	if err := os.Link(filepath.Join(root, "hard-one.dat"), filepath.Join(root, "media", "hard-two.dat")); err != nil {
		t.Fatalf("hard link: %v", err)
	}
	linkPath := filepath.Join(root, "movie.link")
	if err := os.Symlink(filepath.Join(root, "media", "movie.mp4"), linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	linkInfo, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat symlink: %v", err)
	}

	opts := DefaultOptions(root)
	opts.ApparentSize = true
	opts.Workers = 4
	opts.TopFiles = 10
	opts.TopPerDir = 2
	opts.MinFileSize = 1000
	opts.Excludes = []string{"*.tmp"}
	prog := NewProgress()
	res, err := Scan(context.Background(), opts, prog)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.Canceled {
		t.Fatal("scan unexpectedly canceled")
	}
	if !prog.Snapshot().Done {
		t.Error("progress was not marked done")
	}

	wantTotal := int64(1500 + 5000 + 200 + 300 + 5*10 + 700 + 1000 + int(linkInfo.Size()))
	if res.TotalSize != wantTotal {
		t.Errorf("TotalSize = %d, want %d", res.TotalSize, wantTotal)
	}
	if res.TotalFiles != 12 {
		t.Errorf("TotalFiles = %d, want 12", res.TotalFiles)
	}
	if res.TotalDirs != 4 {
		t.Errorf("TotalDirs = %d, want 4", res.TotalDirs)
	}
	if res.Root.Size != wantTotal || res.Root.Count != 12 {
		t.Errorf("root aggregates = %d bytes / %d files", res.Root.Size, res.Root.Count)
	}

	files := map[string]bool{}
	collectFiles(res.Root, files)
	if files[filepath.Join(root, "ignore.tmp")] {
		t.Error("excluded file was retained")
	}
	if files[filepath.Join(root, "hard-one.dat")] && files[filepath.Join(root, "media", "hard-two.dat")] {
		t.Error("both hard link names were counted")
	}

	wantCats := map[categories.Category]int64{
		categories.Video:      5000,
		categories.Images:     200,
		categories.SourceCode: 300,
		categories.Binaries:   1500,
		categories.Caches:     700,
		categories.Other:      5*10 + 1000 + linkInfo.Size(),
	}
	for cat, want := range wantCats {
		if got := res.Root.Categories[cat]; got != want {
			t.Errorf("category %s = %d, want %d", cat, got, want)
		}
	}
	countSum := int64(0)
	for _, n := range res.CategoryCount {
		countSum += n
	}
	if countSum != res.TotalFiles {
		t.Errorf("category file counts sum to %d, want %d", countSum, res.TotalFiles)
	}

	small := findNode(res.Root, "small")
	if small == nil {
		t.Fatal("small directory missing from tree")
	}
	if len(small.Files) != 2 {
		t.Errorf("small.Files = %d, want 2 retained", len(small.Files))
	}
	if small.SmallNum != 3 || small.SmallSize != 30 {
		t.Errorf("small rollup = %d files / %d bytes, want 3 / 30", small.SmallNum, small.SmallSize)
	}
	if small.Count != 5 || small.Size != 50 {
		t.Errorf("small totals = %d files / %d bytes, want 5 / 50", small.Count, small.Size)
	}

	media := findNode(res.Root, "media")
	if media == nil || media.Categories[categories.Video] != 5000 {
		t.Errorf("media category totals = %#v", media)
	}

	if len(res.GlobalTop) == 0 || res.GlobalTop[0].Size != 5000 || res.GlobalTop[0].Name != "movie.mp4" {
		t.Errorf("GlobalTop[0] = %#v, want movie.mp4 at 5000", res.GlobalTop[0])
	}
	if len(res.CategoryTop[categories.Video]) == 0 {
		t.Error("missing per-category top files for Video")
	}
	for i := 1; i < len(res.TopDirs); i++ {
		if res.TopDirs[i-1].Size < res.TopDirs[i].Size {
			t.Fatalf("TopDirs not sorted at %d", i)
		}
	}
	if len(res.TopDirs) == 0 || res.TopDirs[0].Name != "media" {
		t.Errorf("largest directory = %v, want media", res.TopDirs[0].Name)
	}
}

func TestScanCancellationReturnsPartialResult(t *testing.T) {
	root := fixtureRoot(t)
	writeFile(t, filepath.Join(root, "file.bin"), 1024)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts := DefaultOptions(root)
	opts.MinFileSize = 0
	res, err := Scan(ctx, opts, NewProgress())
	if err != nil {
		t.Fatalf("canceled scan returned error: %v", err)
	}
	if !res.Canceled {
		t.Error("result should be marked canceled")
	}
	if res.TotalSize != 0 {
		t.Errorf("canceled scan size = %d, want 0", res.TotalSize)
	}
	if res.Root == nil {
		t.Error("canceled scan should retain the root node")
	}
}

func TestScanPermissionErrorsAreNonFatal(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission tests are meaningless as root")
	}
	root := fixtureRoot(t)
	writeFile(t, filepath.Join(root, "ok.bin"), 100)
	forbidden := filepath.Join(root, "forbidden")
	if err := os.Mkdir(forbidden, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(forbidden, "secret.bin"), 200)
	if err := os.Chmod(forbidden, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(forbidden, 0o700) })

	opts := DefaultOptions(root)
	opts.ApparentSize = true
	res, err := Scan(context.Background(), opts, NewProgress())
	if err != nil {
		t.Fatalf("scan should not fail on unreadable dirs: %v", err)
	}
	if res.Errors == 0 {
		t.Error("expected at least one error")
	}
	if len(res.ErrorSamples) == 0 {
		t.Error("expected error samples")
	}
	if res.TotalSize != 100 {
		t.Errorf("TotalSize = %d, want 100", res.TotalSize)
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"1024", 1024, true},
		{"1K", 1024, true},
		{"1KiB", 1024, true},
		{"1.5M", 1572864, true},
		{"2GB", 2 << 30, true},
		{"0", 0, true},
		{"1.5", 1, true},
		{"1MiB", 1 << 20, true},
		{"", 0, false},
		{"abc", 0, false},
		{"-1M", 0, false},
		{"NaN", 0, false},
		{"Inf", 0, false},
		{"1e30", 0, false},
	}
	for _, tc := range tests {
		got, err := ParseSize(tc.in)
		if (err == nil) != tc.ok {
			t.Errorf("ParseSize(%q) error = %v, ok = %v", tc.in, err, tc.ok)
			continue
		}
		if tc.ok && got != tc.want {
			t.Errorf("ParseSize(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestMatcher(t *testing.T) {
	m := newMatcher([]string{"*.tmp", "/home/u/private"})
	if !m.byName("x.tmp") {
		t.Error("glob should match by name")
	}
	if !m.byPath("/var/tmp/x.tmp") {
		t.Error("glob should match by path")
	}
	if !m.byPath("/proc/1/status") {
		t.Error("default /proc exclusion missing")
	}
	if m.byPath("/home/u/public/proc") {
		t.Error("prefix exclusion should be path-segment aware")
	}
	if !m.byPath("/home/u/private/deep/file") {
		t.Error("absolute prefix exclusion missing")
	}
	if m.byPath("/home/u/privateer/file") {
		t.Error("absolute prefix should not match sibling names")
	}
}

func TestTopHeapTracked(t *testing.T) {
	h := newTopHeap(2)
	if _, dropped := h.offerTracked(model.File{Name: "a", Size: 10}); dropped {
		t.Fatal("first item should be retained")
	}
	if _, dropped := h.offerTracked(model.File{Name: "b", Size: 20}); dropped {
		t.Fatal("second item should be retained")
	}
	evicted, dropped := h.offerTracked(model.File{Name: "c", Size: 15})
	if !dropped || evicted.Name != "a" {
		t.Fatalf("evicted = %#v, dropped = %v; want a", evicted, dropped)
	}
	evicted, dropped = h.offerTracked(model.File{Name: "d", Size: 5})
	if !dropped || evicted.Name != "d" {
		t.Fatalf("small item should be rolled up, got %#v", evicted)
	}
	got := h.sorted()
	if len(got) != 2 || got[0].Name != "b" || got[1].Name != "c" {
		t.Fatalf("heap contents = %#v", got)
	}
}
