package views

// VisibleRange represents the start and end indices for visible items.
type VisibleRange struct {
	Start, End int
}

// CalculateVisibleRange computes which items should be visible based on
// cursor position and available height.
func CalculateVisibleRange(totalItems, cursor, height, headerOffset int) VisibleRange {
	visibleItems := height - headerOffset
	if visibleItems < 1 {
		visibleItems = 10
	}

	start := 0
	if cursor >= visibleItems {
		start = cursor - visibleItems + 1
	}

	end := start + visibleItems
	if end > totalItems {
		end = totalItems
	}

	return VisibleRange{Start: start, End: end}
}
