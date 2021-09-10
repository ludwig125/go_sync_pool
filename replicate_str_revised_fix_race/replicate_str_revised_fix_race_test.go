package main

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func ReplicateStrNTimes(s string, n int) []string {
	ss := make([]string, n)
	for i := 0; i < n; i++ {
		ss[i] = s
	}
	return ss
}

var pool = &sync.Pool{
	New: func() interface{} {
		return &[]string{}
	},
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

var pool2 = &sync.Pool{
	New: func() interface{} {
		return &[]string{}
	},
}

// https://qiita.com/peroxyacyl/items/5e02ddf4480ecd2ec7b1
// この書き方にならった方が速かった
func ReplicateStrNTimesWithPoolUseArray(s string, n int) []string {
	ss := pool2.Get().(*[]string)

	array := *ss
	// GetしたSliceは前の値を保持しているので、[:0]で空にする
	// [:0]をすると、Sliceの参照先のArrayを解放せず値のみクリアできる
	array = array[:0]
	for i := 0; i < n; i++ {
		array = append(array, s)
	}
	*ss = array
	pool2.Put(ss)
	return array
}

func TestReplicateStrNTimes(t *testing.T) {
	num := 3 // 3回ずつやる
	var wg sync.WaitGroup
	wg.Add(num)
	for i := 0; i < num; i++ {
		i := i
		go func() {
			defer wg.Done()
			want := ReplicateStrNTimes("12345", 10+i)

			var got []string
			fmt.Println("ReplicateStrNTimes"+fmt.Sprintf("%d", i), int(testing.AllocsPerRun(100, func() {
				got = ReplicateStrNTimes("12345", 10+i)
			})))
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got: %s, want: %s", got, want)
			}

			var got2 []string
			fmt.Println("ReplicateStrNTimes"+fmt.Sprintf("%d", i), int(testing.AllocsPerRun(100, func() {
				got2 = ReplicateStrNTimesWithPool("12345", 10+i)
			})))
			if !reflect.DeepEqual(got2, want) {
				t.Errorf("got2: %s, want: %s", got2, want)
			}

			var got3 []string
			fmt.Println("ReplicateStrNTimesWithPoolUseArray"+fmt.Sprintf("%d", i), int(testing.AllocsPerRun(100, func() {
				got3 = ReplicateStrNTimesWithPoolUseArray("12345", 10+i)
			})))
			if !reflect.DeepEqual(got3, want) {
				t.Errorf("got3: %s, want: %s", got3, want)
			}
		}()
	}
	wg.Wait()
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

func BenchmarkReplicateStrNTimesWithPoolUseArray(b *testing.B) {
	b.ReportAllocs()
	var r []string
	for n := 0; n < b.N; n++ {
		r = ReplicateStrNTimesWithPoolUseArray("12345", 5)
	}
	Result = r
}

// $go test -bench . -count=4
// goos: linux
// goarch: amd64
// pkg: github.com/ludwig125/sync-pool/replicate_str_revised_compare
// cpu: Intel(R) Core(TM) i7-6700 CPU @ 3.40GHz
// BenchmarkReplicateStrNTimes-8                           13412271                78.18 ns/op           80 B/op          1 allocs/op
// BenchmarkReplicateStrNTimes-8                           13649438                74.94 ns/op           80 B/op          1 allocs/op
// BenchmarkReplicateStrNTimes-8                           13497962                74.84 ns/op           80 B/op          1 allocs/op
// BenchmarkReplicateStrNTimes-8                           13361154                75.79 ns/op           80 B/op          1 allocs/op
// BenchmarkReplicateStrNTimesWithPool-8                   38979388                30.57 ns/op            0 B/op          0 allocs/op
// BenchmarkReplicateStrNTimesWithPool-8                   39021978                27.56 ns/op            0 B/op          0 allocs/op
// BenchmarkReplicateStrNTimesWithPool-8                   37627934                27.98 ns/op            0 B/op          0 allocs/op
// BenchmarkReplicateStrNTimesWithPool-8                   40602952                27.27 ns/op            0 B/op          0 allocs/op
// BenchmarkReplicateStrNTimesWithPoolUseArray-8            3628519               310.1 ns/op           240 B/op          4 allocs/op
// BenchmarkReplicateStrNTimesWithPoolUseArray-8            3411144               312.1 ns/op           240 B/op          4 allocs/op
// BenchmarkReplicateStrNTimesWithPoolUseArray-8            3697792               326.4 ns/op           240 B/op          4 allocs/op
// BenchmarkReplicateStrNTimesWithPoolUseArray-8            3317565               329.1 ns/op           240 B/op          4 allocs/op
// PASS
// ok      github.com/ludwig125/sync-pool/replicate_str_revised_compare    14.928s
