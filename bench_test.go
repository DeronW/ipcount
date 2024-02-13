package ipcount

import (
	"net"
	"testing"
)

func Benchmark_parse(b *testing.B) {
	src := map[string]int{
		"192.168.1.2": 13,
		"::1":         17,
		"afmt.Println(2222, unsafe.Sizeof(tree))": 0,
	}

	for i := 0; i < b.N; i++ {
		Parse(src)
	}
}

func Benchmark_append(b *testing.B) {
	t := New()
	ip := net.ParseIP("::7")

	for i := 0; i < b.N; i++ {
		t.Append(ip)
	}
}
