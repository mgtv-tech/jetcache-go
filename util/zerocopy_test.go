package util

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testStr   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	testBytes = []byte(testStr)
)

func TestByteStringConvert(t *testing.T) {
	b := Bytes(testStr)
	s := String(b)
	assert.Equal(t, testStr, s)
}

func TestWithGC(t *testing.T) {
	b := test()
	assert.Equal(t, string(b), "hello")
	fmt.Printf("%v\n", b)
	fmt.Printf("%v\n", b)
}

func test() []byte {
	defer runtime.GC()
	x := make([]byte, 5)
	x[0] = 'h'
	x[1] = 'e'
	x[2] = 'l'
	x[3] = 'l'
	x[4] = 'o'
	return Bytes(string(x))
}

func BenchmarkBytesSafe(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = []byte(testStr)
		}
	})
}

func BenchmarkStringSafe(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = string(testBytes)
		}
	})
}

func BenchmarkBytesUnSafe(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Bytes(testStr)
		}
	})
}

func BenchmarkStringUnSafe(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			String(testBytes)
		}
	})
}
