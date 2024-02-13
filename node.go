package ipcount

import (
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"strings"
)

type Node struct {
	// level value should be: 1 ~ 16
	level int
	// is leaf node or not
	leaf bool
	// ip value, only leaf node has count vaue
	value *int
	// sub bytes, max count is 256
	children map[byte]*Node
}

var (
	// limit MapValue result, avoid out of memory
	MaxMapValueCount = 1_000_000
)

/*
insert an IP into tree with value count, and return if
the IP is new to the tree.
*/
func (n *Node) insert(ip net.IP, v int) bool {
	node := n
	isNew := false

	for i := 0; i < 16; i++ {
		b := ip[i]
		if node.children == nil {
			node.children = map[byte]*Node{}
		}
		t, ok := node.children[b]

		isLeaf := false

		if i == 15 {
			isLeaf = true
		}

		if !ok {
			children := map[byte]*Node{}

			if isLeaf {
				isNew = true
				children = nil
			}

			t = &Node{
				level:    i,
				leaf:     isLeaf,
				value:    &v,
				children: children,
			}

			node.children[b] = t
		} else {
			if isLeaf {
				*t.value += v
			}
		}
		node = t
	}

	return isNew
}

/*
append a new ip to tree, and will return true if it's a new one
*/
func (n *Node) Append(ip net.IP) (isNew bool) {
	return n.insert(ip.To16(), 1)
}

func (n *Node) AppendIP(s string) (bool, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return false, fmt.Errorf("not a valid IP: %s", s)
	}
	return n.Append(ip), nil
}

func (n *Node) Remove(ip net.IP) {
	var node = n
	var parent *Node
	var b byte
	for _, b = range ip.To16() {
		c, ok := node.children[b]
		if !ok {
			return
		}
		parent = node
		node = c
	}
	delete(parent.children, b)
}

func (n *Node) RemoveIP(s string) error {
	ip := net.ParseIP(s)
	if ip == nil {
		return fmt.Errorf("not a valid IP: %s", s)
	}
	n.Remove(ip)
	return nil
}

/*
return tree's value as a map, same like params of `ipcount.Parse`
*/
func (n *Node) MapValue() map[string]int {
	data := map[string]int{}
	c := 0

	toIP := func(bs []byte) string {
		if isIPv4(bs) {
			return fmt.Sprintf("%d.%d.%d.%d", bs[12], bs[13], bs[14], bs[15])
		}
		seg := []string{}
		for i := 0; i < 16; i += 2 {
			seg = append(seg, hex.EncodeToString(bs[i:i+2]))
		}
		return strings.Join(seg, ":")
	}

	dfs(n, []byte{}, func(path []byte, value int) {
		c += 1
		if c > MaxMapValueCount {
			return
		}
		data[toIP(path)] = value
	})

	return data
}

/*
return counts: (ipv4_count, ipv6_count)
*/
func (n *Node) stat() (v4 int, v6 int) {
	dfs(n, []byte{}, func(path []byte, value int) {
		if isIPv4(path) {
			v4 += 1
		} else {
			v6 += 1
		}
	})
	return v4, v6
}

func (n *Node) Count() int {
	v4, v6 := n.stat()
	return v4 + v6
}

func (n *Node) IPv4Count() int {
	v4, _ := n.stat()
	return v4
}

func (n *Node) IPv6Count() int {
	_, v6 := n.stat()
	return v6
}

/*
depth first search, by ip in increasing order
*/
func dfs(n *Node, path []byte, cb func(path []byte, value int)) {
	if n == nil {
		return
	}
	if n.leaf {
		if n.value != nil {
			cb(path, *n.value)
		} else {
			cb(path, 0)
		}
	} else {
		keys := []byte{}
		for b := range n.children {
			keys = append(keys, b)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})
		for _, k := range keys {
			dfs(n.children[k], append(path, k), cb)
		}
	}
}
