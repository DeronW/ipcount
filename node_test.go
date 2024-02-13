package ipcount

import (
	"net"
	"testing"
)

func Test_node(t *testing.T) {
	n := Node{}

	isNew := n.insert(net.ParseIP("1.1.1.1"), 1)
	if !isNew {
		t.Error("insert a new IP, but not recorgnized")
	}

	v4, v6 := n.stat()
	if v4 != 1 {
		t.Error("IPv4 count error")
	}
	if v6 != 0 {
		t.Error("IPv6 count error")
	}

	mapValue := n.MapValue()
	if mapValue["1.1.1.1"] != 1 {
		t.Error("map value error")
	}

	isNew2 := n.insert(net.ParseIP("1.1.1.1"), 2)
	if isNew2 {
		t.Error("insert error")
	}
	mapValue2 := n.MapValue()
	if mapValue2["1.1.1.1"] != 3 {
		t.Error("mapValue2 error")
	}

	n.Append(net.ParseIP("1.1.1.1"))

	mapValue3 := n.MapValue()
	if mapValue3["1.1.1.1"] != 4 {
		t.Error("mapValue3 error")
	}

	n.Append(net.ParseIP("::2"))
	mapValue4 := n.MapValue()
	if mapValue4["0000:0000:0000:0000:0000:0000:0000:0002"] != 1 {
		t.Error("mapValue4 error")
	}
}
