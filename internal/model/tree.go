// Package model contains the aggregated filesystem tree used by the scanner and UI.
package model

import (
	"os"
	"time"

	"diskscope/internal/categories"
)

// File is a retained file entry. Only files above the scanner threshold and the
// largest files per directory are retained; everything else is aggregated.
type File struct {
	Name     string
	Path     string
	Size     int64
	Category categories.Category
	ModTime  time.Time
	Mode     os.FileMode
}

// Node is a directory with recursively aggregated usage statistics.
type Node struct {
	Name     string
	Path     string
	Parent   *Node
	Children []*Node
	Files    []File
	Depth    int

	// Size and Count are aggregated over the whole subtree after Finalize.
	Size      int64
	Count     int64 // files
	DirCount  int64 // descendant directories
	Errors    int64
	SmallSize int64 // bytes in files too small/too many to retain
	SmallNum  int64 // number of aggregated files

	Categories [categories.Count]int64
	ModTime    time.Time
	Mode       os.FileMode

	finalized bool
}

// AddFile records a file in this directory. The caller has already decided
// whether the file itself is retained.
func (n *Node) AddFile(f File, retain bool) {
	n.Size += f.Size
	n.Count++
	n.Categories[f.Category] += f.Size
	if retain {
		n.Files = append(n.Files, f)
	} else {
		n.SmallSize += f.Size
		n.SmallNum++
	}
}

// TopCategory returns the category with the largest byte total in this subtree.
func (n *Node) TopCategory() categories.Category {
	best := categories.Other
	var bestSize int64
	for i, size := range n.Categories {
		if size > bestSize {
			best = categories.Category(i)
			bestSize = size
		}
	}
	return best
}

// TopCategories returns up to limit categories ordered by descending size.
type CategorySize struct {
	Category categories.Category
	Size     int64
}

func (n *Node) TopCategories(limit int) []CategorySize {
	out := make([]CategorySize, 0, categories.Count)
	for i, size := range n.Categories {
		if size > 0 {
			out = append(out, CategorySize{Category: categories.Category(i), Size: size})
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Size > out[j-1].Size; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Finalize aggregates child directory totals into their parents. Nodes must be
// sorted by descending Depth before this is called; the scanner does this.
func Finalize(nodes []*Node) {
	for _, n := range nodes {
		if n.finalized {
			continue
		}
		if p := n.Parent; p != nil {
			p.Size += n.Size
			p.Count += n.Count
			p.DirCount += n.DirCount + 1
			p.Errors += n.Errors
			for i := range p.Categories {
				p.Categories[i] += n.Categories[i]
			}
		}
		n.finalized = true
	}
}
