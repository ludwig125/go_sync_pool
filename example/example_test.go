package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestLog(t *testing.T) {
	// Log関数も、LogWithoutPool関数も動作の内容自体は同じであることを保証するテスト
	// 標準出力os.Stdoutの代わりに、byte.Bufferに書きだして、それがwantと同じことを確認する

	want := "2006-01-02T15:04:05Z test_path=/test?q=balls"

	// Log関数が何回実行しても同じ結果か確認するため、
	// ２回実行している。
	// Log関数内でbufPool.Get().(*bytes.Buffer)の後に
	// b.Reset()を呼ばないと、２回目の実行では１回目と合わせて
	// 以下のように重複したデータになる
	// 2006-01-02T15:04:05Z test_path=/test?q=balls2006-01-02T15:04:05Z test_path=/test?q=balls
	for i := 0; i < 2; i++ {
		t.Run("Log"+fmt.Sprintf("%d", i), func(t *testing.T) {
			buf := &bytes.Buffer{}
			Log(buf, "test_path", "/test?q=balls")
			got := buf.String()
			if got != want {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
		t.Run("LogWithoutPool"+fmt.Sprintf("%d", i), func(t *testing.T) {
			buf := &bytes.Buffer{}
			LogWithoutPool(buf, "test_path", "/test?q=balls")
			got := buf.String()
			if got != want {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
	}
}

var globalBuf *bytes.Buffer

func init() {
	// 関数内の処理が外部に何も影響を与えないと、
	// コンパイラの最適化で無視されてBenchmarkが正しく測れないことがあるので
	// 最初にグローバルのバッファを宣言しておいて、
	// Benchmark内で上書きする。
	globalBuf = &bytes.Buffer{}
}

func BenchmarkLog(b *testing.B) {
	b.ReportAllocs() // allocation結果を確認するため最初に呼び出す
	buf := &bytes.Buffer{}
	for n := 0; n < b.N; n++ {
		Log(buf, "this_path", "/test?q=query&format=json&groupid=100001&area=200000001")
	}
	globalBuf = buf
}

func BenchmarkLogWithoutPool(b *testing.B) {
	b.ReportAllocs()
	buf := &bytes.Buffer{}
	for n := 0; n < b.N; n++ {
		LogWithoutPool(buf, "this_path", "/test?q=query&format=json&groupid=100001&area=200000001")
	}
	globalBuf = buf
}

// $go test -bench . -benchmem -count=4
// goos: linux
// goarch: amd64
// pkg: github.com/ludwig125/sync-pool/example
// BenchmarkLog-8                   3310735               329 ns/op             249 B/op          1 allocs/op
// BenchmarkLog-8                   3708666               311 ns/op             226 B/op          1 allocs/op
// BenchmarkLog-8                   3619632               309 ns/op             231 B/op          1 allocs/op
// BenchmarkLog-8                   3715123               307 ns/op             226 B/op          1 allocs/op
// BenchmarkLogWithoutPool-8        2733618               424 ns/op             551 B/op          3 allocs/op
// BenchmarkLogWithoutPool-8        2826265               417 ns/op             543 B/op          3 allocs/op
// BenchmarkLogWithoutPool-8        2785218               423 ns/op             547 B/op          3 allocs/op
// BenchmarkLogWithoutPool-8        2785072               411 ns/op             547 B/op          3 allocs/op
// PASS
// ok      github.com/ludwig125/sync-pool/example  12.380s
