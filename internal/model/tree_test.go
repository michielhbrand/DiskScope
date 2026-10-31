package model

import (
	"testing"

	"diskscope/internal/categories"
)

func TestFinalizeAggregatesChildren(t *testing.T) {
	root := &Node{Name: "root", Path: "/root"}
	child := &Node{Name: "child", Path: "/root/child", Parent: root, Depth: 1}
	grandchild := &Node{Name: "grand", Path: "/root/child/grand", Parent: child, Depth: 2}

	root.AddFile(File{Path: "/root/a", Size: 100, Category: categories.Video}, true)
	child.AddFile(File{Path: "/root/child/b", Size: 200, Category: categories.Images}, true)
	child.AddFile(File{Path: "/root/child/small", Size: 30, Category: categories.Other}, false)
	grandchild.AddFile(File{Path: "/root/child/grand/c", Size: 300, Category: categories.Video}, true)
	grandchild.Errors = 2
	child.Errors = 1

	Finalize([]*Node{grandchild, child, root})

	if root.Size != 630 {
		t.Errorf("root.Size = %d, want 630", root.Size)
	}
	if root.Count != 4 {
		t.Errorf("root.Count = %d, want 4", root.Count)
	}
	if root.DirCount != 2 {
		t.Errorf("root.DirCount = %d, want 2", root.DirCount)
	}
	if root.Errors != 3 {
		t.Errorf("root.Errors = %d, want 3", root.Errors)
	}
	if got := root.Categories[categories.Video]; got != 400 {
		t.Errorf("video bytes = %d, want 400", got)
	}
	if got := child.SmallSize; got != 30 {
		t.Errorf("child.SmallSize = %d, want 30", got)
	}
	if got := child.SmallNum; got != 1 {
		t.Errorf("child.SmallNum = %d, want 1", got)
	}
	if root.TopCategory() != categories.Video {
		t.Errorf("TopCategory = %s, want Video", root.TopCategory())
	}
	top := root.TopCategories(2)
	if len(top) != 2 || top[0].Category != categories.Video || top[1].Category != categories.Images {
		t.Errorf("TopCategories = %#v", top)
	}
}

func TestFinalizeDoesNotDoubleApply(t *testing.T) {
	root := &Node{Path: "/root"}
	child := &Node{Path: "/root/c", Parent: root, Depth: 1}
	child.AddFile(File{Size: 10, Category: categories.Other}, true)
	Finalize([]*Node{child, root})
	Finalize([]*Node{child, root}) // safe to call again
	if root.Size != 10 {
		t.Fatalf("root.Size = %d, want 10", root.Size)
	}
}
