package internal

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortedKeys(t *testing.T) {
	assert.Equal(t, []string(nil), slices.Collect(SortedKeys(map[string]bool{})))
	assert.Equal(t, []string{"a", "m", "z"}, slices.Collect(SortedKeys(map[string]bool{"z": true, "m": false, "a": true})))
	for x := range SortedKeys(map[string]bool{"z": true, "m": false, "a": true}) {
		assert.Equal(t, "a", x)
		break
	}
}

func TestSortedEntries(t *testing.T) {
	for range SortedEntries(map[string]bool{}) {
		assert.Fail(t, "should not reach")
	}
	seen := make([]string, 0)
	for k, v := range SortedEntries(map[string]bool{"z": true, "m": false, "a": true}) {
		seen = append(seen, fmt.Sprintf("%v: %v", k, v))
		if len(seen) >= 2 {
			break
		}
	}
	assert.Equal(t, []string{"a: true", "m: false"}, seen)
}

func TestCount(t *testing.T) {
	assert.Equal(t, 0, CountSeq(slices.Values([]int{})))
	assert.Equal(t, 3, CountSeq(slices.Values([]int{1, 2, 3})))
}
