package main

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

var pool = &sync.Pool{
	New: func() interface{} {
		return &[]string{}
	},
}

func ReplicateStrNTimes(s string, n int) []string {
	ss := make([]string, n)
	for i := 0; i < n; i++ {
		ss[i] = s
	}
	return ss
}

func ReplicateStrNTimesWithPool(s string, n int) []string {
	ss := pool.Get().(*[]string)

	// GetしたSliceは前の値を保持しているので、[:0]で空にする
	// [:0]をすると、Sliceの参照先のArrayを解放せず値のみクリアできる
	(*ss) = (*ss)[:0]
	// (*ss) = nil // メモリ割り当てごと初期化してしまうとかえってアロケーションコストが増える
	defer pool.Put(ss)
	for i := 0; i < n; i++ {
		(*ss) = append((*ss), s)
	}
	return *ss
}

func TestReplicateStrNTimes(t *testing.T) {
	n := 5
	want := []string{
		"12345",
		"12345",
		"12345",
		"12345",
		"12345",
	}

	for i := 0; i < 3; i++ {
		t.Run("ReplicateStrNTimes"+fmt.Sprintf("%d", i), func(t *testing.T) {
			got := ReplicateStrNTimes("12345", n)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
		t.Run("ReplicateStrNTimesWithPool"+fmt.Sprintf("%d", i), func(t *testing.T) {
			got := ReplicateStrNTimesWithPool("12345", n)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
	}
}

var Result []string

func BenchmarkReplicateStrNTimes(b *testing.B) {
	b.ReportAllocs()
	var r []string
	for n := 0; n < b.N; n++ {
		r = ReplicateStrNTimes("12345", 5)
	}
	Result = r
}

func BenchmarkReplicateStrNTimesWithPool(b *testing.B) {
	b.ReportAllocs()
	var r []string
	for n := 0; n < b.N; n++ {
		r = ReplicateStrNTimesWithPool("12345", 5)
	}
	Result = r
}

// [~/go/src/github.com/ludwig125/sync-pool/replicate_str_revised] $go test -bench . -count=4
// goos: linux
// goarch: amd64
// pkg: github.com/ludwig125/sync-pool/replicate_str_revised
// BenchmarkReplicateStrNTimes-8                   14956290                77.2 ns/op            80 B/op          1 allocs/op
// BenchmarkReplicateStrNTimes-8                   13447201                78.7 ns/op            80 B/op          1 allocs/op
// BenchmarkReplicateStrNTimes-8                   13579498                77.7 ns/op            80 B/op          1 allocs/op
// BenchmarkReplicateStrNTimes-8                   13568305                78.7 ns/op            80 B/op          1 allocs/op
// BenchmarkReplicateStrNTimesWithPool-8           35776681                33.7 ns/op             0 B/op          0 allocs/op
// BenchmarkReplicateStrNTimesWithPool-8           38441823                32.2 ns/op             0 B/op          0 allocs/op
// BenchmarkReplicateStrNTimesWithPool-8           35170194                31.8 ns/op             0 B/op          0 allocs/op
// BenchmarkReplicateStrNTimesWithPool-8           36508634                32.2 ns/op             0 B/op          0 allocs/op
// PASS
// ok      github.com/ludwig125/sync-pool/replicate_str_revised    11.372s
