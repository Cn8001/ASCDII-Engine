// Package par splits row loops across cores. Per dot passes touch only their
// own rows, so bands can run at the same time without locks.
package par

import (
	"runtime"
	"sync"
)

// Rows calls fn on contiguous bands [lo, hi) covering [0, n). Small inputs run
// inline because starting goroutines would cost more than the work.
func Rows(n int, fn func(lo, hi int)) {
	workers := min(runtime.GOMAXPROCS(0), n/8)
	if workers <= 1 {
		fn(0, n)
		return
	}
	var wg sync.WaitGroup
	step := (n + workers - 1) / workers
	for lo := 0; lo < n; lo += step {
		hi := min(n, lo+step)
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn(lo, hi)
		}()
	}
	wg.Wait()
}
