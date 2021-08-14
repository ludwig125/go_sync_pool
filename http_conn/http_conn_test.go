package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"testing"
)

func hello(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("Hello, Gophers!")); err != nil {
		log.Printf("failed to Write: %v", err)
	}
}

func startServer() *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", hello)

	wg.Done()
	go func() {
		log.Println("start server")
		if err := http.ListenAndServe(":3000", mux); err != http.ErrServerClosed {
			// if err := srv.ListenAndServe(); err != nil {
			log.Printf("failed to ListenAndServe: %v", err)
		}
		log.Println("server shutdown")
	}()
	return &wg
}

func requestClient() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:3000", nil)
	if err != nil {
		log.Printf("NewRequest failed: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("failed to Do request: %v", err)
	}
	defer resp.Body.Close()

	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		log.Fatal(err)
	}
	// log.Println(res)
}

func init() {
	wg := startServer()
	wg.Wait()
	fmt.Println("done wg.Wait")
	// go func() {
	// 	fmt.Println("sleep 10s")
	// 	time.Sleep(10 * time.Second)
	// 	if err := srv.Shutdown(context.TODO()); err != nil {
	// 		// Error from closing listeners, or context timeout:
	// 		log.Println("Failed to gracefully shutdown:", err)
	// 	}
	// }()
}

func BenchmarkRequest(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		requestClient()
	}
}

func BenchmarkRequest2(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		requestClient()
	}
}
