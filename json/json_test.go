package main

import (
	"bytes"
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

func EncodeJSONStream(in JsonData) (string, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(in); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

var encRespPool = &sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func EncodeJSONStreamWithPool(in JsonData) (string, error) {
	buf := encRespPool.Get().(*bytes.Buffer)
	defer encRespPool.Put(buf)

	buf.Reset() // 前のデータが残ったままなのでresetする
	if err := json.NewEncoder(buf).Encode(in); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
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

func TestEncodeJSON(t *testing.T) {
	data := JsonData{
		ID:    1,
		Name:  "Jack",
		Items: []string{"knife", "shield", "herbs"},
	}
	want := `{"id":1,"name":"Jack","items":["knife","shield","herbs"]}`

	// Poolを正しく使わないと前にPutした値をGetで取ってきてしまうミスがあり得る
	// そのため、２回実行しても同じ結果であることを確認している
	for i := 0; i < 2; i++ {
		t.Run("EncodeJSON", func(t *testing.T) {
			got, err := EncodeJSON(data)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
		t.Run("EncodeJSONStream", func(t *testing.T) {
			got, err := EncodeJSONStream(data)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
		t.Run("EncodeJSONStreamWithPool", func(t *testing.T) {
			got, err := EncodeJSONStreamWithPool(data)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	encodedData := `{"id":1,"name":"Jack","items":["knife","shield","herbs"]}`
	want := JsonData{
		ID:    1,
		Name:  "Jack",
		Items: []string{"knife", "shield", "herbs"},
	}

	for i := 0; i < 2; i++ {
		t.Run("DecodeJSON", func(t *testing.T) {
			got, err := DecodeJSON(encodedData)
			if err != nil {
				t.Fatal(err)
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
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("got: %v,want: %v, diff: %s", got, want, diff)
			}
		})
		t.Run("DecodeJSONWithPool", func(t *testing.T) {
			got, err := DecodeJSONWithPool(encodedData)
			if err != nil {
				t.Fatal(err)
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
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("got: %v,want: %v, diff: %s", got, want, diff)
			}
		})
	}
}

var (
	EncResult string
	JData     = JsonData{
		ID:    1,
		Name:  "Jack",
		Items: []string{"knife", "shield", "herbs"},
	}

	DecResult JsonData
	SData     = `{"id":1,"name":"Jack","items":["knife","shield","herbs"]}`
)

func BenchmarkEncodeJSON(b *testing.B) {
	b.ReportAllocs()
	var r string
	for n := 0; n < b.N; n++ {
		r, _ = EncodeJSON(JData)
	}
	EncResult = r
}

func BenchmarkEncodeJSONStream(b *testing.B) {
	b.ReportAllocs()
	var r string
	for n := 0; n < b.N; n++ {
		r, _ = EncodeJSONStream(JData)
	}
	EncResult = r
}

func BenchmarkEncodeJSONStreamWithPool(b *testing.B) {
	b.ReportAllocs()
	var r string
	for n := 0; n < b.N; n++ {
		r, _ = EncodeJSONStreamWithPool(JData)
	}
	EncResult = r
}

func BenchmarkDecodeJSON(b *testing.B) {
	b.ReportAllocs()
	var r JsonData
	for n := 0; n < b.N; n++ {
		r, _ = DecodeJSON(SData)
	}
	DecResult = r
}

func BenchmarkDecodeJSONWithPool(b *testing.B) {
	b.ReportAllocs()
	var r JsonData
	for n := 0; n < b.N; n++ {
		r, _ = DecodeJSONWithPool(SData)
	}
	DecResult = r
}

func BenchmarkDecodeJSONStream(b *testing.B) {
	b.ReportAllocs()
	var r JsonData
	for n := 0; n < b.N; n++ {
		data := strings.NewReader(SData)
		r, _ = DecodeJSONStream(data)
	}
	DecResult = r
}

func BenchmarkDecodeJSONStreamWithPool(b *testing.B) {
	b.ReportAllocs()
	var r JsonData
	for n := 0; n < b.N; n++ {
		data := strings.NewReader(SData)
		r, _ = DecodeJSONStreamWithPool(data)
	}
	DecResult = r
}

// [~/go/src/github.com/ludwig125/sync-pool/json] $go test -bench . -count=4
// goos: linux
// goarch: amd64
// pkg: github.com/ludwig125/sync-pool/json
// BenchmarkEncodeJSON-8                    2344576               502 ns/op             176 B/op          3 allocs/op
// BenchmarkEncodeJSON-8                    2357299               507 ns/op             176 B/op          3 allocs/op
// BenchmarkEncodeJSON-8                    2357732               503 ns/op             176 B/op          3 allocs/op
// BenchmarkEncodeJSON-8                    2345443               509 ns/op             176 B/op          3 allocs/op
// BenchmarkEncodeJSONStream-8              1862427               637 ns/op             256 B/op          5 allocs/op
// BenchmarkEncodeJSONStream-8              1851087               642 ns/op             256 B/op          5 allocs/op
// BenchmarkEncodeJSONStream-8              1848727               639 ns/op             256 B/op          5 allocs/op
// BenchmarkEncodeJSONStream-8              1853800               636 ns/op             256 B/op          5 allocs/op
// BenchmarkEncodeJSONStreamWithPool-8      2063480               580 ns/op             144 B/op          3 allocs/op
// BenchmarkEncodeJSONStreamWithPool-8      2061885               574 ns/op             144 B/op          3 allocs/op
// BenchmarkEncodeJSONStreamWithPool-8      2052324               572 ns/op             144 B/op          3 allocs/op
// BenchmarkEncodeJSONStreamWithPool-8      2086417               577 ns/op             144 B/op          3 allocs/op
// BenchmarkDecodeJSON-8                     574186              1894 ns/op             448 B/op         12 allocs/op
// BenchmarkDecodeJSON-8                     629776              1900 ns/op             448 B/op         12 allocs/op
// BenchmarkDecodeJSON-8                     620904              1904 ns/op             448 B/op         12 allocs/op
// BenchmarkDecodeJSON-8                     566812              1903 ns/op             448 B/op         12 allocs/op
// BenchmarkDecodeJSONWithPool-8             621783              1767 ns/op             312 B/op         10 allocs/op
// BenchmarkDecodeJSONWithPool-8             734518              1753 ns/op             312 B/op         10 allocs/op
// BenchmarkDecodeJSONWithPool-8             705708              1752 ns/op             312 B/op         10 allocs/op
// BenchmarkDecodeJSONWithPool-8             703803              1697 ns/op             312 B/op         10 allocs/op
// BenchmarkDecodeJSONStream-8               516535              2232 ns/op            1136 B/op         15 allocs/op
// BenchmarkDecodeJSONStream-8               471819              2264 ns/op            1136 B/op         15 allocs/op
// BenchmarkDecodeJSONStream-8               480862              2263 ns/op            1136 B/op         15 allocs/op
// BenchmarkDecodeJSONStream-8               451242              2255 ns/op            1136 B/op         15 allocs/op
// BenchmarkDecodeJSONStreamWithPool-8       578415              2035 ns/op            1000 B/op         13 allocs/op
// BenchmarkDecodeJSONStreamWithPool-8       508789              2074 ns/op            1000 B/op         13 allocs/op
// BenchmarkDecodeJSONStreamWithPool-8       548799              2068 ns/op            1000 B/op         13 allocs/op
// BenchmarkDecodeJSONStreamWithPool-8       523879              2075 ns/op            1000 B/op         13 allocs/op
// PASS
// ok      github.com/ludwig125/sync-pool/json     43.694s
// [~/go/src/github.com/ludwig125/sync-pool/json] $
