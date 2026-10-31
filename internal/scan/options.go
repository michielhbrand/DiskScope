package scan

import (
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
)

// Options controls a scan.
type Options struct {
	Root          string
	ApparentSize  bool
	OneFileSystem bool
	Excludes      []string
	TopFiles      int
	TopPerDir     int
	MinFileSize   int64
	Workers       int
}

// DefaultOptions returns tuned defaults for an interactive disk-usage session.
func DefaultOptions(root string) Options {
	return Options{
		Root:          root,
		OneFileSystem: true,
		TopFiles:      500,
		TopPerDir:     20,
		MinFileSize:   1 << 20,
		Workers:       runtime.NumCPU(),
	}
}

// Normalize fills zero values with sane defaults and clamps invalid values.
func (o *Options) Normalize() {
	if o.Workers < 1 {
		o.Workers = runtime.NumCPU()
	}
	if o.Workers > 256 {
		o.Workers = 256
	}
	if o.TopFiles < 1 {
		o.TopFiles = 500
	}
	if o.TopPerDir < 1 {
		o.TopPerDir = 20
	}
	if o.MinFileSize < 0 {
		o.MinFileSize = 0
	}
}

// ParseSize parses a byte count with an optional suffix. Supported suffixes are
// B, K/KB/KiB, M/MB/MiB, G/GB/GiB, T/TB/TiB (binary multiples).
func ParseSize(s string) (int64, error) {
	t := strings.TrimSpace(strings.ToUpper(s))
	if t == "" {
		return 0, fmt.Errorf("empty size")
	}
	mult := int64(1)
	for _, suffix := range []struct {
		text string
		mult int64
	}{
		{"TIB", 1 << 40}, {"TB", 1 << 40}, {"T", 1 << 40},
		{"GIB", 1 << 30}, {"GB", 1 << 30}, {"G", 1 << 30},
		{"MIB", 1 << 20}, {"MB", 1 << 20}, {"M", 1 << 20},
		{"KIB", 1 << 10}, {"KB", 1 << 10}, {"K", 1 << 10},
		{"B", 1},
	} {
		if strings.HasSuffix(t, suffix.text) {
			t = strings.TrimSpace(strings.TrimSuffix(t, suffix.text))
			mult = suffix.mult
			break
		}
	}
	f, err := strconv.ParseFloat(t, 64)
	if err != nil || f < 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	if f > float64(math.MaxInt64)/float64(mult) {
		return 0, fmt.Errorf("size %q is too large", s)
	}
	return int64(f * float64(mult)), nil
}
