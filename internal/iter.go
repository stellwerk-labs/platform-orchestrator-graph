package internal

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// SortedKeysFunc returns an iterator over the sorted keys in the map. It uses the given compare function
// for sorting.
func SortedKeysFunc[a comparable, b any](in map[a]b, compare func(a, a) int) iter.Seq[a] {
	return func(yield func(a) bool) {
		for _, k := range slices.SortedFunc(maps.Keys(in), compare) {
			if !yield(k) {
				break
			}
		}
	}
}

// SortedKeys is SortedKeysFunc but uses the natural cmp.Compare order.
func SortedKeys[a cmp.Ordered, b any](in map[a]b) iter.Seq[a] {
	return SortedKeysFunc(in, cmp.Compare)
}

// SortedEntriesFunc returns an iterator over the entries, sorted by key using SortedKeysFunc.
func SortedEntriesFunc[a comparable, b any](in map[a]b, compare func(a, a) int) iter.Seq2[a, b] {
	return func(yield func(a, b) bool) {
		for k := range SortedKeysFunc(in, compare) {
			if !yield(k, in[k]) {
				break
			}
		}
	}
}

// SortedEntries is SortedEntriesFunc but uses the natural cmp.Compare order.
func SortedEntries[a cmp.Ordered, b any](in map[a]b) iter.Seq2[a, b] {
	return SortedEntriesFunc(in, cmp.Compare)
}

// CountSeq just counts the items in the given sequence and returns the total.
func CountSeq[a any](in iter.Seq[a]) int {
	var count int
	for range in {
		count++
	}
	return count
}

func ContainsSeq[a comparable](in iter.Seq[a], item a) bool {
	for x := range in {
		if x == item {
			return true
		}
	}
	return false
}
