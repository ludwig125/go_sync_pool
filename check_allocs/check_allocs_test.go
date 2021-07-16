package main

import (
	"fmt"
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
	fmt.Println("Allocs:", int(testing.AllocsPerRun(1, func() {
		AddNum(1)
	})))
}
