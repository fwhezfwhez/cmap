<p align="center">
    <a href="https://github.com/fwhezfwhez/cmap"><img src="https://user-images.githubusercontent.com/36189053/79290712-70a76400-7eff-11ea-8cb5-cefca8e4adfc.png"></a>
</p>

<p align="center">
    <a href="https://godoc.org/github.com/fwhezfwhez/cmap"><img src="http://img.shields.io/badge/godoc-reference-blue.svg?style=flat"></a>
    <a href="https://www.travis-ci.org/fwhezfwhez/cmap"><img src="https://www.travis-ci.org/fwhezfwhez/cmap.svg?branch=master"></a>
    <a href="https://codecov.io/gh/fwhezfwhez/cmap"><img src="https://codecov.io/gh/fwhezfwhez/cmap/branch/master/graph/badge.svg"></a>
</p>

cmap is a concurrently safe map in golang. Providing two stable map type with apis:

**map types**

- map   `cmap.NewMap`
- mapv2 `cmap.NewMapV2`

**apis**

- SET
- GET
- Incr
- IncrBy
- IncrByEx
- SETEX
- SETNX
- SETEXNX

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
| cmap | 5000000 | 345 ns/op |24 B/op | 1 allocs/op ||
| sync.map | 3000000 | 347 ns/op | 24 B/op | 2 allocs/op | |
| chan-map | 100000 |15670 ns/op | 6112 B/op | 14 allocs/op | |

GET-parallel-pb

| cases | n | ns/op | B/ob | allocs/op | link |
| ---- | --- | --- | -- | --- |----- |
| cmap | 500000 | 3409 ns/op | 5399 B/op | 3 allocs/op | |
| sync.map | 200000 | 5359 ns/op | 5399 B/op | 3 allocs/op | |
| chan-map | 500000	| 5483 ns/op | 6111 B/op | 14 allocs/op | |

SET-benchmark-b.N

| cases | n | ns/op | B/ob | allocs/op | link |
| ---- | --- | --- | -- | --- |----- |
| cmap | 1000000 | 1820 ns/op | 617,B/op | 5 allocs/op |  |
| sync.map | 1000000 | 1931 ns/op | 243 B/op | 9 allocs/op | |
| chan-map | 500000	| 4140 ns/op | 1043 B/op | 14 allocs/op | |

SET-parallel-pb

| cases | n | ns/op | B/ob | allocs/op | link |
| ---- | --- | --- | -- | --- |----- |
| cmap | 500000 | 4020 ns/op | 6434 B/op | 40 allocs/op |  |
| sync.map | 500000 | 4100 ns/op | 6464 B/op | 42 allocs/op | |
| chan-map | 300000 | 6186 ns/op | 7164 B/op | 51 allocs/op | |

## start
go get github.com/fwhezfwhez/cmap

## 1. Anal
### 1.1 map
map is concurrently safe map. It consists of [m, dirty, write, del] and has three states in runtime: M_FREE1, M_FREE2, M_BUSY.
m, dirty, write, del are all golang official map. In different states, they work differently.

mode: M_FREE2
At this moment, `m` is totally working. All commands are available to `m` and all `write`/`del` operations will do the same to dirty.
`write`, `del` are resting and no use.

| x=mem(m.m, m.dirty, m.write, m.del) <br> y=-state(readable, writable) | m.m | m.dirty | m.write | m.del |
| --- | --- | --- | --- |------ |
| read | yes | no | no | no |
| write| yes | yes | no | no |

mode: M_BUSY
At this moment, it means a process of clearing expire keys of `m` are working. Now `m` is disable and dirty is put into use.
Now commands read from `dirty`, write to `dirty`. New keys are write to `write` and deleting options are write to `del`.
Since dirty share all read and write in M_FREE2 Mode, thus dirty provides consistent data to callers.

| x=mem(m.m, m.dirty, m.write, m.del) <br> y=-state(readable, writable) | m.m | m.dirty | m.write | m.del |
| --- | --- | --- |-- | ---- |
| read | no | yes | no | no |
| write| no | yes | yes | yes |

mode: M_FREE1
At this moment, it means clearing expire keys of `m` has finished, but it now should be migrated data from `write` and `del`.
Still `dirty` provides read and write.

As soon as mode from `M_FREE1` to `M_FREE2`, `m` will return to use and then clear dirty expired keys and reset `write` and `del`

| x=mem(m.m, m.dirty, m.write, m.del) <br> y=-state(readable, writable) | m.m | m.dirty | m.write | m.del |
| --- | --- | --- |-- | ---- |
| read | no | yes | no | no |
| write| no | yes | yes | yes |


** Why map is so fast? **

There should be  two jobs costing much time: `clear all m's expired keys`, `clear all dirty's expired keys`.

In this package, while clearing m, in M_BUSY mode, data read from dirty without blocking. write to `del`/`write` without blocking.

While clearing dirty, it's already in M_FREE2, write `m` and read `m` without blocking.

Where blocking, when data are migrating from `write`/`del` to `m`, set/del operations will get blocked. Apparently these data are few.

We transfer costs of clearing all expire keys into costs of migrating all increasing keys/deleting keys.This is why cmap is fast.

** Why design mapv2? **
map is fast however all keys share a common race lock. This is improvable when two keys are irrelevant totally. Thus mapv2 is working like
`hash` + `map`.

Only when two keys are hit into a same map by hash function, they hit a race lock.

To lower its rate, it's good to set bigger number of mapv2.slotnum.


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
