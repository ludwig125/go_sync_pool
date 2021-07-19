package main

import (
	"fmt"
	"sync"
	"testing"
)

var pool = &sync.Pool{
	New: func() interface{} {
		return &[]string{}
	},
}

func ReplicateStrNTimesWithPool(s string, n int) []string {
	cnt++
	ss := pool.Get().(*[]string)

	(*ss) = (*ss)[:0]
	defer pool.Put(ss)
	for i := 0; i < n; i++ {
		(*ss) = append((*ss), s)
	}
	return *ss
}

var Result []string

var cnt int

func BenchmarkReplicateStrNTimesWithPool(b *testing.B) {
	b.ReportAllocs()
	Result = ReplicateStrNTimesWithPool("12345", 1)
	fmt.Printf("\ncnt %d\n", cnt)
}
