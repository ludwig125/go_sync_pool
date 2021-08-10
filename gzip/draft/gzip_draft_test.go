package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"sync"
	"testing"
)

func Gunzip3(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to gzip.NewReader: %v", err)
	}

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(gr); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var pool = &sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func GzipWithBytesBufferPool(data []byte) ([]byte, error) {
	buf := pool.Get().(*bytes.Buffer)
	defer pool.Put(buf)
	buf.Reset()

	gz := gzip.NewWriter(buf)
	if _, err := gz.Write(data); err != nil {
		return nil, fmt.Errorf("failed to gzip Write: %v", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to gzip Close: %v", err)
	}

	return buf.Bytes(), nil
}

func GunzipWithBytesBufferPool(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to gzip.NewReader: %v", err)
	}
	defer gr.Close()

	buf := pool.Get().(*bytes.Buffer)
	defer pool.Put(buf)
	buf.Reset()

	d, err := ioutil.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("failed to ReadAll: %v", err)
	}
	buf.Write(d)

	return buf.Bytes(), nil
}

// これうまくいかない
// failed to io.Copy: gzip: invalid checksum が出る
// func GunzipWithBytesBufferPool2(data []byte) ([]byte, error) {
// 	gr, err := gzip.NewReader(bytes.NewBuffer(data))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to gzip.NewReader: %v", err)
// 	}
// 	// defer gr.Close()

// 	buf := pool.Get().(*bytes.Buffer)
// 	defer pool.Put(buf)
// 	buf.Reset()

// 	if _, err := io.Copy(buf, gr); err != nil {
// 		return nil, fmt.Errorf("failed to io.Copy: %v", err)
// 	}
// 	if err := gr.Close(); err != nil {
// 		return nil, fmt.Errorf("failed to Close gzip Reader: %v", err)
// 	}

// 	return buf.Bytes(), nil
// }

func (g *GunzipperWithSyncPool) Gunzip2(data []byte) ([]byte, error) {
	gr := g.GzipReaderPool.Get().(*gzipReader)
	defer g.GzipReaderPool.Put(gr)
	defer gr.r.Close()
	gr.buf.Reset()
	if err := gr.r.Reset(bytes.NewBuffer(data)); err != nil {
		return nil, err
	}

	d, err := ioutil.ReadAll(gr.r)
	if err != nil {
		return nil, fmt.Errorf("failed to ReadAll: %v", err)
	}
	if _, err := gr.buf.Write(d); err != nil {
		return nil, err
	}

	return gr.buf.Bytes(), nil
}

func TestGzipDraft(t *testing.T) {
	data := `https://pkg.go.dev/compress/gzip
Documentation
Overview
Package gzip implements reading and writing of gzip format compressed files, as specified in RFC 1952.`

	// Poolを正しく使わないと前にPutした値をGetで取ってきてしまうミスがあり得る
	// そのため、２回実行しても同じ結果であることを確認している
	for i := 0; i < 3; i++ {
		t.Run("Gzip_and_Gunzip", func(t *testing.T) {
			res, err := Gzip([]byte(data))
			if err != nil {
				t.Fatal(err)
			}

			got3, err := Gunzip3(res)
			if err != nil {
				t.Fatal(err)
			}
			if string(got3) != data {
				t.Errorf("got3: %s, want: %s", string(got3), data)
			}
		})

		t.Run("GzipWithBytesBufferPool_GunzipWithGzipReaderPool", func(t *testing.T) {
			res, err := GzipWithBytesBufferPool([]byte(data))
			if err != nil {
				t.Fatal(err)
			}
			got, err := GunzipWithBytesBufferPool(res)
			if err != nil {
				t.Fatal(err)
			}

			if string(got) != data {
				t.Errorf("got: %s, want: %s", string(got), data)
			}
		})
	}
}

func BenchmarkGzipWithBytesBufferPool(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = GzipWithBytesBufferPool([]byte(data))
	}
	Result = r
}

func BenchmarkGunzipWithBytesBufferPool(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = GunzipWithBytesBufferPool(gzippedData)
	}
	Result = r
}
