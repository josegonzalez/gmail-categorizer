package stats

import "sort"

// SortByInt sorts a slice by an integer field extracted via the given function.
func SortByInt[T any](items []T, extract func(T) int, desc bool) {
	sort.Slice(items, func(i, j int) bool {
		if desc {
			return extract(items[i]) > extract(items[j])
		}
		return extract(items[i]) < extract(items[j])
	})
}

// SortByString sorts a slice by a string field extracted via the given function.
func SortByString[T any](items []T, extract func(T) string, desc bool) {
	sort.Slice(items, func(i, j int) bool {
		if desc {
			return extract(items[i]) > extract(items[j])
		}
		return extract(items[i]) < extract(items[j])
	})
}
