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
