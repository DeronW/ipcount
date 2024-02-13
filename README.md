# IP Count

Counting distinct IPs, within tree structure, it have better performace, within custom serialize func could have smaller storage space.

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

Benchmark

```sh
goos: darwin
goarch: amd64
pkg: github.com/DeronW/ipcount
cpu: Intel(R) Core(TM) i5-1038NG7 CPU @ 2.00GHz
Benchmark_parse-8         343771              3539 ns/op
Benchmark_append-8       2187868               520.1 ns/op
PASS
ok      github.com/DeronW/ipcount       4.386s
```