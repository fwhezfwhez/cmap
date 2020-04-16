<p align="center">
    <a href="https://github.com/fwhezfwhez/cmap"><img src="https://user-images.githubusercontent.com/36189053/79290712-70a76400-7eff-11ea-8cb5-cefca8e4adfc.png"></a>
</p>

<p align="center">
    <a href="https://godoc.org/github.com/fwhezfwhez/cmap"><img src="http://img.shields.io/badge/godoc-reference-blue.svg?style=flat"></a>
    <a href="https://www.travis-ci.org/fwhezfwhez/cmap"><img src="https://www.travis-ci.org/fwhezfwhez/cmap.svg?branch=master"></a>
    <a href="https://codecov.io/gh/fwhezfwhez/cmap"><img src="https://codecov.io/gh/fwhezfwhez/cmap/branch/master/graph/badge.svg"></a>
</p>

cmap is a concurrently safe map in golang. Providing apis below:

- SET
- GET
- SETEX
- SETNX

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Comparing with sync.map](#comparing-with-syncmap)
- [Analysis](#analysis)
- [Start](#start)
- [Auto-generate](#auto-generate)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Comparing with sync.map, chan-map
GET-benchmark-b.N

| cases | n | ns/op | B/ob | allocs/op | link |
| ---- | --- | --- | -- | --- |----- |
| cmap | 5000000 | 345 ns/op |24 B/op | 1 allocs/op | [cmap](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/map_test.go#L232) |
| sync.map | 3000000 | 347 ns/op | 24 B/op | 2 allocs/op | [sync.map](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/map_test.go#L247) |
| chan-map | 100000 |15670 ns/op | 6112 B/op | 14 allocs/op | [chan-map](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/chan-map_test.go#L56) |

GET-parallel-pb

| cases | n | ns/op | B/ob | allocs/op | link |
| ---- | --- | --- | -- | --- |----- |
| cmap | 500000 | 3409 ns/op | 5399 B/op | 3 allocs/op | [cmap](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/map_test.go#L290) |
| sync.map | 200000 | 5359 ns/op | 5399 B/op | 3 allocs/op | [sync.map](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/map_test.go#L308) |
| chan-map | 500000	| 5483 ns/op | 6111 B/op | 14 allocs/op | [chan-map](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/chan-map_test.go#L108) |

SET-benchmark-b.N

| cases | n | ns/op | B/ob | allocs/op | link |
| ---- | --- | --- | -- | --- |----- |
| cmap | 1000000 | 1820 ns/op | 617,B/op | 5 allocs/op | [cmap](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/map_test.go#L213) |
| sync.map | 1000000 | 1931 ns/op | 243 B/op | 9 allocs/op | [sync.map](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/map_test.go#L222) |
| chan-map | 500000	| 4140 ns/op | 1043 B/op | 14 allocs/op | [chan-map](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/chan-map_test.go#L40) |

SET-parallel-pb

| cases | n | ns/op | B/ob | allocs/op | link |
| ---- | --- | --- | -- | --- |----- |
| cmap | 500000 | 4020 ns/op | 6434 B/op | 40 allocs/op | [cmap](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/map_test.go#L262) |
| sync.map | 500000 | 4100 ns/op | 6464 B/op | 42 allocs/op | [sync.map](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/map_test.go#L276) |
| chan-map | 300000 | 6186 ns/op | 7164 B/op | 51 allocs/op | [chan-map](https://github.com/fwhezfwhez/cmap/blob/6df9dfc8a3c29eb19c0a72cbd7d3917185c5ecfa/chan-map_test.go#L90) |

## Analysis
mode: M_FREE

| x=mem(m.m, m.dirty, m.write, m.del) <br> y=-state(readable, writable) | m.m | m.dirty | m.write | m.del |
| --- | --- | --- | --- |------ |
| read | yes | no | no | no |
| write| yes | yes | no | no |

mode: M_BUSY

| x=mem(m.m, m.dirty, m.write, m.del) <br> y=-state(readable, writable) | m.m | m.dirty | m.write | m.del |
| --- | --- | --- |-- | ---- |
| read | no | yes | no | no |
| write| no | yes | yes | yes |

## Start
`go get github.com/fwhezfwhez/cmap`

```go
package main
import (
   "fmt"
   "github.com/fwhezfwhez/cmap"
)
func main() {
    m := cmap.NewMap()
    m.Set("username", "cmap")
    m.SetEx("password", 123, 5)
    m.Get("username")
    m.Delete("username")
}
```

## Auto-generate(develping)
cmap provides auto-generate api to generate a type-defined map.It will save cost of assertion while using interface{}
```go
 package main
 import (
         "fmt"
         "github.com/fwhezfwhez/cmap"
 )
 func main() {
        fmt.Println(cmap.GenerateTypeSyncMap("Teacher", map[string]string{
		"${package_name}": "model",
	}))
 }
```
Output:
```go
```
