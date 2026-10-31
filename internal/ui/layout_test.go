package ui

import "testing"

func TestComputeLayout(t *testing.T) {
	l := computeLayout(120, 40, true)
	if !l.ShowDetails {
		t.Fatal("expected details pane at 120 cols")
	}
	if l.LeftW+1+l.DetailW != 120 {
		t.Errorf("columns do not add up: %d + 1 + %d", l.LeftW, l.DetailW)
	}
	if l.BodyH != 36 {
		t.Errorf("BodyH = %d, want 36", l.BodyH)
	}
	if l.DetailW < 30 || l.DetailW > 48 {
		t.Errorf("DetailW = %d, want 30..48", l.DetailW)
	}

	if computeLayout(80, 30, true).ShowDetails {
		t.Error("details should be hidden below 110 cols")
	}
	if computeLayout(109, 40, true).ShowDetails {
		t.Error("details should be hidden at 109 cols")
	}
	if !computeLayout(200, 50, true).ShowDetails {
		t.Error("details should be shown at 200 cols")
	}
	if computeLayout(120, 12, true).ShowDetails {
		t.Error("details should be hidden when body height is small")
	}

	tiny := computeLayout(40, 3, false)
	if tiny.ShowTabs || tiny.ShowStatus || !tiny.ShowFooter {
		t.Errorf("chrome flags for tiny layout = %+v", tiny)
	}
	if tiny.BodyH != 1 {
		t.Errorf("tiny BodyH = %d, want 1", tiny.BodyH)
	}

	zero := computeLayout(0, 0, false)
	if zero.Width != 1 || zero.Height != 1 || zero.BodyH != 1 {
		t.Errorf("zero layout = %+v", zero)
	}
}

func TestTooSmall(t *testing.T) {
	cases := []struct {
		w, h int
		want bool
	}{
		{39, 10, true},
		{40, 9, true},
		{40, 10, false},
		{120, 40, false},
	}
	for _, tc := range cases {
		if got := tooSmall(tc.w, tc.h); got != tc.want {
			t.Errorf("tooSmall(%d,%d) = %v, want %v", tc.w, tc.h, got, tc.want)
		}
	}
}

func TestEnsureVisible(t *testing.T) {
	if got := ensureVisible(0, 0, 5, 20); got != 0 {
		t.Errorf("offset = %d, want 0", got)
	}
	if got := ensureVisible(8, 0, 5, 20); got != 4 {
		t.Errorf("offset = %d, want 4", got)
	}
	if got := ensureVisible(2, 8, 5, 20); got != 2 {
		t.Errorf("offset = %d, want 2", got)
	}
	if got := ensureVisible(19, 0, 5, 20); got != 15 {
		t.Errorf("offset = %d, want 15", got)
	}
	if got := ensureVisible(5, 10, 5, 3); got != 0 {
		t.Errorf("clamped offset = %d, want 0", got)
	}
	if got := ensureVisible(-1, -5, 5, 0); got != 0 {
		t.Errorf("empty offset = %d, want 0", got)
	}
}

func TestScrollPercent(t *testing.T) {
	tests := []struct {
		offset, content, height, want int
	}{
		{0, 100, 10, 0},
		{45, 100, 10, 50},
		{90, 100, 10, 100},
		{999, 100, 10, 100},
		{-5, 100, 10, 0},
		{3, 5, 10, 100},
	}
	for _, tc := range tests {
		if got := scrollPercent(tc.offset, tc.content, tc.height); got != tc.want {
			t.Errorf("scrollPercent(%d, %d, %d) = %d, want %d", tc.offset, tc.content, tc.height, got, tc.want)
		}
	}
}

func TestScrollOffset(t *testing.T) {
	if got := scrollOffset(0, -3, 100, 10); got != 0 {
		t.Errorf("scroll = %d, want 0", got)
	}
	if got := scrollOffset(0, 99, 100, 10); got != 90 {
		t.Errorf("scroll = %d, want 90", got)
	}
	if got := scrollOffset(0, 99, 5, 10); got != 0 {
		t.Errorf("short content scroll = %d, want 0", got)
	}
}
