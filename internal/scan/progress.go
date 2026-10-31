package scan

import (
	"sync/atomic"
	"time"
)

// Progress holds live scan counters. It is safe for concurrent use.
type Progress struct {
	bytes   atomic.Int64
	files   atomic.Int64
	dirs    atomic.Int64
	errors  atomic.Int64
	current atomic.Value // string
	done    atomic.Bool
	start   time.Time
}

// ProgressSnapshot is a consistent-enough view for rendering.
type ProgressSnapshot struct {
	Bytes   int64
	Files   int64
	Dirs    int64
	Errors  int64
	Current string
	Elapsed time.Duration
	Done    bool
}

func NewProgress() *Progress {
	p := &Progress{start: time.Now()}
	p.current.Store("")
	return p
}

func (p *Progress) addBytes(n int64) { p.bytes.Add(n) }
func (p *Progress) addFile()         { p.files.Add(1) }
func (p *Progress) addDir()          { p.dirs.Add(1) }
func (p *Progress) addError()        { p.errors.Add(1) }

func (p *Progress) setCurrent(path string) {
	if v, _ := p.current.Load().(string); v != path {
		p.current.Store(path)
	}
}

func (p *Progress) finish() { p.done.Store(true) }

// Snapshot returns counters plus elapsed time. It may be called from the UI
// goroutine while scanning continues.
func (p *Progress) Snapshot() ProgressSnapshot {
	current, _ := p.current.Load().(string)
	return ProgressSnapshot{
		Bytes:   p.bytes.Load(),
		Files:   p.files.Load(),
		Dirs:    p.dirs.Load(),
		Errors:  p.errors.Load(),
		Current: current,
		Elapsed: time.Since(p.start),
		Done:    p.done.Load(),
	}
}
