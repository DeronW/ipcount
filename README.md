# IP Tree

statistic distinct IP count, within tree structure could have better performace, within custom serialize func could have smaller storage space.

### Features

- Memory optimized
- IPv6 supported
- Serialize/Deserialize
- Counting support

### Install

```sh
go get github.com/DeronW/iptree
```

### Usage

```go
import "github.com/DeronW/iptree"

var (
    tree *iptree.Node
)

tree, _ = iptree.Parse(map[string]int{
    "192.168.1.1": 1,
    "192.168.1.2": 2,
    "::1": 3,
    "::2": 4,
})

tree.Append(net.ParseIP("192.168.1.1"))
tree.Append(net.ParseIP("127.0.0.1"))
tree.Append(net.ParseIP("::3"))

fmt.Println(tree.MapValue())
tree.MapValue()

```
