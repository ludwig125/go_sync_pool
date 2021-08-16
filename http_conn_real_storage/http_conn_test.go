package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
)

func requestClient(id string) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:3000", nil)
	if err != nil {
		log.Printf("NewRequest failed: %v", err)
	}
	params := req.URL.Query()
	params.Add("id", id)
	req.URL.RawQuery = params.Encode()

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to Do request: %v", err)
	}
	defer resp.Body.Close()

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(res)
}

func init() {
	go func() {
		if err := runPersonAPI("3000", "sample_db"); err != nil {
			log.Panicf("failed to runPersonAPI: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond) // serverが起動するまで少し待つ
}

func TestRequest(t *testing.T) {
	got := requestClient("1")
	t.Log(got)

	// time.Sleep(20 * time.Second)
}

func BenchmarkRequest(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		requestClient("1")
	}
}

func BenchmarkRequest2(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		requestClient("1")
	}
}
