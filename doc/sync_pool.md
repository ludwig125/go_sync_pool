# golangのsync.Pool

Go言語のsync.Poolをうまく使えば性能改善できる、という話を見たので自分の理解を深めつついろいろ検証してみました

# 公式の説明

https://golang.org/pkg/sync/#Pool

簡単にまとめると、

- Poolは、個別に保存や取得が可能な一時的なオブジェクトの集合体
- Poolの中身は、突然削除される可能性がある（GCとかで）
- Poolは、複数のゴルーチンが同時に使用してもOK
- Poolの目的は、割り当てられたものの未使用のアイテムを、
  後で再利用するためにキャッシュし、ガベージコレクタの負担を軽減すること

ということで、Poolに保存したキャッシュを使いまわすことで処理の効率化を狙うことができます

# 簡単な例

簡単な例で動作を確認してみます。

```golang
package main

import (
	"fmt"
	"runtime"
	"sync"
)

func main() {
	pool := &sync.Pool{ // <1> poolの定義
		New: func() interface{} { // Poolから最初にGetした時はこのNew関数が呼ばれる
			return &[]int{}
		},
	}

	// append 10
	l := pool.Get().(*[]int) // <2> poolから取得。[]int{}が取れる
	fmt.Println("got slice", *l)
	(*l) = append((*l), 10)
	fmt.Println("after append", *l)
	pool.Put(l) // <3> poolに戻す

	// append 20
	l = pool.Get().(*[]int) // <4> poolから取得。[]int{10}が取れる
	fmt.Println("got slice", *l)
	(*l) = append((*l), 20)
	fmt.Println("after append", *l)
	pool.Put(l) // poolに戻す

	// ガベージコレクションをしてpoolの中身を消す
	runtime.GC() // <5> GCをすると一次的なキャッシュのPoolの中身は消える

	// append 30
	l = pool.Get().(*[]int) // <6> poolから取得。[]int{}が取れる
	fmt.Println("got slice", *l)
	(*l) = append((*l), 30)
	fmt.Println("after append", *l)
	pool.Put(l) // poolに戻す
}

```

このプログラムの実行結果は以下の通りです。

```
[~/go/src/github.com/ludwig125/sync-pool] $go run simple/simple.go
got slice []
after append [10]
got slice [10]
after append [10 20]
got slice []
after append [30]
```

このプログラムを順番に説明します。

#### <1> poolの定義

- 最初にPoolを定義します
  - この例ではmain関数内で定義していますが、様々な関数から呼び出す場合はGlobal変数として定義することが多いです
- Poolから最初にGetしたときに実行されるNew関数を定義しておきます
- New関数では、pointerを返す必要があり、返り値はinterface型にするという注意点があります

#### <2> PoolからGet

- Poolから最初にGetした時は初めてインスタンスを作成するので、poolに定義したNew関数が実行されます
- New関数の返り値はinterface型なので、型アサーション `.(*[]int)` をする必要があります

- 私見ですが、型アサーションはプログラムを実行するまで気づけないので、重要な処理の場合はsync.Poolを使ったコードはテストによる動作の確認が必須と言えそうです

#### <3> Put

- PutでPoolに値を戻します
- 一度Putで値を戻すと、次回以降のGetでは前の値を使うことができます

#### <4> ２回目のGet

- ２回目以降のGetでは前のPoolの値が取得できます

#### <5> GCでPoolを消す

- Poolはただのキャッシュなので、ガベージコレクション（GC）が走ると消えます
- 公式のドキュメントではそれに該当する説明が以下にあります

https://pkg.go.dev/sync#Pool

> Pool's purpose is to cache allocated but unused items for later reuse, relieving pressure on the garbage collector. That is, it makes it easy to build efficient, thread-safe free lists. However, it is not suitable for all free lists.

- ここでは意図的に`runtime.GC`関数でPoolの中身を消しています

#### <6> 再度Poolから取得

- 前述までのPoolの中身はGCで消えてしまったので、Getすると再びNew関数が呼ばれます

# 公式のexample

もう少しだけ実用的な使い方を見てみます。

公式のexampleを見ると以下のようなコードがありました

https://golang.org/pkg/sync/#example_Pool

```golang
package main

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		// The Pool's New function should generally only return pointer
		// types, since a pointer can be put into the return interface
		// value without an allocation:
		return new(bytes.Buffer) // <1>
	},
}

// timeNow is a fake version of time.Now for tests.
func timeNow() time.Time {
	return time.Unix(1136214245, 0)
}

func Log(w io.Writer, key, val string) {
	b := bufPool.Get().(*bytes.Buffer) // <2>
	b.Reset() // <4>
	// Replace this with time.Now() in a real logger.
	b.WriteString(timeNow().UTC().Format(time.RFC3339))
	b.WriteByte(' ')
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(val)
	w.Write(b.Bytes())
	bufPool.Put(b) // <3>
}

func main() {
	Log(os.Stdout, "path", "/search?q=flowers")
}
```

## 上の処理の内容

この例でも、同様に順番に見ていきます

#### <1>, <2>

b(*bytes.Buffer)をbufPoolから取得しています。

（もしPoolを使わない単純な方法であれば、この部分は、
`b := new(bytes.Buffer)`または、`b := &bytes.Buffer{}`と書けるでしょう）

まだインスタンスが初期化されていない場合は、
bufPoolのGetメソッドを呼び出すことで、事前に定義したNew関数が呼び出されて、
`new(bytes.Buffer)`で bytes.Bufferのメモリが確保されます。

#### <3>

やりたい処理が終わったら、
bufPoolにPutメソッドを使ってbを戻しています。

#### <4>

２回目以降はGetすると前の中身を取ってきてしまうので、
`b.Reset()`で値を空にしています。

## 公式のExampleコードの動作確認

上のコードの動作確認をしてみます。
せっかくなので、Poolを使わないバージョンである`LogWithoutPool`関数も作ってみました

```golang
package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// https://golang.org/pkg/sync/#example_Pool

var bufPool = sync.Pool{
	New: func() interface{} {
		// The Pool's New function should generally only return pointer
		// types, since a pointer can be put into the return interface
		// value without an allocation:
		return new(bytes.Buffer)
	},
}

// timeNow is a fake version of time.Now for tests.
func timeNow() time.Time {
	return time.Unix(1136214245, 0) // 2006-01-02T15:04:05Z
}

func Log(w io.Writer, key, val string) {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	// Replace this with time.Now() in a real logger.
	b.WriteString(timeNow().UTC().Format(time.RFC3339))
	b.WriteByte(' ')
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(val)
	if _, err := w.Write(b.Bytes()); err != nil {
		// エラーチェックしないとLinterが警告出したのでチェックを追加
		log.Fatal(err)
	}
	bufPool.Put(b)
}

// Log関数のPoolを使わない版
func LogWithoutPool(w io.Writer, key, val string) {
	b := &bytes.Buffer{}
	// Replace this with time.Now() in a real logger.
	b.WriteString(timeNow().UTC().Format(time.RFC3339))
	b.WriteByte(' ')
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(val)
	if _, err := w.Write(b.Bytes()); err != nil {
		log.Fatal(err)
	}
}

func main() {
	Log(os.Stdout, "path", "/search?q=flowers")
	fmt.Println() // 改行
	LogWithoutPool(os.Stdout, "path", "/search?q=flowers")
}
```

このコードを実行すると以下のようになります。

```
 $go run example.go
2006-01-02T15:04:05Z path=/search?q=flowers
2006-01-02T15:04:05Z path=/search?q=flowers
```

## 公式のExampleコードのTest

さっと動作確認はしましたが、念のためテストコードを書いておくとこのようになります

２回実行しているのは、上で説明した`b.Reset()`が機能していることの確認です。

```golang
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
```

## 公式のExampleコードのBenchmark

Poolを使わない`LogWithoutPool`も、元の`Log`も、挙動としては同じです。

### Benchmark関数

この２つの関数のBenchmark用コードを書いて結果を比較してみます。

```golang
package main

import (
	"bytes"
	"testing"
)

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

```

### Benchmarkの取り方についての注意

上のBenchmarkでは、各ループの処理内で`os.Stdout`の代わりに`buf`に書き込んで、さらに最後に`buf`を`globalBuf`というグローバル変数に書き込んでいます。

一見無意味に見える、この変数の書き込みは、
Compilerの最適化を防止するためのものです。

もし関数内を以下のようにしてしまうと、for文の中の処理が外部に何も影響を与えないので、Compilerが最適化を行い中の処理をまるごと無視することがあるそうです。
（「あるそうです」と書いたのは、Compilerの最適化について以下の記事で言及されていたのですが、私の環境では違いを確認できなかったためです）

```golang
func BenchmarkLogWrong(b *testing.B) {
	for n := 0; n < b.N; n++ {
		buf := &bytes.Buffer{}
		Log(buf, "this_path", "/test?q=query&format=json&groupid=100001&area=200000001")
	}
}
```

Compiler最適化を防ぐBenchmarkの取り方については、
以下の記事が非常に参考になります。

- https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go
- https://dave.cheney.net/high-performance-go-workshop/gophercon-2019.html#watch_out_for_compiler_optimisations


### Benchmarkの実行

上のBenchmarkを実行すると私の環境では以下の結果になりました
（WSL2 Ubuntuです）

```
$go test -bench . -count=4
goos: linux
goarch: amd64
pkg: github.com/ludwig125/sync-pool/example
BenchmarkLog-8                   3310735               329 ns/op             249 B/op          1 allocs/op
BenchmarkLog-8                   3708666               311 ns/op             226 B/op          1 allocs/op
BenchmarkLog-8                   3619632               309 ns/op             231 B/op          1 allocs/op
BenchmarkLog-8                   3715123               307 ns/op             226 B/op          1 allocs/op
BenchmarkLogWithoutPool-8        2733618               424 ns/op             551 B/op          3 allocs/op
BenchmarkLogWithoutPool-8        2826265               417 ns/op             543 B/op          3 allocs/op
BenchmarkLogWithoutPool-8        2785218               423 ns/op             547 B/op          3 allocs/op
BenchmarkLogWithoutPool-8        2785072               411 ns/op             547 B/op          3 allocs/op
PASS
ok      github.com/ludwig125/sync-pool/example  12.380s
```

結果に誤差が生じることを考えて、`-count=4`をつけて４回ずつ実行しています。

ちなみに、`go test -bench . -benchmem`のように`-benchmem`をつければ、
Benchmark関数内で`b.ReportAllocs()`を書かなくてもメモリのアロケーションが出力されます。

### Benchmark結果について

BenchmarkLogWithoutPoolに比べて、BenchmarkLogの方がメモリのアロケーションが少なく済んでいます。

下は左が１ループごとのアロケーションされたバイト数、
右が１ループごとのアロケーション回数を意味します。

【メモリアロケーション】Log関数のメモリアロケーション
```
249 B/op          1 allocs/op
226 B/op          1 allocs/op
231 B/op          1 allocs/op
226 B/op          1 allocs/op
```

【メモリアロケーション】LogWithoutPool関数のメモリアロケーション
```
551 B/op          3 allocs/op
543 B/op          3 allocs/op
547 B/op          3 allocs/op
547 B/op          3 allocs/op
```


また、これにより、実行時間もLog関数の方が多少速くなっています。

【実行回数と速度】Log関数のループが実行された回数（左）と１ループごとの所要時間（右）
```
3310735               329 ns/op
3708666               311 ns/op
3619632               309 ns/op
3715123               307 ns/op
```

【実行回数と速度】LogW ithoutPool関数のループが実行された回数（左）と１ループごとの所要時間（右）
```
2733618               424 ns/op
2826265               417 ns/op
2785218               423 ns/op
2785072               411 ns/op
```

sync.Poolを使った処理では、GetとPutを呼び出す分余計に時間がかかりますが、
それでも今回のLog関数の場合はPoolを使った方がメモリに優しく処理も速いということになりました。

# sync.Poolを使ったSlice操作の例

sync.Poolの練習用に、以下のような、与えられた文字列を５つ複製したSliceとして返す関数を考えてみます。

```golang
func ReplicateStrNTimes(s string) []string {
	n := 5
	ss := make([]string, n)
	for i := 0; i < n; i++ {
		ss[i] = s
	}
	return ss
}
```

要素数が事前に５とわかっているので、appendを使わず以下のように
要素番号を指定して代入するのが処理としては最速なはずです。
```golang
ss := make([]string, n)
for i := 0; i < n; i++ {
	ss[i] = s
}
```

この関数に例えば`abc`という文字列を与えると、
`[]slice{"abc","abc","abc","abc","abc"}`が返ってきます。

## sync.Poolでの高速化

上のReplicateStrNTimes関数をsync.Poolを使って書き直すと以下のようになります。

```golang
var pool = &sync.Pool{
	New: func() interface{} {
		mem := make([]string, 5)
		return &mem
	},
}

func ReplicateStrNTimesWithPool(s string) []string {
	n := 5
	ss := pool.Get().(*[]string)
	defer pool.Put(ss)
	for i := 0; i < n; i++ {
		(*ss)[i] = s
	}
	return *ss
}
```

Exampleのコードと同様に、扱いたいもの（ここでは`[]string`）のポインタをNewするようにしています。

ここで、注意点としてはNew関数で`make([]string, 5)`とLengthを5と宣言してしまったことです。

最初のReplicateStrNTimesをまねて要素番号に直接代入する方法を取った以上、New関数でlengthを指定しないといけなくなってしまいました。
```golang
for i := 0; i < n; i++ {
	(*ss)[i] = s
}
```

これだと５以外が指定できないのであとで書き直しますが、
とりあえずこれでBenchmarkを取ってみます。


```golang
package main

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

var pool = &sync.Pool{
	New: func() interface{} {
		mem := make([]string, 5)
		return &mem
	},
}

func ReplicateStrNTimes(s string) []string {
	n := 5
	ss := make([]string, n)
	for i := 0; i < n; i++ {
		ss[i] = s
	}
	return ss
}

func ReplicateStrNTimesWithPool(s string) []string {
	n := 5
	ss := pool.Get().(*[]string)
	defer pool.Put(ss)
	for i := 0; i < n; i++ {
		(*ss)[i] = s
	}
	return *ss
}

func TestReplicateStrNTimes(t *testing.T) {
	want := []string{
		"12345",
		"12345",
		"12345",
		"12345",
		"12345",
	}

	for i := 0; i < 2; i++ {
		count := fmt.Sprintf("%d", i)
		t.Run("ReplicateStrNTimes"+count, func(t *testing.T) {
			got := ReplicateStrNTimes("12345")
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
		t.Run("ReplicateStrNTimesWithPool"+count, func(t *testing.T) {
			got := ReplicateStrNTimes("12345")
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
		r = ReplicateStrNTimes("12345")
	}
	Result = r
}

func BenchmarkReplicateStrNTimesWithPool(b *testing.B) {
	b.ReportAllocs()
	var r []string
	for n := 0; n < b.N; n++ {
		r = ReplicateStrNTimesWithPool("12345")
	}
	Result = r
}

```

Benchmark結果
```
[~/go/src/github.com/ludwig125/sync-pool/replicate_str] $go test -bench . -count=4
goos: linux
goarch: amd64
pkg: github.com/ludwig125/sync-pool/replicate_str
BenchmarkReplicateStrNTimes-8                   12753705                82.7 ns/op            80 B/op          1 allocs/op
BenchmarkReplicateStrNTimes-8                   15265990                76.3 ns/op            80 B/op          1 allocs/op
BenchmarkReplicateStrNTimes-8                   15168483                77.4 ns/op            80 B/op          1 allocs/op
BenchmarkReplicateStrNTimes-8                   14881579                77.0 ns/op            80 B/op          1 allocs/op
BenchmarkReplicateStrNTimesWithPool-8           45195015                26.2 ns/op             0 B/op          0 allocs/op
BenchmarkReplicateStrNTimesWithPool-8           41568518                25.2 ns/op             0 B/op          0 allocs/op
BenchmarkReplicateStrNTimesWithPool-8           41879115                25.1 ns/op             0 B/op          0 allocs/op
BenchmarkReplicateStrNTimesWithPool-8           44395609                24.3 ns/op             0 B/op          0 allocs/op
PASS
ok      github.com/ludwig125/sync-pool/replicate_str    9.387s
```


BenchmarkReplicateStrNTimesWithPoolの方は、
メモリアロケーションが０になりました。

また、BenchmarkReplicateStrNTimesの１ループ当たりの
所要時間が`76~82ns`なのに対して、

BenchmarkReplicateStrNTimesWithPoolは`24~26ns`程なので、
３倍くらい速くなったことがわかります。


## sync.Poolでの高速化(nを任意の数に)

上の関数では、` make([]string, 5)`で`5`という数字が固定されてしまっていたので書き直します。


PoolのNew関数で、makeの代わりに空のSliceのポインタを返すようにしてみます。

```golang
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
	defer pool.Put(ss)
	for i := 0; i < n; i++ {
		(*ss) = append((*ss), s)
	}
	return *ss
}

```

New関数で返すのが空のSliceだと、事前にSliceのlengthを確保していないので、
ReplicateStrNTimesWithPoolはappendを使って、
都度sliceのlengthとcapを伸ばしつつ要素を追加しています。

lengthもcapも全く確保されていない状態でappendを呼ぶと、
しょっちゅう新規のメモリアロケーションが起こる（capに余裕のない状態でappendをすると、capを倍にする処理が走ります）ので要素番号を直接指定して代入するより遅くなりそうですが、
試しにやってみます。

注意点として、この方法だと、要素を後ろに追加しているので、
PoolからGetしてきた値を事前にリセットしておく必要があります。

そこで、上のコードでは`[:0]`を使って値を消しています

`[:0]`をすると、すでにappendで確保したメモリ自体は残しつつ、
値のみ消すことができます。

Benchmarkを含んだ全体のコードは以下のようになります

nに自由な数を渡せるようになりました。

```golang
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
```

このコードのBenchmarkを取ると以下のようになりました。

```
[~/go/src/github.com/ludwig125/sync-pool/replicate_str_revised] $go test -bench . -count=4
goos: linux
goarch: amd64
pkg: github.com/ludwig125/sync-pool/replicate_str_revised
BenchmarkReplicateStrNTimes-8                   13020055                77.1 ns/op            80 B/op          1 allocs/op
BenchmarkReplicateStrNTimes-8                   13794530                78.8 ns/op            80 B/op          1 allocs/op
BenchmarkReplicateStrNTimes-8                   13322482                77.2 ns/op            80 B/op          1 allocs/op
BenchmarkReplicateStrNTimes-8                   13647513                76.7 ns/op            80 B/op          1 allocs/op
BenchmarkReplicateStrNTimesWithPool-8           37328289                37.4 ns/op             0 B/op          0 allocs/op
BenchmarkReplicateStrNTimesWithPool-8           34388486                30.8 ns/op             0 B/op          0 allocs/op
BenchmarkReplicateStrNTimesWithPool-8           36526566                30.5 ns/op             0 B/op          0 allocs/op
BenchmarkReplicateStrNTimesWithPool-8           35319776                30.6 ns/op             0 B/op          0 allocs/op
PASS
ok      github.com/ludwig125/sync-pool/replicate_str_revised    9.353s
```


BenchmarkReplicateStrNTimesの１ループ当たりの
所要時間が`80ns弱`なのに対して、

BenchmarkReplicateStrNTimesWithPoolは`30nsちょっと`なので、
速度の差は２倍ちょっとにとどまりました。

最初の要素番号を指定して代入した場合と比べると速度改善の幅は
小さいですが、nの数を自由に指定できるようになりました。

ちなみにnを以下のように100にしても同じくらいの改善度（２倍くらいの差）になりました。
```
ReplicateStrNTimes("12345", 100)

ReplicateStrNTimesWithPool("12345", 100)
```


```
[~/go/src/github.com/ludwig125/sync-pool/join_str_revised] $go test -bench . -benchmem -count=4
goos: linux
goarch: amd64
pkg: github.com/ludwig125/sync-pool/join_str_revised
BenchmarkReplicateStrNTimes-8                 1964716               613 ns/op            1792 B/op          1 allocs/op
BenchmarkReplicateStrNTimes-8                 1949122               613 ns/op            1792 B/op          1 allocs/op
BenchmarkReplicateStrNTimes-8                 1925007               615 ns/op            1792 B/op          1 allocs/op
BenchmarkReplicateStrNTimes-8                 1989157               610 ns/op            1792 B/op          1 allocs/op
BenchmarkReplicateStrNTimesWithPool-8         4566765               253 ns/op               0 B/op          0 allocs/op
BenchmarkReplicateStrNTimesWithPool-8         4694205               251 ns/op               0 B/op          0 allocs/op
BenchmarkReplicateStrNTimesWithPool-8         4734534               250 ns/op               0 B/op          0 allocs/op
BenchmarkReplicateStrNTimesWithPool-8         4749656               249 ns/op               0 B/op          0 allocs/op
PASS
ok      github.com/ludwig125/sync-pool/join_str_revised 13.058s
```

# appendを使った方法でアロケーション数が0になる理由は？

上の気になる点は、appendを使う方法に変えたにもかかわらず、
メモリのアロケーション回数、アロケーションされたバイト数は0のままなことです。

appendをしているので少しは増えるかと思ったのですが、ここが0のままなのはなぜでしょう。。？

## gcflags=-m で最適化の確認

goのプログラムが変数を処理するとき、stack割り当てについてはBenchmarkはアロケーションに含めていません。heap割り当てのみが対象になります。

参考：

- https://hnakamur.github.io/blog/2018/01/30/go-heap-allocations/
- https://yoru9zine.hatenablog.com/entry/2016/08/31/055025

そのため、アロケーションが0ということは、`ReplicateStrNTimesWithPool`はheapではなく
すべてstack割り当てをしているか、どこにも割り当てしていないということになってしまいます。

確認するために、上のBenchmark実行時に`-gcflags=-m`オプションを渡してみました。
`-gcflags=-m`オプションをつけると、build時、test時に内部の変数がheap割り当てになるかなどの、最適化を確認することができます。
ちなみに、`-gcflags='-m -m'`とmを増やすとより詳細な結果が得られます。

```
[~/go/src/github.com/ludwig125/sync-pool/replicate_str_revised] $go test -bench . -count=4 -gcflags=-m
# github.com/ludwig125/sync-pool/replicate_str_revised [github.com/ludwig125/sync-pool/replicate_str_revised.test]
./replicate_str_revised_test.go:66:16: inlining call to testing.(*B).ReportAllocs
./replicate_str_revised_test.go:75:16: inlining call to testing.(*B).ReportAllocs
./replicate_str_revised_test.go:11:7: can inline glob..func1
./replicate_str_revised_test.go:16:25: leaking param: s
./replicate_str_revised_test.go:17:12: make([]string, n) escapes to heap
./replicate_str_revised_test.go:29:8: ReplicateStrNTimesWithPool ignoring self-assignment in *ss = (*ss)[:0]
./replicate_str_revised_test.go:24:33: leaking param: s
./replicate_str_revised_test.go:37:29: leaking param: t
./replicate_str_revised_test.go:48:57: leaking param: t
./replicate_str_revised_test.go:54:65: leaking param: t

（以下省略：37行目以降は TestReplicateStrNTimes部分なので見る必要ないです）
```

これを見ると、`ReplicateStrNTimes`の`make([]string, n)`ではheap割り当てが発生しています。
```
./replicate_str_revised_test.go:17:12: make([]string, n) escapes to heap
```

一方で`ReplicateStrNTimesWithPool`はそれにあたるものが出ていません。
もしstack割り当てが発生しているとしたら、`does not escape`というheapに変数を退避させなかったメッセージが出るのですが、それも出ていません。

なので、最適化の結果だけを見ると、appendにした場合のsliceについて heapにもstackにも割り当てされている様子が見えない

=> 新たな割り当てが起きていない？

という謎なことになりました。これはおかしいです。

## appendの場合のアロケーションが0となってしまう原因の推測

おそらく、appendでcapを拡張するのは毎回行うことではないからでは、という推測をしています。

capは最初0で、次にappendされるたびに1、2、4, 8, 16と倍々に増やされますが、
例えば一度capを16まで増やしたら、要素数が9から16まで追加される間はcapがその範囲内なので拡張しません。
次に要素数が17になるときにcapは32に拡張されますが、
そうなると、要素数18~32の間は同じく拡張されず、次に拡張されるのは33番目を追加するタイミングです。

もしcapが拡張されないタイミングを測定したらアロケーションは0になるはずです。

## 毎回nilにする場合のアロケーション回数の確認

そこで、試しに以下のように、`(*ss) = (*ss)[:0]`ではなく、
`(*ss) = nil`として、sliceをメモリごと初期化するようにしてみました。

```golang
func ReplicateStrNTimesWithPool(s string, n int) []string {
	ss := pool.Get().(*[]string)

	// (*ss) = (*ss)[:0]
	(*ss) = nil
	defer pool.Put(ss)
	for i := 0; i < n; i++ {
		(*ss) = append((*ss), s)
	}
	return *ss
}
```

これでBenchmarkを取ると、

`ReplicateStrNTimesWithPool("12345", 1)`のように文字列１個だけのSliceを作る時は
```
1 allocs/op
```


`ReplicateStrNTimesWithPool("12345", 2)`のように文字列２個のSliceを作る時は
```
2 allocs/op
```


`ReplicateStrNTimesWithPool("12345", 32)`は
```
6 allocs/op
```


`ReplicateStrNTimesWithPool("12345", 33)`は
```
7 allocs/op
```

となりました。

capの拡張操作は以下の順番で行われるので、32要素のSliceを作るには6回操作 `= 6 allocs/op`、
33要素のSliceには７回操作 `= 7 allocs/op`となることが確認できました。

capの推移
```
0 -> 1: 1回目
1 -> 2: 2回目
2 -> 4: 3回目
4 -> 8: 4回目
8 -> 16: 5回目
16 -> 32: 6回目
32 -> 64: 7回目
```

毎回nilする場合は想定通りのメモリアロケーションがありましたが、
最初の例で全部 0 allocなのはまだよくわかっていません。

例えば
`ReplicateStrNTimesWithPool("12345", 5)`のときは、
3回目と5回目はcap拡張によるアロケーションがカウントされないので、
「カウントしない場合もあるから0と見なそう」とBenchmarkが働いているということならわかりますが、
以下のように１回にしたら必ず`0->1`のcap拡張があるはずです。

```golang
func BenchmarkReplicateStrNTimesWithPool(b *testing.B) {
	b.ReportAllocs()
	var r []string
	for n := 0; n < b.N; n++ {
		// r = ReplicateStrNTimesWithPool("12345", 5)
		r = ReplicateStrNTimesWithPool("12345", 1)
	}
	Result = r
}
```


それでもこの結果は `0 allocs/op`になりました。

よくわからないので次に単純なサンプルで確認してみました。

## Allocsの確認（AllocsPerRunで確認）

testingパッケージにはAllocsPerRunという関数があります。

https://pkg.go.dev/testing#AllocsPerRun

これは、引数として関数と実行回数を与えると、１実行回数あたりのアロケーション数をカウントする関数です。

試しにこの関数で上と同様にappendをする関数のアロケーションを確認してみます。

*以下の関数では、stringではなくintのsliceとなっていますが、検証の過程でintにしただけで挙動に違いはありません*

```golang
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
```

この実行結果は以下のようになりました。

```
[~/go/src/github.com/ludwig125/sync-pool/check_allocs] $go test -v .
=== RUN   TestAddNum
Allocs: 0
--- PASS: TestAddNum (0.00s)
PASS
ok      github.com/ludwig125/sync-pool/check_allocs     0.002s
```

やはりBenchmarkのときと同様にアロケーション数0となりました。

ここでこのAllocsPerRunの実装を見てみます。
https://github.com/golang/go/blob/2ebe77a2fda1ee9ff6fd9a3e08933ad1ebaea039/src/testing/allocs.go#L20-L45

```golang
func AllocsPerRun(runs int, f func()) (avg float64) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))

	// Warm up the function
	f()  // <1> warm upで一度実行

	// Measure the starting statistics
	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats) // <2> runtime.MemStatsで現在のメモリ情報を取得
	mallocs := 0 - memstats.Mallocs

	// Run the function the specified number of times
	for i := 0; i < runs; i++ {
		f() // <3> runの回数だけ関数を実行
	}

	// Read the final statistics
	runtime.ReadMemStats(&memstats) // <4> 再度runtime.MemStatsでメモリ情報を取得
	mallocs += memstats.Mallocs  // <5> run回関数を実行する前のmalloc数との差分を計算

	// Average the mallocs over the runs (not counting the warm-up).
	// We are forced to return a float64 because the API is silly, but do
	// the division as integers so we can ask if AllocsPerRun()==1
	// instead of AllocsPerRun()<2.
	return float64(mallocs / uint64(runs)) // <6> １runあたりのmallocsを計算
}
```

これを見ると、以下の処理になっています。

- <1> 最初に引数として与えた関数を Warm upとして１回実行
- <2> runtime.MemStatsで現在のメモリ情報を取得
- <3> runの回数だけ関数を実行
- <4> 再度runtime.MemStatsでメモリ情報を取得
- <5> run回関数を実行する前のmalloc数との差分を計算
- <6> 最後に１runあたりのmallocsを計算

最初のwarm up関数があることで、以下のように`testing.AllocsPerRun`のRun回数として１を指定しても、
実際には２回`AddNum`関数が実行されていることが分かりました。

```golang
fmt.Println("Allocs:", int(testing.AllocsPerRun(1, func() {
	AddNum(1)
})))
```

１回だけ関数を実行したときのmalloc数を確認するために、`testing.AllocsPerRun`を参考に
`MyAllocsPerRun`関数を作ってみました。
また、関数の実行前後のmallocの差分をプリントするようにしてみたのが以下のコードです。

`AddNum`関数は同じなので省略しています。

```golang
func TestAddNum(t *testing.T) {
	fmt.Println("MyAllocs:", int(MyAllocsPerRun(1, func() {
		AddNum(1)
	})))
}

func MyAllocsPerRun(runs int, f func()) (avg float64) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))

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
```

これを実行すると以下のようになります

```
[~/go/src/github.com/ludwig125/sync-pool/check_allocs2] $go test -v . -count=1
=== RUN   TestAddNum
mallocs: 4, memstats.Mallocs(before 278 ->after 282). run: 1
MyAllocs: 4
--- PASS: TestAddNum (0.00s)
PASS
ok      github.com/ludwig125/sync-pool/check_allocs2    0.002s
```

なんといきなり `MyAllocs: 4`となりました！

`testing.AllocsPerRun` の結果が0だったということは、
warm upの関数実行がmalloc数4にあたり、
そのあとの以下の部分では新しいアロケーションは行われなかったということが考えられます。

```golang
for i := 0; i < runs; i++ {
	f()
}
```

実際に`MyAllocsPerRun`でも、最初の行で`f()`を実行すると`MyAllocs: 0`になりました。

```golang
func MyAllocsPerRun(runs int, f func()) (avg float64) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))

	f()
	＜略＞
```

```
mallocs: 0, memstats.Mallocs(before 279 ->after 279). run: 2
MyAllocs: 0
```

## Allocsの確認（直接runtime.ReadMemStatsで確認）

アロケーションが0とカウントされる原因がかなり分かってきたので、

以下では直接`AddNum`関数本体にアロケーションの確認を入れてみました。

```golang
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
	defer pool.Put(l)
	(*l) = (*l)[:0]

	(*l) = append((*l), n)

	runtime.ReadMemStats(&memstats)
	mallocs += memstats.Mallocs
	fmt.Printf("mallocs: %v, memstats.Mallocs(before %d ->after %d). IN FUNC\n", mallocs, before, memstats.Mallocs)

	return (*l)
}

func TestAddNum(t *testing.T) {
	AddNum(1)
	AddNum(1)
}
```

この実行結果は以下です。

予想通り、最初の`AddNum`関数実行時には`mallocs: 4`なのに、２回目は`mallocs: 0`となりました。

```
[~/go/src/github.com/ludwig125/sync-pool/check_allocs3] $go test -v . -count=1
=== RUN   TestAddNum
mallocs: 4, memstats.Mallocs(before 272 ->after 276). IN FUNC
mallocs: 0, memstats.Mallocs(before 279 ->after 279). IN FUNC
--- PASS: TestAddNum (0.00s)
PASS
ok      github.com/ludwig125/sync-pool/check_allocs3    0.003s
```

以下のようにさらに`AddNum`を書き換えるとようやく分かってきます。
`l := pool.Get().(*[]int)` 部分のみをmallocの計測範囲としました。

ついでに、pool内のsliceの長さlenと容量capを出力するようにします。
`fmt.Printf("*l: %v, len(*l): %v, cap(*l): %v\n", *l, len(*l), cap(*l))`

結果が分かりやすいように`TestAddNum`内で`AddNum`関数を実行するときに、１回目は`1`を２回目は`2`を引数にしました。


```golang
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
```

これの実行結果は以下の通りです。

```
mallocs: 3, memstats.Mallocs(before 278 ->after 281). IN FUNC
*l: [1], len(*l): 1, cap(*l): 1
mallocs: 0, memstats.Mallocs(before 287 ->after 287). IN FUNC
*l: [2], len(*l): 1, cap(*l): 1
```

次に、今度は`(*l) = append((*l), n)`部分のみをmallocsの確認対象として囲んでみます。

```golang
func AddNum(n int) []int {
	l := pool.Get().(*[]int)
	defer pool.Put(l)
	(*l) = (*l)[:0]

	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)
	mallocs := 0 - memstats.Mallocs
	before := memstats.Mallocs

	(*l) = append((*l), n)

	runtime.ReadMemStats(&memstats)
	mallocs += memstats.Mallocs
	fmt.Printf("mallocs: %v, memstats.Mallocs(before %d ->after %d). IN FUNC\n", mallocs, before, memstats.Mallocs)
	fmt.Printf("*l: %v, len(*l): %v, cap(*l): %v\n", *l, len(*l), cap(*l))
	return (*l)
}
```

実行結果
```
mallocs: 1, memstats.Mallocs(before 278 ->after 279). IN FUNC
*l: [1], len(*l): 1, cap(*l): 1
mallocs: 0, memstats.Mallocs(before 284 ->after 284). IN FUNC
*l: [2], len(*l): 1, cap(*l): 1
```

ここから次の結果が得られました。

- AddNum １回目: `l := pool.Get().(*[]int)`のアロケーション数は3
- AddNum １回目: `(*l) = append((*l), n)`のアロケーション数は1
- AddNum ２回目: アロケーション数は０

関数内で、`(*l) = (*l)[:0]`をしたあとappendしているので、
`*l`の容量は１回目と同じ１のまま、中身の数字だけ`1`から`2`に変わっています。

以上のことから、最初に私が疑問に思った通り、appendをするとアロケーションがされることは確認できました。

また、`testing.AllocsPerRun` の結果が0になる理由も確認できました。

`testing.AllocsPerRun` はmallocの計測前に関数を１回実行するので、それ以降の実行時のアロケーションは0になるからです。

## Allocsの確認（Benchmarkのコードを確認）

同様のことがBenchmarkにも言えそうです。

Benchmarkのアロケーションのカウントも同じように
`memstats.Mallocs`の差分を実行回数で割るという方法は変わりません。

Benchmarkでも、以下のように最初の時点のmalloc数との差分を取ってから、

```golang
runtime.ReadMemStats(&memStats)
b.netAllocs += memStats.Mallocs - b.startAllocs
```

- https://github.com/golang/go/blob/0941dbca6ae805dd7b5f7871d5811b7b7f14f77f/src/testing/benchmark.go#L123-L124
- https://github.com/golang/go/blob/0941dbca6ae805dd7b5f7871d5811b7b7f14f77f/src/testing/benchmark.go#L137-L138

実行回数で割っています
```golang
int64(r.MemAllocs) / int64(r.N)
```

- https://github.com/golang/go/blob/0941dbca6ae805dd7b5f7871d5811b7b7f14f77f/src/testing/benchmark.go#L389-L397

また、Benchmarkでは、
最初に`b.run1()`と１回だけ実行してから、そのあとメインの`r := b.doBench()`をしているようです。
- https://github.com/golang/go/blob/0941dbca6ae805dd7b5f7871d5811b7b7f14f77f/src/testing/benchmark.go#L568-L583


さらに、この`doBench`関数の先を見ていくと、Benchmarkの測定対象である`runN`関数内では、事前にGCをしていることが分かりました。
事前に`run1`でPoolに何か書き込まれても、GCで空になるので次の関数実行タイミングではまっさらの状態からのメモリ割り当てが置きそうです。

```golang
// Try to get a comparable environment for each run
// by clearing garbage from previous runs.
runtime.GC()  <- ガベージコレクションをするのでPoolの中身が空になる
b.raceErrors = -race.Errors()
b.N = n
b.parallelism = 1
b.ResetTimer()
b.StartTimer()
b.benchFunc(b) <- ここで関数を実行
```

- https://github.com/golang/go/blob/0941dbca6ae805dd7b5f7871d5811b7b7f14f77f/src/testing/benchmark.go#L186


確認するために、`ReplicateStrNTimesWithPool` 関数内でグローバル変数`cnt`をインクリメントして、Benchmark時に出力するようにしてみます。

またBenchmark関数内は１回だけ実行するように`for n := 0; n < b.N; n++ {`をやめます。

```golang
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
```

そして、Benchmarkを測定するときに、実行回数が確実に１回だけになるように
`-benchtime=1x` を指定、さらに余計な並行処理をしないように `-cpu 1`とします。

Benchmarkのオプションについては公式ドキュメントを参考にしています。

- https://pkg.go.dev/cmd/go#hdr-Testing_flags

この実行結果は以下の通りです。

```
[~/go/src/github.com/ludwig125/sync-pool/replicate_str_revised_run1] $go test -bench . -benchtime=1x -cpu 1

cnt 1
goos: linux
goarch: amd64
pkg: github.com/ludwig125/sync-pool/replicate_str_revised_run1
BenchmarkReplicateStrNTimesWithPool
cnt 2
       1             51800 ns/op             280 B/op          4 allocs/op
PASS
ok      github.com/ludwig125/sync-pool/replicate_str_revised_run1       0.004s
```

想定通り、Benchmarkの測定前に１回関数が実行されているので、`cnt 1`が最初にでます。

そのあとで、やはり１回だけ関数が実行されて`cnt 2`となり、メモリアロケーションは `4 allocs/op`となりました。
この `4 allocs/op`というのは、前述の「PoolからのGet: 3」+「append: 1」で確認した４という数字と一致します。

先に書いた通り、Benchmarkのアロケーション数は、「実際のアロケーション数／関数の実行回数」となるので、
`benchtime=5x` のように５回以上を指定すれば、`4/5` は１未満になるので、結果は`0 allocs/op`とみなされます。

以下の通りです。

```
[~/go/src/github.com/ludwig125/sync-pool/replicate_str_revised_run1] $go test -bench . -benchtime=5x -cpu 1

cnt 1
goos: linux
goarch: amd64
pkg: github.com/ludwig125/sync-pool/replicate_str_revised_run1
BenchmarkReplicateStrNTimesWithPool
cnt 2
       5             10540 ns/op              56 B/op          0 allocs/op
PASS
ok      github.com/ludwig125/sync-pool/replicate_str_revised_run1       0.004s
[~/go/src/github.com/ludwig125/sync-pool/replicate_str_revised_run1] $
```

これで、
一番最初の疑問の、`ReplicateStrNTimesWithPool`関数でappendをしているのに、アロケーション数が0になる原因がはっきりと分かりました。

Benchmarkでは実際にアロケーションがあっても、実行回数で割った時に１未満になる時は0と出力される、という単純なことでした。

この確認にずいぶんと時間がかかりましたが、Benchmark関数の挙動の理解ができて満足です。


# sync.Poolを使ったjsonデコードの例

ここまで使ってみて、sync.Poolが特に役に立つのは、
`データの入れ物を事前に用意してそこにデータを詰める` 作業なのだろうと私なりに理解しました。

そこで、他にもそういう操作があれば高速化してみたいです。

分かりやすそうなのがjsonのデコードです。

## 一般的なjsonデコード

例えば文字列を構造体にDecodeするコードは単純に書くと以下になります。

```golang
type JsonData struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Items []string `json:"items"`
}

func DecodeJSON(in string) (JsonData, error) {
	var res JsonData
	if err := json.Unmarshal([]byte(in), &res); err != nil {
		return JsonData{}, err
	}
	return res, nil
}
```

この例では、`res`という`JsonData`型の入れ物を用意しておいて、そこにデコード（Unmarshal）した結果を入れています。

ちなみに、Webリクエストの結果のようにStreamのデータをdecodeしたい場合は一旦バッファを確保してからデコードするために、
以下のような`json.NewDecoder.Decode` を使った方法があります。


```golang

func DecodeJSONStream(in io.Reader) (JsonData, error) {
	var res JsonData
	if err := json.NewDecoder(in).Decode(&res); err != nil {
		return JsonData{}, err
	}
	return res, nil
}
```

#### 参考：GoでJSONのデコードをするときの、UnmarshalとNewDecoder.Decodeの違いについて

- https://stackoverflow.com/questions/21197239/decoding-json-using-json-unmarshal-vs-json-newdecoder-decode

以下のように使い分ければ良いです

- Unmarshalは、ファイルなどから読み込んだデータをデコードするとき
- NewDecoder.Decodeは、httpでのGetのように終わりが見えていないデータをデコードするとき

`NewDecoder.Decode`について公式のドキュメントには以下のように書いてあります。

- https://blog.golang.org/json#TOC_7.

> such as reading and writing to HTTP connections, WebSockets, or files.

#### UnmarshalとNewDecoder.Decodeの処理の違い

通常のUnmarshalはunmarshalメソッドをほぼ直接呼んでいるのに対して、

- https://github.com/golang/go/blob/ab4085ce84f8378b4ec2dfdbbc44c98cb92debe5/src/encoding/json/decode.go#L96-L108

Decodeの方は、一旦バッファを確保してからunmarshalメソッドを呼んでいます。

- https://github.com/golang/go/blob/296ddf2a936a30866303a64d49bc0e3e034730a8/src/encoding/json/stream.go#L31-L79

ということで、

**最終的にどちらもunmarshal関数を呼んでいますが、すでにメモリに置かれたデータをデコードする場合はバッファを確保する分だけNewDecoder.Decodeの方が遅くなりそうです**

## sync.Poolを使ったjsonデコード

上のjsonデコードを、sync.Poolを使って書き直した関数を加えて、Benchmarkを取ってみます。

以下ではデコードした結果を入れるPoolとして`decRespPool`を用意しました。

また、それぞれのデコード結果が同じであることは、`TestDecodeJSON`で確認しました。

（エンコード用の関数`EncodeJSON`はおまけです）

```golang
package main

import (
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type JsonData struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Items []string `json:"items"`
}

func EncodeJSON(in JsonData) (string, error) {
	res, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func DecodeJSON(in string) (JsonData, error) {
	var res JsonData
	if err := json.Unmarshal([]byte(in), &res); err != nil {
		return JsonData{}, err
	}
	return res, nil
}

func DecodeJSONStream(in io.Reader) (JsonData, error) {
	var res JsonData
	if err := json.NewDecoder(in).Decode(&res); err != nil {
		return JsonData{}, err
	}
	return res, nil
}

var decRespPool = &sync.Pool{
	New: func() interface{} {
		return &JsonData{}
	},
}

func DecodeJSONWithPool(in string) (JsonData, error) {
	res := decRespPool.Get().(*JsonData)
	defer decRespPool.Put(res)

	if err := json.Unmarshal([]byte(in), &res); err != nil {
		return JsonData{}, err
	}
	return *res, nil
}

func DecodeJSONStreamWithPool(in io.Reader) (JsonData, error) {
	res := decRespPool.Get().(*JsonData)
	defer decRespPool.Put(res)

	if err := json.NewDecoder(in).Decode(&res); err != nil {
		return JsonData{}, err
	}
	return *res, nil
}

func TestDecodeJSON(t *testing.T) {
	t.Run("EncodeJSON", func(t *testing.T) {
		data := JsonData{
			ID:    1,
			Name:  "Jack",
			Items: []string{"knife", "shield", "herbs"},
		}
		got, err := EncodeJSON(data)
		if err != nil {
			t.Fatal(err)
		}
		want := `{"id":1,"name":"Jack","items":["knife","shield","herbs"]}`
		if got != want {
			t.Errorf("got: %s, want: %s", got, want)
		}
	})

	encodedData := `{"id":1,"name":"Jack","items":["knife","shield","herbs"]}`
	t.Run("DecodeJSON", func(t *testing.T) {
		got, err := DecodeJSON(encodedData)
		if err != nil {
			t.Fatal(err)
		}
		want := JsonData{
			ID:    1,
			Name:  "Jack",
			Items: []string{"knife", "shield", "herbs"},
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("got: %v,want: %v, diff: %s", got, want, diff)
		}
	})
	t.Run("DecodeJSONStream", func(t *testing.T) {
		data := strings.NewReader(encodedData)
		got, err := DecodeJSONStream(data)
		if err != nil {
			t.Fatal(err)
		}
		want := JsonData{
			ID:    1,
			Name:  "Jack",
			Items: []string{"knife", "shield", "herbs"},
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("got: %v,want: %v, diff: %s", got, want, diff)
		}
	})

	// 以下は上の関数のPool対応版
	t.Run("DecodeJSONWithPool", func(t *testing.T) {
		got, err := DecodeJSONWithPool(encodedData)
		if err != nil {
			t.Fatal(err)
		}
		want := JsonData{
			ID:    1,
			Name:  "Jack",
			Items: []string{"knife", "shield", "herbs"},
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("got: %v,want: %v, diff: %s", got, want, diff)
		}
	})
	t.Run("DecodeJSONStreamWithPool", func(t *testing.T) {
		data := strings.NewReader(encodedData)
		got, err := DecodeJSONStreamWithPool(data)
		if err != nil {
			t.Fatal(err)
		}
		want := JsonData{
			ID:    1,
			Name:  "Jack",
			Items: []string{"knife", "shield", "herbs"},
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("got: %v,want: %v, diff: %s", got, want, diff)
		}
	})
}

var Result JsonData

var encodedData = `{"id":1,"name":"Jack","items":["knife","shield","herbs"]}`

func BenchmarkDecodeJSON(b *testing.B) {
	b.ReportAllocs()
	var r JsonData
	for n := 0; n < b.N; n++ {
		res, _ := DecodeJSON(encodedData)
		r = res
	}
	Result = r
}

func BenchmarkDecodeJSONWithPool(b *testing.B) {
	b.ReportAllocs()
	var r JsonData
	for n := 0; n < b.N; n++ {
		res, _ := DecodeJSONWithPool(encodedData)
		r = res
	}
	Result = r
}

func BenchmarkDecodeJSONStream(b *testing.B) {
	b.ReportAllocs()
	var r JsonData
	for n := 0; n < b.N; n++ {
		data := strings.NewReader(encodedData)
		res, _ := DecodeJSONStream(data)
		r = res
	}
	Result = r
}

func BenchmarkDecodeJSONStreamWithPool(b *testing.B) {
	b.ReportAllocs()
	var r JsonData
	for n := 0; n < b.N; n++ {
		data := strings.NewReader(encodedData)
		res, _ := DecodeJSONStreamWithPool(data)
		r = res
	}
	Result = r
}
```

４回ずつBenchmarkを実行した結果は以下の通りです。

```
[~/go/src/github.com/ludwig125/sync-pool/json] $go test -bench . -count=4
goos: linux
goarch: amd64
pkg: github.com/ludwig125/sync-pool/json
BenchmarkDecodeJSON-8                     634494              1900 ns/op             448 B/op         12 allocs/op
BenchmarkDecodeJSON-8                     555051              1906 ns/op             448 B/op         12 allocs/op
BenchmarkDecodeJSON-8                     570580              1939 ns/op             448 B/op         12 allocs/op
BenchmarkDecodeJSON-8                     532183              1986 ns/op             448 B/op         12 allocs/op
BenchmarkDecodeJSONWithPool-8             690313              1798 ns/op             312 B/op         10 allocs/op
BenchmarkDecodeJSONWithPool-8             712689              1736 ns/op             312 B/op         10 allocs/op
BenchmarkDecodeJSONWithPool-8             626592              1945 ns/op             312 B/op         10 allocs/op
BenchmarkDecodeJSONWithPool-8             632752              1680 ns/op             312 B/op         10 allocs/op
BenchmarkDecodeJSONStream-8               448029              2236 ns/op            1136 B/op         15 allocs/op
BenchmarkDecodeJSONStream-8               518344              2251 ns/op            1136 B/op         15 allocs/op
BenchmarkDecodeJSONStream-8               490837              2419 ns/op            1136 B/op         15 allocs/op
BenchmarkDecodeJSONStream-8               473403              2317 ns/op            1136 B/op         15 allocs/op
BenchmarkDecodeJSONStreamWithPool-8       491476              2225 ns/op            1000 B/op         13 allocs/op
BenchmarkDecodeJSONStreamWithPool-8       555782              2056 ns/op            1000 B/op         13 allocs/op
BenchmarkDecodeJSONStreamWithPool-8       509203              2063 ns/op            1000 B/op         13 allocs/op
BenchmarkDecodeJSONStreamWithPool-8       541905              2097 ns/op            1000 B/op         13 allocs/op
PASS
ok      github.com/ludwig125/sync-pool/json     22.339s
[~/go/src/github.com/ludwig125/sync-pool/json] $
```

前述の通り、Streamを扱う`NewDecoder.Decode`は最初にバッファを確保する分、単純な`Unmarshal`に比べて時間もメモリアロケーションも余計にかかるようです。

肝心のsync.Poolを使った場合の改善度合いですが、

`BenchmarkDecodeJSON`と`BenchmarkDecodeJSONWithPool`、`BenchmarkDecodeJSONStream`と`BenchmarkDecodeJSONStreamWithPool`をそれぞれ比較して、そこまでの改善はありませんでした。
