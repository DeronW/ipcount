package iptree

import (
	"fmt"
	"net"
	"testing"
)

func Test_parse(t *testing.T) {
	src := map[string]int{
		"192.168.1.2": 13,
		// "::abcd":      17,
	}
	tree, err := Parse(src)
	fmt.Println("tree", tree, err)

	v := tree.MapValue()
	fmt.Println(v)

	isNew := tree.Append(net.ParseIP("192.168.1.1"))
	fmt.Println("isNew", isNew)

	isNew = tree.Append(net.ParseIP("192.168.1.2"))
	fmt.Println("isNew", isNew)
}

func Test_encode(t *testing.T) {
	tree, _ := Parse(map[string]int{})
	tree.Append(net.ParseIP("192.168.10.11"))
	tree.Append(net.ParseIP("192.168.10.11"))
	tree.Append(net.ParseIP("192.168.10.12"))
	tree.Append(net.ParseIP("192.168.12.13"))
	tree.Append(net.ParseIP("::1"))
	tree.Append(net.ParseIP("::2"))
	tree.Append(net.ParseIP("::2"))
	// tree.Append(net.ParseIP(""))
	fmt.Println("tree", tree)
	fmt.Println(tree.MapValue())
	s, _ := Encode(tree)
	fmt.Println("hex", s)

	tree2, _ := Decode(s)
	fmt.Println(tree2.MapValue())
}
