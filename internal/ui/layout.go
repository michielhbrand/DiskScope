package ui

// layout describes the available space after chrome is subtracted.
type layout struct {
	Width, Height int
	BodyW, BodyH  int

	ShowTabs    bool
	ShowStatus  bool
	ShowFooter  bool
	ShowDetails bool
	DetailW     int
	LeftW       int
}

func computeLayout(w, h int, wantDetails bool) layout {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	l := layout{Width: w, Height: h}
	l.ShowTabs = h >= 6
	l.ShowStatus = h >= 4
	l.ShowFooter = h >= 3

	chrome := 1
	if l.ShowTabs {
		chrome++
	}
	if l.ShowStatus {
		chrome++
	}
	if l.ShowFooter {
		chrome++
	}
	l.BodyH = h - chrome
	if l.BodyH < 1 {
		l.BodyH = 1
	}
	l.BodyW = w
	l.LeftW = w

	if wantDetails && w >= 110 && l.BodyH >= 10 {
		detailW := w * 32 / 100
		if detailW < 30 {
			detailW = 30
		}
		if detailW > 48 {
			detailW = 48
		}
		if w-detailW-1 >= 60 {
			l.ShowDetails = true
			l.DetailW = detailW
			l.LeftW = w - detailW - 1
		}
	}
	return l
}

func tooSmall(w, h int) bool {
	return w < 40 || h < 10
}

// ensureVisible adjusts offset so selected remains inside a viewport.
func ensureVisible(selected, offset, height, total int) int {
	if total <= 0 {
		return 0
	}
	if height < 1 {
		height = 1
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= total {
		selected = total - 1
	}
	if selected < offset {
		offset = selected
	}
	if selected >= offset+height {
		offset = selected - height + 1
	}
	maxOffset := total - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}

// scrollOffset is used for free-scrolling views such as overview/help.
func scrollOffset(offset, delta, content, height int) int {
	maxOffset := content - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset += delta
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	return offset
}

// scrollPercent reports how far a free-scrolling view has been scrolled.
func scrollPercent(offset, content, height int) int {
	maxOffset := content - height
	if maxOffset <= 0 {
		return 100
	}
	if offset <= 0 {
		return 0
	}
	if offset >= maxOffset {
		return 100
	}
	return offset * 100 / maxOffset
}
