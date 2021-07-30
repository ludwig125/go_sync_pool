package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sync"
	"testing"
)

func Gzip(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gw := gzip.NewWriter(&b)
	if _, err := gw.Write(data); err != nil {
		return nil, fmt.Errorf("failed to gzip Write: %v", err)
	}
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("failed to Close gzip Writer: %v", err)
	}

	return b.Bytes(), nil
}

func Gunzip(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to gzip.NewReader: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, gr); err != nil {
		return nil, fmt.Errorf("failed to io.Copy: %v", err)
	}
	if err := gr.Close(); err != nil {
		return nil, fmt.Errorf("failed to Close gzip Reader: %v", err)
	}

	return buf.Bytes(), nil
}

func Gunzip2(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to gzip.NewReader: %v", err)
	}

	data, err = ioutil.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("failed to ReadAll: %v", err)
	}
	buf := &bytes.Buffer{}
	buf.Write(data)
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

	data, err = ioutil.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("failed to ReadAll: %v", err)
	}
	buf.Write(data)

	return buf.Bytes(), nil
}

type gzipWriter struct {
	w   *gzip.Writer
	buf *bytes.Buffer
}

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		buf := &bytes.Buffer{}
		w := gzip.NewWriter(buf)
		return &gzipWriter{
			w:   w,
			buf: buf,
		}
	},
}

func GzipWithGzipWriterPool(data []byte) ([]byte, error) {
	gw := gzipWriterPool.Get().(*gzipWriter)
	defer gzipWriterPool.Put(gw)
	gw.buf.Reset()
	gw.w.Reset(gw.buf)

	if _, err := gw.w.Write(data); err != nil {
		return nil, fmt.Errorf("failed to gzip Write: %v", err)
	}
	if err := gw.w.Close(); err != nil {
		return nil, fmt.Errorf("failed to gzip Close: %v", err)
	}

	return gw.buf.Bytes(), nil
}

type gzipReader struct {
	r   *gzip.Reader
	buf *bytes.Buffer
}

var gzipReaderPool = sync.Pool{
	New: func() interface{} {
		var buf bytes.Buffer
		// 空のbufをgzip.NewReaderで読み込むと unexpected EOF を出すので、
		// gzip header情報を書き込む
		zw := gzip.NewWriter(&buf)
		if err := zw.Close(); err != nil {
			log.Println(err)
		}

		r, err := gzip.NewReader(&buf)
		if err != nil {
			log.Println(err)
		}
		return &gzipReader{
			r:   r,
			buf: &buf,
		}
	},
}

func GunzipWithGzipReaderPool(data []byte) ([]byte, error) {
	gr := gzipReaderPool.Get().(*gzipReader)
	defer gzipReaderPool.Put(gr)
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

type GzipperWithSyncPool struct {
	GzipWriterPool *sync.Pool
}

func NewGzipperWithSyncPool() *GzipperWithSyncPool {
	return &GzipperWithSyncPool{
		GzipWriterPool: &sync.Pool{
			New: func() interface{} {
				buf := &bytes.Buffer{}
				w := gzip.NewWriter(buf)
				return &gzipWriter{
					w:   w,
					buf: buf,
				}
			},
		},
	}
}

func (g *GzipperWithSyncPool) Gzip(data []byte) ([]byte, error) {
	gw := g.GzipWriterPool.Get().(*gzipWriter)
	defer g.GzipWriterPool.Put(gw)
	gw.buf.Reset()
	gw.w.Reset(gw.buf)

	if _, err := gw.w.Write(data); err != nil {
		return nil, fmt.Errorf("failed to gzip Write: %v", err)
	}
	if err := gw.w.Close(); err != nil {
		return nil, fmt.Errorf("failed to gzip Close: %v", err)
	}

	return gw.buf.Bytes(), nil
}

type GunzipperWithSyncPool struct {
	GzipReaderPool *sync.Pool
}

func NewGunzipperWithSyncPool() *GunzipperWithSyncPool {
	return &GunzipperWithSyncPool{
		GzipReaderPool: &sync.Pool{
			New: func() interface{} {
				var buf bytes.Buffer
				zw := gzip.NewWriter(&buf)
				if err := zw.Close(); err != nil {
					log.Println(err)
				}

				r, err := gzip.NewReader(&buf)
				if err != nil {
					log.Println(err)
				}
				return &gzipReader{
					r:   r,
					buf: &buf,
				}
			},
		},
	}
}

func (g *GunzipperWithSyncPool) Gunzip(data []byte) ([]byte, error) {
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

func TestGzip(t *testing.T) {
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
			got, err := Gunzip(res)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != data {
				t.Errorf("got: %s, want: %s", string(got), data)
			}

			got2, err := Gunzip2(res)
			if err != nil {
				t.Fatal(err)
			}
			if string(got2) != data {
				t.Errorf("got2: %s, want: %s", string(got2), data)
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

		t.Run("GzipWithGzipWriterPool_GunzipWithGzipReaderPool", func(t *testing.T) {
			res, err := GzipWithGzipWriterPool([]byte(data))
			if err != nil {
				t.Fatal(err)
			}

			got, err := GunzipWithGzipReaderPool(res)
			if err != nil {
				t.Fatal(err)
			}

			if string(got) != data {
				t.Errorf("got: %s, want: %s", string(got), data)
			}
		})

		t.Run("GzipperWithSyncPool_GunzipperWithSyncPool", func(t *testing.T) {
			g := NewGzipperWithSyncPool()
			res, err := g.Gzip([]byte(data))
			if err != nil {
				t.Fatal(err)
			}

			gu := NewGunzipperWithSyncPool()
			got, err := gu.Gunzip(res)
			if err != nil {
				t.Fatal(err)
			}

			if string(got) != data {
				t.Errorf("got: %s, want: %s", string(got), data)
			}
		})
	}
}

var (
	Result []byte
	data   = `https://pkg.go.dev/compress/gzip
	Documentation
	Overview
	Package gzip implements reading and writing of gzip format compressed files, as specified in RFC 1952.`

	gzippedData, _ = Gzip([]byte(data))
)

func BenchmarkGzip(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = Gzip([]byte(data))
	}
	Result = r
}

func BenchmarkGzipWithBytesBufferPool(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = GzipWithBytesBufferPool([]byte(data))
	}
	Result = r
}

func BenchmarkGzipWithGzipWriterPool(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = GzipWithGzipWriterPool([]byte(data))
	}
	Result = r
}

func BenchmarkGunzip(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = Gunzip(gzippedData)
	}
	Result = r
}

func BenchmarkGunzip2(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = Gunzip2(gzippedData)
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

func BenchmarkGunzipWithGzipReaderPool(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = GunzipWithGzipReaderPool(gzippedData)
	}
	Result = r
}

func BenchmarkGzipperWithSyncPool(b *testing.B) {
	g := NewGzipperWithSyncPool()
	b.ResetTimer()
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = g.Gzip([]byte(data))
	}
	Result = r
}

func BenchmarkGunzipperWithSyncPool(b *testing.B) {
	g := NewGunzipperWithSyncPool()
	b.ResetTimer()
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = g.Gunzip(gzippedData)
	}
	Result = r
}
