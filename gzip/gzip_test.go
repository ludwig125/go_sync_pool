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

func Gunzip(data io.Reader) ([]byte, error) {
	gr, err := gzip.NewReader(data)
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

// 上のGunzipと比べてbytes.NewBuffer(data)の分だけアロケーションが余計にかかる
func GunzipByteSlice(data []byte) ([]byte, error) {
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

// 上のGunzipのio.Copyの代わりにioutil.ReadAllを使ってみた場合
func GunzipIoutilReadAll(data io.Reader) ([]byte, error) {
	gr, err := gzip.NewReader(data)
	if err != nil {
		return nil, fmt.Errorf("failed to gzip.NewReader: %v", err)
	}
	var buf bytes.Buffer
	d, err := ioutil.ReadAll(gr)
	if err != nil {
		log.Fatalf("failed to ReadAll: %v", err)
	}
	buf.Write(d)
	if err := gr.Close(); err != nil {
		return nil, fmt.Errorf("failed to Close gzip Reader: %v", err)
	}

	return buf.Bytes(), nil
}

// 上のGunzipのio.Copyの代わりにbytes.Buffer.ReadFromを使ってみた場合
func GunzipBufferReadFrom(data io.Reader) ([]byte, error) {
	gr, err := gzip.NewReader(data)
	if err != nil {
		return nil, fmt.Errorf("failed to gzip.NewReader: %v", err)
	}
	var buf bytes.Buffer
	if _, err = buf.ReadFrom(gr); err != nil {
		return nil, err
	}
	if err := gr.Close(); err != nil {
		return nil, fmt.Errorf("failed to Close gzip Reader: %v", err)
	}

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
	err error
}

var gzipReaderPool = sync.Pool{
	New: func() interface{} {
		var buf bytes.Buffer
		// 空のbufをgzip.NewReaderで読み込むと EOF を出すので、
		// gzip header情報を書き込む
		zw := gzip.NewWriter(&buf)
		if err := zw.Close(); err != nil {
			return &gzipReader{
				err: err,
			}
		}

		r, err := gzip.NewReader(&buf)
		if err != nil {
			return &gzipReader{
				err: err,
			}
		}
		return &gzipReader{
			r:   r,
			buf: &buf,
		}
	},
}

func GunzipWithGzipReaderPool(data io.Reader) ([]byte, error) {
	gr := gzipReaderPool.Get().(*gzipReader)
	if gr.err != nil {
		return nil, fmt.Errorf("failed to Get gzipReaderPool: %v", gr.err)
	}
	defer gzipReaderPool.Put(gr)
	defer gr.r.Close()
	gr.buf.Reset()
	if err := gr.r.Reset(data); err != nil {
		return nil, err
	}

	if _, err := io.Copy(gr.buf, gr.r); err != nil {
		return nil, fmt.Errorf("failed to io.Copy: %v", err)
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
					return &gzipReader{
						err: err,
					}
				}

				r, err := gzip.NewReader(&buf)
				if err != nil {
					return &gzipReader{
						err: err,
					}
				}
				return &gzipReader{
					r:   r,
					buf: &buf,
				}
			},
		},
	}
}

func (g *GunzipperWithSyncPool) Gunzip(data io.Reader) ([]byte, error) {
	gr := g.GzipReaderPool.Get().(*gzipReader)
	if gr.err != nil {
		return nil, fmt.Errorf("failed to Get gzipReaderPool: %v", gr.err)
	}
	defer g.GzipReaderPool.Put(gr)
	defer gr.r.Close()
	gr.buf.Reset()
	if err := gr.r.Reset(data); err != nil {
		return nil, err
	}

	if _, err := io.Copy(gr.buf, gr.r); err != nil {
		return nil, fmt.Errorf("failed to io.Copy: %v", err)
	}

	return gr.buf.Bytes(), nil
}

func TestGzip(t *testing.T) {
	data := `https://pkg.go.dev/compress/gzip
Documentation
Overview
Package gzip implements reading and writing of gzip format compressed files, as specified in RFC 1952.`

	t.Run("Gzip_and_Gunzip", func(t *testing.T) {
		res, err := Gzip([]byte(data))
		if err != nil {
			t.Fatal(err)
		}

		got, err := Gunzip(bytes.NewBuffer(res))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != data {
			t.Errorf("got: %s, want: %s", string(got), data)
		}

		got2, err := GunzipByteSlice(res)
		if err != nil {
			t.Fatal(err)
		}
		if string(got2) != data {
			t.Errorf("got2: %s, want: %s", string(got2), data)
		}

		got3, err := GunzipIoutilReadAll(bytes.NewBuffer(res))
		if err != nil {
			t.Fatal(err)
		}
		if string(got3) != data {
			t.Errorf("got3: %s, want: %s", string(got3), data)
		}

		got4, err := GunzipBufferReadFrom(bytes.NewBuffer(res))
		if err != nil {
			t.Fatal(err)
		}
		if string(got4) != data {
			t.Errorf("got4: %s, want: %s", string(got4), data)
		}
	})

	// Poolを正しく使わないと前にPutした値をGetで取ってきてしまうミスがあり得る
	// そのため、２回実行しても同じ結果であることを確認している
	for i := 0; i < 3; i++ {
		t.Run("GzipWithGzipWriterPool_GunzipWithGzipReaderPool", func(t *testing.T) {
			res, err := GzipWithGzipWriterPool([]byte(data))
			if err != nil {
				t.Fatal(err)
			}

			res2, err := GunzipWithGzipReaderPool(bytes.NewBuffer(res))
			if err != nil {
				t.Fatal(err)
			}

			if string(res2) != data {
				t.Errorf("got: %s, want: %s", string(res2), data)
			}
		})

		t.Run("GzipperWithSyncPool_GunzipperWithSyncPool", func(t *testing.T) {
			g := NewGzipperWithSyncPool()
			res, err := g.Gzip([]byte(data))
			if err != nil {
				t.Fatal(err)
			}

			gu := NewGunzipperWithSyncPool()
			res2, err := gu.Gunzip(bytes.NewBuffer(res))
			if err != nil {
				t.Fatal(err)
			}

			if string(res2) != data {
				t.Errorf("got: %s, want: %s", string(res2), data)
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

	gzippedData, _    = Gzip([]byte(data))
	gzippedDataStream = bytes.NewBuffer(gzippedData)
)

func BenchmarkGzip(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = Gzip([]byte(data))
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

func BenchmarkGunzip(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = Gunzip(gzippedDataStream)
	}
	Result = r
}

func BenchmarkGunzipByteSlice(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = GunzipByteSlice(gzippedData)
	}
	Result = r
}

func BenchmarkGunzipIoutilReadAll(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = GunzipIoutilReadAll(gzippedDataStream)
	}
	Result = r
}

func BenchmarkGunzipBufferReadFrom(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = GunzipBufferReadFrom(gzippedDataStream)
	}
	Result = r
}

func BenchmarkGunzipWithGzipReaderPool(b *testing.B) {
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = GunzipWithGzipReaderPool(gzippedDataStream)
	}
	Result = r
}

func BenchmarkGunzipperWithSyncPool(b *testing.B) {
	g := NewGunzipperWithSyncPool()
	b.ResetTimer()
	b.ReportAllocs()
	var r []byte
	for n := 0; n < b.N; n++ {
		r, _ = g.Gunzip(gzippedDataStream)
	}
	Result = r
}
