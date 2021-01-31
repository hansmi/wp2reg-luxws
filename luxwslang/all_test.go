package luxwslang

import (
	"sort"
	"testing"
)

func TestAllSorted(t *testing.T) {
	allTerms := All()

	if !sort.SliceIsSorted(allTerms, func(i, j int) bool {
		lhs, rhs := allTerms[i], allTerms[j]

		return lhs.ID < rhs.ID
	}) {
		t.Errorf("List of languages is not sorted: %v", allTerms)
	}
}
