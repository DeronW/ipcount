package ipcount

import (
	"bytes"
	"compress/flate"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sort"
)

type byteValue struct {
	Byte  byte
	Value int
}

var (
	// use `SL` as start symbol
	Head         = [2]byte{0x53, 0x4C}
	v4InV6Prefix = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff}
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
	// Pow_2_32 = 4_294_967_296
)

func New() *Node {
	return &Node{}
}

/*
parse a key-value map to ipcount, source map example

	{
		"127.0.0.1": 3,
		"::1": 5
	}
*/
func Parse(src map[string]int) *Node {
	tree := &Node{}
	for k, v := range src {
		ip := net.ParseIP(k)
		if ip != nil {
			tree.insert(ip, v)
		}
	}
	return tree
}

func isIPv4(bs []byte) bool {
	return bytes.Equal(bs[:12], v4InV6Prefix)
}

// breath first search
func bfs(n *Node, path []byte, cb func(path []byte, nodes map[byte]*Node)) {
	if n.children == nil {
		return
	}

	cb(path, n.children)

	keys := []byte{}
	for b := range n.children {
		keys = append(keys, b)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	for _, k := range keys {
		c := n.children[k]
		bfs(c, append(path, k), cb)
	}
}

func int2bytes(n int) []byte {
	// less than 2^6, share one byte
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
	// max value
	return []byte{0xff, 0xff, 0xff, 0xff}
}

/*
convert dynamic bytes to int number, return int and bytes count
*/
func bytes2int(bs []byte) (int, int) {
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

func Encode(root *Node) string {
	bs := []byte{Head[0], Head[1]}

	bfs(root, []byte{}, func(path []byte, nodes map[byte]*Node) {
		/*
			bytes, serialize rules, has 4 parts
			- [1 byte]
				- 1~4 bits, parent's level (0~15)
				- 5th bit, 1 if current node is leaf, 0 means not
				- 6~8th bits, reserved
			- [1 byte] byte of parent's key，it's omitted if parent is root
			- [1 byte] an integer, means how many nodes tailed
			- [n bytes] node's mark
				- [1 byte] address
				- [1 byte] number mark, only in leaf node. NOTE: in big-endian
					- 1~2 bits，means how many bytes tailed
						- 00: means no tailed bytes
						- 01: means 1 byte tailed
						- 10: means 2 bytes tailed
						- 11: means 3 bytes tailed
						if 00 presented, use `and` operator with `0x3f`,
						got final value
				-[0~3 bytes] if node's value need more bytes, 1 or at most 3 bytes
					will be tailed, max value will be [0x3f,0xff,0xff,0xff]
					NOTE: in uint-type
		*/

		// level: 0 ~ 15，leaf is and only leaf is 15
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

		// sort nodes
		keys := []byte{}
		for k := range nodes {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i] < keys[j]
		})

		for _, k := range keys {
			v := nodes[k]
			nodeLen += 1
			nodeCnt = append(nodeCnt, k)
			if isLeaf {
				nodeCnt = append(nodeCnt, int2bytes(*v.value)...)
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
	return hex.EncodeToString(bs)
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

	// record decode position with a cursor
	cursor := 0

	if bs[0] != Head[0] || bs[1] != Head[1] {
		return nil, errors.New("invalid bytes prefix")
	}
	// head always ocuppy 2 bytes
	cursor += 2

	parents := []byte{}

	// decode byte by byte, until the end
	for cursor < size {
		flag := bs[cursor]
		// level and isLeaf flag, take one byte
		cursor += 1

		level := (flag & 0xf0) >> 4
		parent := byte(0)
		if level > 0 {
			// non-root parent, take one byte
			parent = bs[cursor]
			cursor += 1
		}
		// level 如果小于 parents 的长度，说明需要回退到树的组父级节点
		// if level is less than length of parents,
		// it needs to back to someone grand level
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
		nCount := int(bs[cursor])
		// tailed nodes count, take one byte
		cursor += 1

		bvNodes := []byteValue{}

		for j := 0; j < nCount; j++ {
			bv := byteValue{Byte: bs[cursor]}
			// node address, take one byte
			cursor += 1

			if isLeaf {
				// 按最大字节数计算，但结尾的字节数不足，需要判断
				end := cursor + 4
				if end > size {
					end = size
				}
				v, bytesCount := bytes2int(bs[cursor:end])
				// dynamic count bytes, 0~3 bytes
				cursor += bytesCount
				bv.Value = v
			}

			bvNodes = append(bvNodes, bv)
		}
		err := mountChildren(root, parents, bvNodes)
		if err != nil {
			return nil, err
		}
	}

	return root, nil
}

func mountChildren(root *Node, parents []byte, bvNodes []byteValue) error {
	node := root
	for _, p := range parents {
		t, ok := node.children[p]
		if !ok {
			return fmt.Errorf("append failed, parents not exist: %v", parents)
		}
		node = t
	}

	node.children = map[byte]*Node{}
	level := len(parents)

	for _, t := range bvNodes {
		isLeaf := level == 15
		var value *int

		if isLeaf {
			// make a value copy, or it will cause value cover by same pointer
			a := t.Value
			value = &a
		}

		node.children[t.Byte] = &Node{
			level: level,
			leaf:  isLeaf,
			value: value,
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
