package par

import (
	"sync/atomic"
	"testing"
)

func TestUnit_RowsCoversEveryRowOnce(t *testing.T) {
	// Arrange
	seen := make([]atomic.Int32, 1000)

	// Act
	Rows(len(seen), func(lo, hi int) {
		for i := lo; i < hi; i++ {
			seen[i].Add(1)
		}
	})

	// Assert
	for i := range seen {
		if seen[i].Load() != 1 {
			t.Fatalf("row %d visited %d times", i, seen[i].Load())
		}
	}
}

func TestUnit_RowsHandlesTinyInput(t *testing.T) {
	// Arrange
	calls := 0

	// Act
	Rows(3, func(lo, hi int) { calls += hi - lo })

	// Assert
	if calls != 3 {
		t.Fatalf("calls %d", calls)
	}
}
