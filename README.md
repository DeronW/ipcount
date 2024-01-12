# IP Count

statistic distinct IP count, within tree structure could have better performace, within custom serialize func could have smaller storage space.

### Features

- Memory optimized
- IPv6 supported
- Serialize/Deserialize
- Counting support

### Install

```sh
go get github.com/DeronW/ipcount
```

### Usage

```go
import "github.com/DeronW/ipcount"

var (
    tree *ipcount.Node
)

tree, _ = ipcount.Parse(map[string]int{
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
