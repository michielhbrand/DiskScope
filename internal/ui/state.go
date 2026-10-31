package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"diskscope/internal/categories"
	"diskscope/internal/model"
	"diskscope/internal/scan"
)

type sortKey int

const (
	sortSize sortKey = iota
	sortName
	sortCount
	sortModified
)

func (s sortKey) label() string {
	switch s {
	case sortName:
		return "name"
	case sortCount:
		return "files"
	case sortModified:
		return "modified"
	default:
		return "size"
	}
}

func (s sortKey) defaultDesc() bool { return s != sortName }

func nextSortKey(s sortKey) sortKey {
	return sortKey((int(s) + 1) % 4)
}

type item struct {
	Name     string
	Path     string
	Size     int64
	Count    int64
	IsDir    bool
	Rollup   bool
	Category categories.Category
	ModTime  time.Time
	Mode     os.FileMode
	Node     *model.Node
}

func itemFromFile(f model.File) item {
	return item{
		Name:     f.Name,
		Path:     f.Path,
		Size:     f.Size,
		Count:    1,
		Category: f.Category,
		ModTime:  f.ModTime,
		Mode:     f.Mode,
	}
}

func itemsForNode(n *model.Node) []item {
	if n == nil {
		return nil
	}
	items := make([]item, 0, len(n.Children)+len(n.Files)+1)
	for _, child := range n.Children {
		items = append(items, item{
			Name:     child.Name,
			Path:     child.Path,
			Size:     child.Size,
			Count:    child.Count,
			IsDir:    true,
			Category: child.TopCategory(),
			ModTime:  child.ModTime,
			Mode:     child.Mode,
			Node:     child,
		})
	}
	for _, f := range n.Files {
		items = append(items, itemFromFile(f))
	}
	return items
}

func rollupItem(n *model.Node) (item, bool) {
	if n == nil || n.SmallNum == 0 {
		return item{}, false
	}
	return item{
		Name:     fmt.Sprintf("%s smaller files", FormatCount(n.SmallNum)),
		Size:     n.SmallSize,
		Count:    n.SmallNum,
		Rollup:   true,
		Category: categories.Other,
	}, true
}

func filterItems(items []item, filter string) []item {
	filter = strings.TrimSpace(strings.ToLower(filter))
	if filter == "" {
		return items
	}
	out := items[:0]
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.Name), filter) ||
			strings.Contains(strings.ToLower(it.Path), filter) {
			out = append(out, it)
		}
	}
	return out
}

func sortItems(items []item, key sortKey, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		var less, equal bool
		switch key {
		case sortName:
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
			equal = strings.EqualFold(a.Name, b.Name)
		case sortCount:
			less, equal = a.Count < b.Count, a.Count == b.Count
		case sortModified:
			less, equal = a.ModTime.Before(b.ModTime), a.ModTime.Equal(b.ModTime)
		default:
			less, equal = a.Size < b.Size, a.Size == b.Size
		}
		if equal {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		if desc {
			return !less
		}
		return less
	})
}

type browserState struct {
	stack    []*model.Node
	items    []item
	selected int
	offset   int
	sortKey  sortKey
	sortDesc bool
	filter   string
}

func newBrowserState(root *model.Node) browserState {
	b := browserState{
		stack:    []*model.Node{root},
		sortKey:  sortSize,
		sortDesc: true,
	}
	b.rebuild()
	return b
}

func (b *browserState) current() *model.Node {
	if len(b.stack) == 0 {
		return nil
	}
	return b.stack[len(b.stack)-1]
}

func (b *browserState) rebuild() {
	cur := b.current()
	items := filterItems(itemsForNode(cur), b.filter)
	sortItems(items, b.sortKey, b.sortDesc)
	if roll, ok := rollupItem(cur); ok && b.filter == "" {
		items = append(items, roll)
	}
	b.items = items
	if b.selected >= len(b.items) {
		b.selected = len(b.items) - 1
	}
	if b.selected < 0 {
		b.selected = 0
	}
	b.offset = ensureVisible(b.selected, b.offset, 1, len(b.items))
}

func (b *browserState) selectedItem() (item, bool) {
	if b.selected < 0 || b.selected >= len(b.items) {
		return item{}, false
	}
	return b.items[b.selected], true
}

func (b *browserState) move(delta int) {
	if len(b.items) == 0 {
		b.selected = 0
		return
	}
	b.selected += delta
	if b.selected < 0 {
		b.selected = 0
	}
	if b.selected >= len(b.items) {
		b.selected = len(b.items) - 1
	}
}

func (b *browserState) enter() bool {
	it, ok := b.selectedItem()
	if !ok || it.Rollup || !it.IsDir || it.Node == nil {
		return false
	}
	b.stack = append(b.stack, it.Node)
	b.selected = 0
	b.offset = 0
	b.rebuild()
	return true
}

func (b *browserState) up() bool {
	if len(b.stack) <= 1 {
		return false
	}
	b.stack = b.stack[:len(b.stack)-1]
	b.selected = 0
	b.offset = 0
	b.rebuild()
	return true
}

func (b *browserState) cycleSort() {
	b.sortKey = nextSortKey(b.sortKey)
	b.sortDesc = b.sortKey.defaultDesc()
	b.rebuild()
}

func (b *browserState) reverseSort() {
	b.sortDesc = !b.sortDesc
	b.rebuild()
}

type categoryRow struct {
	Cat   categories.Category
	Size  int64
	Files int64
}

type categoryState struct {
	rows         []categoryRow
	selected     int
	offset       int
	detailCat    categories.Category
	detailItems  []item
	detailSel    int
	detailOffset int
}

func newCategoryState(res *scan.Result) categoryState {
	var c categoryState
	if res == nil {
		return c
	}
	for _, cat := range categories.All() {
		size := res.Root.Categories[cat]
		if size > 0 {
			c.rows = append(c.rows, categoryRow{
				Cat:   cat,
				Size:  size,
				Files: res.CategoryCount[cat],
			})
		}
	}
	sort.SliceStable(c.rows, func(i, j int) bool {
		if c.rows[i].Size != c.rows[j].Size {
			return c.rows[i].Size > c.rows[j].Size
		}
		return c.rows[i].Cat.String() < c.rows[j].Cat.String()
	})
	return c
}

func (c *categoryState) selectedRow() (categoryRow, bool) {
	if c.selected < 0 || c.selected >= len(c.rows) {
		return categoryRow{}, false
	}
	return c.rows[c.selected], true
}

func (c *categoryState) move(delta int) {
	if len(c.rows) == 0 {
		c.selected = 0
		return
	}
	c.selected += delta
	if c.selected < 0 {
		c.selected = 0
	}
	if c.selected >= len(c.rows) {
		c.selected = len(c.rows) - 1
	}
}

func (c *categoryState) openDetail(res *scan.Result) bool {
	row, ok := c.selectedRow()
	if !ok || res == nil {
		return false
	}
	files := res.CategoryTop[row.Cat]
	c.detailCat = row.Cat
	c.detailItems = c.detailItems[:0]
	for _, f := range files {
		c.detailItems = append(c.detailItems, itemFromFile(f))
	}
	sortItems(c.detailItems, sortSize, true)
	c.detailSel = 0
	c.detailOffset = 0
	return true
}

func (c *categoryState) moveDetail(delta int) {
	if len(c.detailItems) == 0 {
		c.detailSel = 0
		return
	}
	c.detailSel += delta
	if c.detailSel < 0 {
		c.detailSel = 0
	}
	if c.detailSel >= len(c.detailItems) {
		c.detailSel = len(c.detailItems) - 1
	}
}

func (c *categoryState) selectedDetail() (item, bool) {
	if c.detailSel < 0 || c.detailSel >= len(c.detailItems) {
		return item{}, false
	}
	return c.detailItems[c.detailSel], true
}
