package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
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

// func GunzipWithGzipReaderPool(data []byte) ([]byte, error) {
// 	gr, err := gzip.NewReader(bytes.NewBuffer(data))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to gzip.NewReader: %v", err)
// 	}

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

// func GunzipWithGzipReaderPool2(data []byte) ([]byte, error) {
// 	gr, err := gzip.NewReader(bytes.NewBuffer(data))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to gzip.NewReader: %v", err)
// 	}

// 	buf := pool.Get().(*bytes.Buffer)
// 	defer pool.Put(buf)
// 	buf.Reset()

// 	_, err = buf.ReadFrom(gr)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to ReadFrom: %v", err)
// 	}

// 	return buf.Bytes(), nil
// }

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
		buf := new(bytes.Buffer)
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

// type gzipReader struct {
// 	r   *gzip.Reader
// 	buf *bytes.Buffer
// }

// var gzipReaderPool = sync.Pool{
// 	New: func() interface{} {
// 		buf := new(bytes.Buffer)
// 		r, _ := gzip.NewReader(buf)
// 		return &gzipReader{
// 			r:   r,
// 			buf: buf,
// 		}
// 	},
// }

type gzipReader struct {
	r   *gzip.Reader
	buf *bytes.Buffer
}

var gzipReaderPool = sync.Pool{
	New: func() interface{} {
		buf := new(bytes.Buffer)
		r, _ := gzip.NewReader(buf)
		return &gzipReader{
			r:   r,
			buf: buf,
		}
	},
}

// func GunzipWithGzipReaderPool(data []byte) ([]byte, error) {
// 	gr := gzipReaderPool.Get().(*gzipReader)
// 	defer gzipReaderPool.Put(gr)
// 	// gr.buf.Reset()
// 	// if err := gr.r.Reset(gr.buf); err != nil {
// 	// 	return nil, err
// 	// }

// 	gr.buf = bytes.NewBuffer(data)

// 	data, err := ioutil.ReadAll(gr.buf)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to ReadAll: %v", err)
// 	}
// 	gr.buf.Write(data)

// 	return gr.buf.Bytes(), nil
// }

// func GunzipWithGzipReaderPool(data []byte) ([]byte, error) {
// 	// gr := gzipReaderPool.Get().(*gzipReader)
// 	// // defer gr.r.Close()
// 	// defer gzipReaderPool.Put(gr)
// 	// // gr.buf.Reset()
// 	// if err := gr.r.Reset(bytes.NewReader(data)); err != nil {
// 	// 	return nil, err
// 	// }

// 	var gr *gzipReader
// 	if r := gzipReaderPool.Get(); r != nil {
// 		// fmt.Printf("kokoooooooooooooooooooo: %#v\n", r)
// 		gr = r.(*gzipReader)
// 		// defer gr.r.Close()
// 		gr.buf.Reset()
// 		tmp, _ := gzip.NewReader(bytes.NewBuffer(data))
// 		gr.r = tmp
// 		// if err := gr.r.Reset(tmp); err != nil {
// 		// 	return nil, err
// 		// }
// 	} else {
// 		fmt.Printf("!kokoooooooooooooooooooo: %#v\n", r)
// 		var err error
// 		if gr.r, err = gzip.NewReader(bytes.NewBuffer(data)); err != nil {
// 			return nil, err
// 		}
// 	}
// 	defer gr.r.Close()

// 	defer gzipReaderPool.Put(gr)

// 	// if err := gr.r.Reset(gr.buf); err != nil {
// 	// 	return nil, err
// 	// }

// 	// gr.buf = bytes.NewBuffer(data)

// 	// gr, err := gzip.NewReader(bytes.NewBuffer(data))
// 	// if err != nil {
// 	// 	return nil, fmt.Errorf("failed to gzip.NewReader: %v", err)
// 	// }
// 	// defer gr.Close()

// 	// buf := pool.Get().(*bytes.Buffer)
// 	// defer pool.Put(buf)
// 	// buf.Reset()

// 	d, err := ioutil.ReadAll(gr.r)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to ReadAll: %v", err)
// 	}
// 	if _, err := gr.buf.Write(d); err != nil {
// 		return nil, err
// 	}

// 	return gr.buf.Bytes(), nil
// }

func GunzipWithGzipReaderPool(data []byte) ([]byte, error) {
	b := bytes.NewBuffer(data)

	gr := gzipReaderPool.Get().(*gzipReader)
	// defer gr.r.Close()
	gr.buf.Reset()
	// tmp, _ := gzip.NewReader(bytes.NewBuffer(data))
	// gr.r = tmp
	if gr.r != nil {
		// fmt.Println("1")
		// if err := gr.r.Reset(bytes.NewBuffer(data)); err != nil {
		if err := gr.r.Reset(b); err != nil {
			return nil, err
		}
	} else {
		// fmt.Println("2")
		// tmp, _ := gzip.NewReader(bytes.NewBuffer(data))
		tmp, _ := gzip.NewReader(b)
		gr.r = tmp
	}
	defer gr.r.Close()

	defer gzipReaderPool.Put(gr)

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

		t.Run("GzipWithBytesBufferPool_and_GunzipWithGzipReaderPool", func(t *testing.T) {
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
				// t.Log(string(got))
			}
		})

		t.Run("GzipWithGzipWriterPool_and_GunzipWithGzipReaderPool", func(t *testing.T) {
			res, err := GzipWithGzipWriterPool([]byte(data))
			if err != nil {
				t.Fatal(err)
			}

			got, err := GunzipWithGzipReaderPool(res)
			// got, err := GunzipWithBytesBufferPool(res)
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
