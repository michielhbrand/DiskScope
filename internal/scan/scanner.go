package scan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"diskscope/internal/categories"
	"diskscope/internal/model"
)

// DiskInfo describes capacity of the filesystem that was scanned.
type DiskInfo struct {
	Total int64
	Used  int64
	Free  int64
	Ok    bool
	Err   string
}

// Result is the outcome of a scan. It remains useful after cancellation.
type Result struct {
	Root          *model.Node
	RootPath      string
	TotalSize     int64
	TotalFiles    int64
	TotalDirs     int64
	Errors        int64
	ErrorSamples  []string
	Elapsed       time.Duration
	ApparentSize  bool
	MinFileSize   int64
	Canceled      bool
	GlobalTop     []model.File
	CategoryTop   map[categories.Category][]model.File
	CategoryCount [categories.Count]int64
	TopDirs       []*model.Node
	Disk          DiskInfo
}

type job struct {
	node *model.Node
}

type scanner struct {
	opts   Options
	prog   *Progress
	agg    *aggregator
	match  *matcher
	rootID uint64
	hasID  bool
}

// Scan walks opts.Root and returns an aggregated usage tree. Scanning is
// cancellable; a partial result is returned when ctx is canceled.
func Scan(ctx context.Context, opts Options, prog *Progress) (*Result, error) {
	opts.Normalize()
	start := time.Now()

	rootAbs, err := filepath.Abs(opts.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	rootAbs = filepath.Clean(rootAbs)
	rootInfo, err := os.Stat(rootAbs)
	if err != nil {
		return nil, fmt.Errorf("open root %s: %w", rootAbs, err)
	}
	if !rootInfo.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", rootAbs)
	}

	if prog == nil {
		prog = NewProgress()
	}
	defer prog.finish()

	dev, _, devOK := platformID(rootInfo)
	s := &scanner{
		opts:  opts,
		prog:  prog,
		agg:   newAggregator(opts.TopFiles),
		match: newMatcher(opts.Excludes),
	}
	if opts.OneFileSystem && devOK {
		s.rootID, s.hasID = dev, true
	}

	rootName := filepath.Base(rootAbs)
	if rootAbs == string(filepath.Separator) {
		rootName = rootAbs
	}
	root := &model.Node{
		Name:    rootName,
		Path:    rootAbs,
		ModTime: rootInfo.ModTime(),
		Mode:    rootInfo.Mode(),
	}
	prog.addDir()
	prog.setCurrent(rootAbs)

	allNodes := []*model.Node{root}
	current := []*model.Node{root}
	for len(current) > 0 && ctx.Err() == nil {
		var mu sync.Mutex
		next := make([]*model.Node, 0, len(current)*2)
		parallelFor(len(current), opts.Workers, func(i int) {
			if ctx.Err() != nil {
				return
			}
			dirs := s.scanDir(ctx, current[i])
			if len(dirs) > 0 {
				mu.Lock()
				next = append(next, dirs...)
				mu.Unlock()
			}
		})
		allNodes = append(allNodes, next...)
		current = next
	}

	// BFS produced nodes in ascending depth order. Aggregate from the deepest
	// level upward so every child is complete before it is merged.
	reversed := make([]*model.Node, len(allNodes))
	for i, n := range allNodes {
		reversed[len(allNodes)-1-i] = n
	}
	model.Finalize(reversed)

	res := &Result{
		Root:          root,
		RootPath:      rootAbs,
		TotalSize:     root.Size,
		TotalFiles:    root.Count,
		TotalDirs:     root.DirCount,
		Errors:        root.Errors,
		ErrorSamples:  s.agg.errorSamples(),
		Elapsed:       time.Since(start),
		ApparentSize:  opts.ApparentSize,
		MinFileSize:   opts.MinFileSize,
		Canceled:      ctx.Err() != nil,
		GlobalTop:     s.agg.globalTop(),
		CategoryTop:   s.agg.categoryTop(),
		CategoryCount: s.agg.categoryCounts(),
		Disk:          DiskUsage(rootAbs),
	}
	res.TopDirs = largestDirs(allNodes, 50)
	return res, nil
}

func (s *scanner) scanDir(ctx context.Context, node *model.Node) []*model.Node {
	s.prog.setCurrent(node.Path)
	entries, err := os.ReadDir(node.Path)
	if err != nil {
		s.recordError(node, err)
		return nil
	}

	var dirs []*model.Node
	var retained []model.File
	var small *topHeap
	var rollupSize, rollupNum int64

	for _, entry := range entries {
		if ctx.Err() != nil {
			break
		}
		name := entry.Name()
		if s.match.byName(name) {
			continue
		}
		path := filepath.Join(node.Path, name)
		if s.match.byPath(path) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			s.recordError(node, err)
			continue
		}

		if info.IsDir() {
			if s.opts.OneFileSystem && s.hasID {
				if d, _, ok := platformID(info); ok && d != s.rootID {
					continue
				}
			}
			child := &model.Node{
				Name:    name,
				Path:    path,
				Parent:  node,
				Depth:   node.Depth + 1,
				ModTime: info.ModTime(),
				Mode:    info.Mode(),
			}
			node.Children = append(node.Children, child)
			dirs = append(dirs, child)
			s.prog.addDir()
			continue
		}

		mode := info.Mode()
		isLink := mode&os.ModeSymlink != 0
		if !mode.IsRegular() && !isLink {
			continue // sockets, devices, fifos: not meaningful disk usage
		}

		size := fileSize(info, s.opts.ApparentSize)
		s.prog.addFile()
		s.prog.addBytes(size)

		f := model.File{
			Name:     name,
			Path:     path,
			Size:     size,
			Category: categories.Classify(path),
			ModTime:  info.ModTime(),
			Mode:     mode,
		}
		if !s.agg.consider(f, info) {
			continue // hard link already accounted for
		}

		node.Size += size
		node.Count++
		node.Categories[f.Category] += size

		if size >= s.opts.MinFileSize {
			retained = append(retained, f)
			continue
		}
		if small == nil {
			small = newTopHeap(s.opts.TopPerDir)
		}
		if evicted, dropped := small.offerTracked(f); dropped {
			rollupSize += evicted.Size
			rollupNum++
		}
	}

	if small != nil {
		for _, f := range small.items {
			retained = append(retained, f)
		}
	}
	sort.Slice(retained, func(i, j int) bool {
		if retained[i].Size != retained[j].Size {
			return retained[i].Size > retained[j].Size
		}
		return retained[i].Name < retained[j].Name
	})
	node.Files = retained
	node.SmallSize = rollupSize
	node.SmallNum = rollupNum
	return dirs
}

func (s *scanner) recordError(node *model.Node, err error) {
	node.Errors++
	s.prog.addError()
	s.agg.addError(node.Path + ": " + err.Error())
}

func fileSize(info os.FileInfo, apparent bool) int64 {
	if !apparent {
		if allocated, ok := platformAllocated(info); ok {
			return allocated
		}
	}
	return info.Size()
}

func largestDirs(nodes []*model.Node, limit int) []*model.Node {
	out := make([]*model.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.Parent != nil && n.Size > 0 {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Size != out[j].Size {
			return out[i].Size > out[j].Size
		}
		return out[i].Path < out[j].Path
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// parallelFor runs fn(0..n-1) using a bounded worker pool.
func parallelFor(n, workers int, fn func(int)) {
	if n <= 0 {
		return
	}
	if workers < 1 {
		workers = 1
	}
	if workers > n {
		workers = n
	}
	var next atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1)) - 1
				if i >= n {
					return
				}
				fn(i)
			}
		}()
	}
	wg.Wait()
}

type inodeKey struct {
	dev uint64
	ino uint64
}

type aggregator struct {
	mu       sync.Mutex
	inodes   map[inodeKey]struct{}
	global   *topHeap
	perCat   [categories.Count]*topHeap
	catCount [categories.Count]int64
	errors   []string
}

func newAggregator(topN int) *aggregator {
	a := &aggregator{
		inodes: make(map[inodeKey]struct{}),
		global: newTopHeap(topN),
	}
	for i := range a.perCat {
		a.perCat[i] = newTopHeap(topN)
	}
	return a
}

// consider records a unique file in the global and per-category top lists.
// It returns false when the file is a hard link that was already counted.
func (a *aggregator) consider(f model.File, info os.FileInfo) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if info.Mode().IsRegular() && platformNlink(info) > 1 {
		if dev, ino, ok := platformID(info); ok {
			key := inodeKey{dev: dev, ino: ino}
			if _, seen := a.inodes[key]; seen {
				return false
			}
			a.inodes[key] = struct{}{}
		}
	}
	a.global.offer(f)
	a.perCat[f.Category].offer(f)
	a.catCount[f.Category]++
	return true
}

func (a *aggregator) addError(message string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.errors) < 20 {
		a.errors = append(a.errors, message)
	}
}

func (a *aggregator) errorSamples() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.errors...)
}

func (a *aggregator) globalTop() []model.File {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.global.sorted()
}

func (a *aggregator) categoryTop() map[categories.Category][]model.File {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make(map[categories.Category][]model.File)
	for i, h := range a.perCat {
		if h.Len() > 0 {
			out[categories.Category(i)] = h.sorted()
		}
	}
	return out
}

func (a *aggregator) categoryCounts() [categories.Count]int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.catCount
}

type matcher struct {
	prefixes []string
	globs    []string
}

func newMatcher(excludes []string) *matcher {
	m := &matcher{
		prefixes: []string{"/proc", "/sys", "/dev", "/run"},
	}
	for _, e := range excludes {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if strings.HasPrefix(e, "/") && !strings.ContainsAny(e, "*?[") {
			m.prefixes = append(m.prefixes, filepath.Clean(e))
			continue
		}
		m.globs = append(m.globs, e)
	}
	return m
}

func (m *matcher) byName(name string) bool {
	for _, g := range m.globs {
		if ok, _ := filepath.Match(g, name); ok {
			return true
		}
	}
	return false
}

func (m *matcher) byPath(path string) bool {
	p := filepath.ToSlash(path)
	for _, prefix := range m.prefixes {
		prefix = filepath.ToSlash(prefix)
		if p == prefix || strings.HasPrefix(p, strings.TrimSuffix(prefix, "/")+"/") {
			return true
		}
	}
	for _, g := range m.globs {
		if ok, _ := filepath.Match(filepath.ToSlash(g), p); ok {
			return true
		}
		if ok, _ := filepath.Match(g, filepath.Base(path)); ok {
			return true
		}
	}
	return false
}
