package scan

import (
	"container/heap"
	"sort"

	"diskscope/internal/model"
)

// topHeap is a bounded min-heap that retains the largest N files.
type topHeap struct {
	items []model.File
	cap   int
}

func newTopHeap(capacity int) *topHeap {
	if capacity < 1 {
		capacity = 1
	}
	return &topHeap{cap: capacity}
}

func (h *topHeap) Len() int           { return len(h.items) }
func (h *topHeap) Less(i, j int) bool { return h.items[i].Size < h.items[j].Size }
func (h *topHeap) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *topHeap) Push(x any)         { h.items = append(h.items, x.(model.File)) }
func (h *topHeap) Pop() any {
	old := h.items
	n := len(old)
	item := old[n-1]
	h.items = old[:n-1]
	return item
}

func (h *topHeap) offer(f model.File) {
	if f.Size <= 0 {
		return
	}
	if len(h.items) < h.cap {
		heap.Push(h, f)
		return
	}
	if f.Size > h.items[0].Size {
		heap.Pop(h)
		heap.Push(h, f)
	}
}

// offerTracked inserts f and reports the file that had to be evicted (if any).
// A dropped=false return means f was retained. When dropped=true, the returned
// file is the evicted entry (possibly f itself).
func (h *topHeap) offerTracked(f model.File) (evicted model.File, dropped bool) {
	if len(h.items) < h.cap {
		heap.Push(h, f)
		return model.File{}, false
	}
	if f.Size > h.items[0].Size {
		evicted = heap.Pop(h).(model.File)
		heap.Push(h, f)
		return evicted, true
	}
	return f, true
}

func (h *topHeap) sorted() []model.File {
	out := append([]model.File(nil), h.items...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Size != out[j].Size {
			return out[i].Size > out[j].Size
		}
		return out[i].Path < out[j].Path
	})
	return out
}
