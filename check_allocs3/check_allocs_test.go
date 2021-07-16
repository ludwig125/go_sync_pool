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
	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)
	mallocs := 0 - memstats.Mallocs
	before := memstats.Mallocs

	l := pool.Get().(*[]int)

	runtime.ReadMemStats(&memstats)
	mallocs += memstats.Mallocs
	fmt.Printf("mallocs: %v, memstats.Mallocs(before %d ->after %d). IN FUNC\n", mallocs, before, memstats.Mallocs)

	defer pool.Put(l)
	(*l) = (*l)[:0]

	(*l) = append((*l), n)
	fmt.Printf("*l: %v, len(*l): %v, cap(*l): %v\n", *l, len(*l), cap(*l))
	return (*l)
}

func TestAddNum(t *testing.T) {
	AddNum(1)
	AddNum(2)
}
