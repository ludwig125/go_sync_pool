package main

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

var pool = &sync.Pool{
	New: func() interface{} {
		mem := make([]string, 5)
		return &mem
	},
}

func ReplicateStrNTimes(s string) []string {
	n := 5
	ss := make([]string, n)
	for i := 0; i < n; i++ {
		ss[i] = s
	}
	return ss
}

func ReplicateStrNTimesWithPool(s string) []string {
	n := 5
	ss := pool.Get().(*[]string)
	defer pool.Put(ss)
	for i := 0; i < n; i++ {
		(*ss)[i] = s
	}
	return *ss
}

func TestReplicateStrNTimes(t *testing.T) {
	want := []string{
		"12345",
		"12345",
		"12345",
		"12345",
		"12345",
	}

	for i := 0; i < 2; i++ {
		count := fmt.Sprintf("%d", i)
		t.Run("ReplicateStrNTimes"+count, func(t *testing.T) {
			got := ReplicateStrNTimes("12345")
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
		t.Run("ReplicateStrNTimesWithPool"+count, func(t *testing.T) {
			got := ReplicateStrNTimes("12345")
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got: %s, want: %s", got, want)
			}
		})
	}
}

var Result []string

func BenchmarkReplicateStrNTimes(b *testing.B) {
	b.ReportAllocs()
	var r []string
	for n := 0; n < b.N; n++ {
		r = ReplicateStrNTimes("12345")
	}
	Result = r
}

func BenchmarkReplicateStrNTimesWithPool(b *testing.B) {
	b.ReportAllocs()
	var r []string
	for n := 0; n < b.N; n++ {
		r = ReplicateStrNTimesWithPool("12345")
	}
	Result = r
}
