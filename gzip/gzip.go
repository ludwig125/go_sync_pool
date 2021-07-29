package main

import (
	"compress/gzip"
	"log"
	"os"
)

func main() {
	file, err := os.Create("empty.gz")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	zw := gzip.NewWriter(file)
	if err := zw.Close(); err != nil {
		log.Fatal(err)
	}
}
