package main

import (
	"bytes"
	"compress/gzip"
	"log"
	"os"
)

func main() {
	file, err := os.Create("file.gz")
	if err != nil {
		log.Fatal(err)
	}

	// fileにgzip Headerを書き込み(FlushもCloseもしていないと書き込みは完了していない)
	zw := gzip.NewWriter(file)
	if err := zw.Close(); err != nil {
		log.Fatal(err)
	}

	// gzip.Headerなしのfileをgzip.NewReaderで読み込もうとするとEOFが返る
	if _, err := gzip.NewReader(file); err != nil {
		log.Printf("gzip.NewReader file: %v", err)
	}
	file.Close() // CloseするとHeader情報が書き込まれて閉じられる

	// 同じファイルをfile2として開きなおす
	file2, err := os.Open("file.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer file2.Close()
	// fileにはHeader情報があるので、gzip.NewReaderでエラーは発生しない
	if _, err := gzip.NewReader(file2); err != nil {
		log.Fatalf("gzip.NewReader file2: %v", err)
	}

	var buf bytes.Buffer
	if _, err := gzip.NewReader(&buf); err != nil {
		log.Printf("gzip.NewReader bytes.Buffer: %v", err)
	}
}
