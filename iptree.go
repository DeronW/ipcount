package iptree

import (
	"bytes"
	"compress/flate"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
)

type Node struct {
	// level value should be: 1 ~ 16
	level int
	// is leaf node
	leaf bool
	// ip value, only leaf node has count vaue
	value int
	// sub bytes, max count is 256
	children map[byte]*Node
}

type byteValue struct {
	Byte  byte
	Value int
}

var (
	// use `SL` as start symbol
	Head = [2]byte{0x53, 0x4C}
	// limit MapValue result, avoid out of memory
	MaxMapValueCount = 1_000_000
	v4InV6Prefix     = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}
)

const (
	// thresholds
	Pow_2_6  = 64
	Pow_2_14 = 16_384
	Pow_2_22 = 4_194_304
	Pow_2_30 = 1_037_741_824
	// const
	Pow_2_8  = 256
	Pow_2_16 = 65_536
	Pow_2_24 = 16_777_216
	Pow_2_32 = 4_294_967_296
)

/*
parse a key-value map to iptree, source map example

	{
		"127.0.0.1": 3,
		"::1": 5
	}
*/
func Parse(src map[string]int) (*Node, error) {
	tree := &Node{}
	for k, v := range src {
		ip := net.ParseIP(k)
		if ip != nil {
			tree.insert(ip, v)
		}
	}
	return tree, nil
}

func (n *Node) insert(ip net.IP, v int) bool {
	node := n
	isNew := false

	for i := 0; i < 16; i++ {
		b := ip[i]
		if node.children == nil {
			node.children = map[byte]*Node{}
		}
		t, ok := node.children[b]

		if !ok {
			isNew = true

			leaf := false
			children := map[byte]*Node{}
			value := 0

			if i == 15 {
				leaf = true
				children = nil
				value = v
			}

			t = &Node{
				level:    i,
				leaf:     leaf,
				value:    value,
				children: children,
			}

			node.children[b] = t
		} else {
			t.value += v
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

/*
return tree's value as a map, same like params of `iptree.Parse`
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
func (n *Node) Count() (v4 int, v6 int) {
	dfs(n, []byte{}, func(path []byte, value int) {
		if isIPv4(path) {
			v4 += 1
		} else {
			v6 += 1
		}
	})

	return
}

func isIPv4(bs []byte) bool {
	return bytes.Equal(bs[:12], v4InV6Prefix)
}

// depth first search, by ip in increasing order
func dfs(n *Node, path []byte, cb func(path []byte, value int)) {
	if n == nil {
		return
	}
	if n.leaf {
		cb(path, n.value)
	} else {
		keys := []byte{}
		for b := range n.children {
			keys = append(keys, b)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] > keys[j]
		})
		for _, k := range keys {
			dfs(n.children[k], append(path, k), cb)
		}
	}
}

// breath first search
func bfs(n *Node, path []byte, cb func(path []byte, nodes map[byte]*Node)) {
	if n.children == nil {
		return
	}
	cb(path, n.children)
	for b, c := range n.children {
		bfs(c, append(path, b), cb)
	}
}

func intToBytes(n int) []byte {
	// less than 2^6, use common 1 byte
	if n < Pow_2_6 {
		return []byte{byte(n)}
	}
	// less than 2 ^ (6+8)
	if n < Pow_2_14 {
		return []byte{byte(n/Pow_2_8) | 0x40, byte(n % Pow_2_8)}
	}
	// less then 2 ^ (6+8+8)
	if n < Pow_2_22 {
		return []byte{
			byte(n/Pow_2_16) & 0x80,
			byte(n / Pow_2_8 % Pow_2_8),
			byte(n % Pow_2_8),
		}
	}
	// less then 2 ^ (6+8+8+8)
	if n < Pow_2_30 {
		return []byte{
			byte(n/Pow_2_24) & 0xc0,
			byte(n / Pow_2_16 % Pow_2_8),
			byte(n / Pow_2_8 % Pow_2_8),
			byte(n % Pow_2_8),
		}
	}
	return []byte{0xff, 0xff, 0xff, 0xff}
}

func bytesToInt(bs []byte) (int, int) {
	c := bs[0] & 0xc0
	first := bs[0] & 0x3f
	if c == 0x00 {
		return int(first), 1
	}
	if c == 0x40 {
		return int(first&0x3f)*Pow_2_8 + int(bs[1]), 2
	}
	if c == 0x80 {
		return int(first&0x3f)*Pow_2_16 +
			int(bs[1])*Pow_2_8 +
			int(bs[2]), 3
	}
	if c == 0xc0 {
		return int(first&0x3f)*Pow_2_24 +
			int(bs[1])*Pow_2_16 +
			int(bs[2])*Pow_2_8 +
			int(bs[3]), 4
	}
	return 0, 1
}

func Encode(root *Node) (string, error) {
	bs := []byte{Head[0], Head[1]}

	bfs(root, []byte{}, func(path []byte, nodes map[byte]*Node) {
		/*
			bytes 序列化规则， 按顺序分成 5 部分
			- 【1 byte】
				- 前 4 个 bit，表示父级的 level，顺着当前 parent 往回找
				- 第 5 个 bit，是否为叶子节点，1 为是，0 为否
				- 后 3 个 bit，保留位
			-【1 byte】parent 的 byte，如果父级是根，则省略这个字节
			-【1 byte】整数，表示后面有多少个节点 node
			-【n bytes，节点标记】
				-【1 byte】地址
				-【1 byte】数量标记，只有叶子节点才有数量。注意：用大端序
					- 前 2 个 bit，表示后面还有几个字节表示数量，00 表示没有，11 表示后面还有 3 个字节
						如果 00，则将这个 byte 与 0x3f 进行 “且” 运算，计算出当前值
				-【1~3 byte】如果节点的数值较大，则需要额外字节保存数值，与前一个字节共同构成
					【0x3f,0xff,0xff,0xff】范围的数字表示，注意：无符号整数
		*/

		// level 最小 0 最大 15，叶子节点是 15
		level := len(path)
		isLeaf := level == 15

		flag := byte(level) << 4
		if isLeaf {
			flag |= 0x08
		}

		block := []byte{flag}
		if level > 0 {
			block = append(block, path[level-1])
		}

		nodeLen := 0
		nodeCnt := []byte{}
		for k, v := range nodes {
			nodeLen += 1
			nodeCnt = append(nodeCnt, k)
			if isLeaf {
				nodeCnt = append(nodeCnt, intToBytes(v.value)...)
			}
		}
		block = append(block, byte(nodeLen))
		block = append(block, nodeCnt...)

		bs = append(bs, block...)
	})

	// bs, err := compress(bs)
	// if err != nil {
	// 	return "", err
	// }
	return hex.EncodeToString(bs), nil
}

func Decode(s string) (*Node, error) {
	root := &Node{}

	bs, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}

	size := len(bs)
	if size < 2 {
		return nil, errors.New("bytes length is too short")
	}
	if bs[0] != Head[0] || bs[1] != Head[1] {
		return nil, errors.New("invalid bytes prefix")
	}

	parents := []byte{}

	for i := 2; i < size; {
		flag := bs[i]
		i += 1

		level := flag & 0xf0 >> 4
		parent := byte(0)
		if level > 0 {
			parent = bs[i]
			i += 1
		}
		// level 如果小于 parents 的长度，说明需要回退到树的组父级节点
		if int(level) < len(parents) {
			parents = parents[:level-1]
		}
		// 上一个节点是叶子节点，并且当前节点需要回退到组父级，
		// 则需要把上一个叶子节点的 parent 去掉
		// 此时 level 最大是 14，给当前 level 层预留一个父级位置
		if len(parents) == 15 {
			parents = parents[:14]
		}
		// 一级节点的 parent 是空
		if level != 0 {
			parents = append(parents, parent)
		}

		isLeaf := level == 15
		nCount := int(bs[i])
		i += 1

		nodes := []byteValue{}

		for j := 0; j < nCount; j++ {
			n := byteValue{Byte: bs[i]}
			i += 1

			if isLeaf {
				// 按最大字节数计算，但结尾的字节数不足，需要判断
				end := i + 4
				if end > size {
					end = size
				}
				v, bytesCount := bytesToInt(bs[i:end])
				i += bytesCount
				n.Value = v
			}

			nodes = append(nodes, n)
		}
		err := appendChildren(root, parents, nodes)
		if err != nil {
			return nil, err
		}
	}

	return root, nil
}

func appendChildren(n *Node, parents []byte, nodes []byteValue) error {
	node := n
	for _, p := range parents {
		t, ok := node.children[p]
		if !ok {
			return fmt.Errorf("append failed, parents not exist: %v", parents)
		}
		node = t
	}
	node.children = map[byte]*Node{}
	level := len(parents)

	for _, t := range nodes {
		node.children[t.Byte] = &Node{
			level: level,
			leaf:  level == 15,
			value: t.Value,
		}
	}
	return nil
}

func compress(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	_, err := w.Write(src)
	if err != nil {
		return nil, fmt.Errorf("compression write error: %v", err)
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decomporess(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	r := flate.NewReader(&buf)
	_, err := r.Read(src)
	if err != nil {
		return nil, err
	}

	err = r.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
