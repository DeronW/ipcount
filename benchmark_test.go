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
	tree := New()

	t := 0

	for i := 0; i < b.N; i++ {
		t += 10
		ip := net.IPv4(byte(t/16777216), byte(t/65536), byte(t/256), byte(t%256))
		tree.Append(ip)
	}
	// fmt.Println(tree.Count())
}
