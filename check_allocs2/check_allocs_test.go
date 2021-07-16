package main

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
)

var pool = &sync.Pool{
	New: func() interface{} {
		return &[]int{}
	},
}

func AddNum(n int) []int {
	l := pool.Get().(*[]int)
	defer pool.Put(l)

	(*l) = (*l)[:0]
	(*l) = append((*l), n)

	return (*l)
}

func TestAddNum(t *testing.T) {
	fmt.Println("MyAllocs:", int(MyAllocsPerRun(2, func() {
		AddNum(1)
	})))
}

func MyAllocsPerRun(runs int, f func()) (avg float64) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))

	f()

	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)
	mallocs := 0 - memstats.Mallocs
	before := memstats.Mallocs // 関数実行前のmallocs

	for i := 0; i < runs; i++ {
		f()
	}

	runtime.ReadMemStats(&memstats)
	mallocs += memstats.Mallocs

	fmt.Printf("mallocs: %v, memstats.Mallocs(before %d ->after %d). run: %d\n", mallocs, before, memstats.Mallocs, uint64(runs))

	return float64(mallocs / uint64(runs))
}
