package main

import (
	"bytes"
	"fmt"
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
