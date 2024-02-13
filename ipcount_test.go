package ipcount

import (
	"fmt"
	"net"
	"testing"
)

func Test_new(t *testing.T) {
	tree := New()
	tree.AppendIP("::0")
	if tree.Count() != 1 {
		t.Error("count error")
	}

	fmt.Println(tree.MapValue())

	tree.AppendIP("::1")
	if tree.Count() != 2 {
		t.Error("count error")
	}
	fmt.Println(tree.MapValue())

	tree.RemoveIP("::1")
	if tree.Count() != 1 {
		t.Error("count error")
	}
	fmt.Println(tree.MapValue())
	tree.RemoveIP("::0")
	if tree.Count() != 0 {
		t.Error("count error")
	}
	fmt.Println(tree.MapValue())
}

func Test_parse(t *testing.T) {
	src := map[string]int{
		"192.168.1.2": 13,
		"::1":         17,
		"afmt.Println(2222, unsafe.Sizeof(tree))": 0,
	}

	tree := Parse(src)

	v := tree.MapValue()
	if v == nil {
		t.Error("map value should not be nil")
	}

	isNew := tree.Append(net.ParseIP("192.168.1.1"))
	if !isNew {
		t.Error("append failed, it is not new")
	}

	isNew = tree.Append(net.ParseIP("192.168.1.2"))
	if isNew {
		t.Error("append failed, it's not new")
	}
	// fmt.Println(v)
}

func Test_encode(t *testing.T) {
	tree := Parse(map[string]int{
		"192.168.10.1": 3,
		"192.168.10.5": 7,
	})
	tree.Append(net.ParseIP("192.168.10.11"))
	tree.Append(net.ParseIP("192.168.10.12"))
	tree.Append(net.ParseIP("192.168.12.13"))
	tree.Append(net.ParseIP("::1"))
	tree.Append(net.ParseIP("::2"))
	tree.Append(net.ParseIP("::2"))

	// fmt.Println(tree.MapValue())
	s := Encode(tree)

	tree2, err := Decode(s)
	if err != nil {
		t.Error(err)
	}
	// fmt.Println(tree2.MapValue())

	s2 := Encode(tree2)

	if s != s2 {
		t.Error("Encode or Decode is not correct")
	}
}
