<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [cmap](#cmap)
  - [Comparing with sync.map](#comparing-with-syncmap)
  - [Auto-generate](#auto-generate)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# cmap

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

## Comparing with sync.map
| cases | cmap | sync.map | url |
|-----------| --- | --- |------ |
| GET | 500000,3483 ns/op,5399 B/op,3 allocs/op | 200000,5359 ns/op,5399 B/op,3 allocs/op | [cmap.Get-click to location](https://github.com/fwhezfwhez/cmap/blob/3ea97e6c5de723adc78aa8469c7be61186754c04/map_test.go#L280) ,[sync.Get-click to location](https://github.com/fwhezfwhez/cmap/blob/3ea97e6c5de723adc78aa8469c7be61186754c04/map_test.go#L296)|
| SET | 300000,4273 ns/op,6434 B/op,40 allocs/op | 300000,3833 ns/op,6464 B/op,42 allocs/op | ... |


## Auto-generate

cmap provides auto-generate api to generate a type-defined map.It will save cost of assertion while using interface{}
```go
package main
import(
    "github.com/fwhezfwhez/cmap"
    "fmt"
)
func main() {
    content := cmap.GenerateTypeSyncMap()
    fmt.Println(content)
}
```

Output:

```go
package model

import (
        "encoding/json"
        "fmt"
        "math"
        "sync"
        "sync/atomic"
        "time"
)

// Auto-generate by github.com/fwhezfwhez/cmap.GenerateTypeSyncMap()
// Note:
// You might put this auto-genereted file to the package where User{} is defined.
// How to use this auto-generate concurrently-safe UserMap?
/*
        var user = User{
            UserId: 10086,
        }
        m := NewUserMap()
        m.Set(fmt.Sprintf("%d", user.UserId), user)
        _ = m.Get(fmt.Sprintf("%d", user.UserId))
        m.Delete(fmt.Sprintf("%d", user.UserId))
        m.SetEx(fmt.Sprintf("%d", user.UserId), user, 3*24*60*60)
*/
// And you can supervisor it by:
/*
    go func(){
                for {
                        m.ClearExpireKeys()
                        time.Sleep(24 * time.Hour)
                }
        }()
*/
type UserValue struct {
        //
        IsNil bool
        // value
        v User
        // expire_in, -1-no time limit
        exp int64

        // Map's offset, when map exec set/delete offset++
        offset int64
        // exec time unixnano
        execAt int64
}

// v is latter than v2 in time
func (v UserValue) LatterThan(v2 UserValue) bool {
        if v.execAt > v2.execAt {
                return true
        }

        if v.execAt < v2.execAt {
                return false
        }

        return v.offset > v2.offset
}

// v is former than v2 in time
func (v UserValue) FormerThan(v2 UserValue) bool {
        if v.execAt < v2.execAt {
                return true
        }
        if v.execAt > v2.execAt {
                return false
        }
        return v.offset < v2.offset
}

func (v UserValue) detail() map[string]interface{} {
        m := map[string]interface{}{
                "v":       v.v,
                "exp":     v.exp,
                "offset":  v.offset,
                "exec_at": v.execAt,
        }
        return m
}

const (
        USER_M_FREE = 0
        USER_M_BUSY = 1
)

type UserMap struct {
        mode int

        l *sync.RWMutex
        m map[string]UserValue

        dl    *sync.RWMutex
        dirty map[string]UserValue

        wl    *sync.RWMutex
        write map[string]UserValue

        dll    *sync.RWMutex
        offset int64
        del    map[string]UserValue
}

// Help viewing map's detail.
type userMapView struct {
        Mode int

        M map[string]interface{}

        Dirty map[string]interface{}

        Write map[string]interface{}

        Offset int64
        Del    map[string]interface{}
}

func newUserMap() *UserMap {
        return &UserMap{
                mode: USER_M_FREE,
                l:    &sync.RWMutex{},
                m:    make(map[string]UserValue),

                dl:    &sync.RWMutex{},
                dirty: make(map[string]UserValue),

                wl:    &sync.RWMutex{},
                write: make(map[string]UserValue),

                dll: &sync.RWMutex{},
                del: make(map[string]UserValue),
        }
}

// new a concurrent map
func NewUserMap() *UserMap {
        return newUserMap()
}
func (m *UserMap) GLock() {
        m.l.Lock()
        m.wl.Lock()
        m.dl.Lock()
        m.dll.Lock()
}
func (m *UserMap) GUnlock() {
        m.l.Unlock()
        m.wl.Unlock()
        m.dl.Unlock()
        m.dll.Unlock()
}

func (m *UserMap) GRLock() {
        m.l.RLock()
        m.wl.RLock()
        m.dl.RLock()
        m.dll.RLock()
}
func (m *UserMap) GRUnlock() {
        m.l.RUnlock()
        m.wl.RUnlock()
        m.dl.RUnlock()
        m.dll.RUnlock()
}

func (m *UserMap) IsBusy() bool {
        m.GRLock()

        defer m.GRUnlock()

        return m.mode == USER_M_BUSY
}

// map.Set
func (m *UserMap) Set(key string, value User) {
        ext := time.Now().UnixNano()
        offset := m.offsetIncr()

        // only when not busy, m is writable
        if !m.IsBusy() {
                m.l.Lock()

                m.m[key] = UserValue{
                        v:   value,
                        exp: -1,

                        execAt: ext,
                        offset: offset,
                }
                m.l.Unlock()

        }

        m.dl.Lock()
        m.dirty[key] = UserValue{
                v:      value,
                exp:    -1,
                execAt: ext,
                offset: offset,
        }
        m.dl.Unlock()

        // only when m is busy, write map is writable
        if m.IsBusy() {
                m.wl.Lock()
                m.write[key] = UserValue{
                        v:   value,
                        exp: -1,

                        execAt: ext,
                        offset: offset,
                }
                m.wl.Unlock()
        }
}

func (m *UserMap) offsetIncr() int64 {
        atomic.AddInt64(&m.offset, 1)
        atomic.CompareAndSwapInt64(&m.offset, math.MaxInt64, 0)
        return m.offset
}

// map.SetEX
// key-value will be put with expired time limit.
// expired keys will be deleted as soon as calling m.Get(key), or calling m.ClearExpireKeys()
func (m *UserMap) SetEx(key string, value User, seconds int) {
        ext := time.Now().UnixNano()
        offset := m.offsetIncr()

        // only when not busy, m is writable
        if !m.IsBusy() {
                m.l.Lock()

                m.m[key] = UserValue{
                        v:   value,
                        exp: time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),

                        execAt: ext,
                        offset: offset,
                }
                m.l.Unlock()

        }

        m.dl.Lock()

        m.dirty[key] = UserValue{
                v:      value,
                exp:    time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),
                execAt: ext,
                offset: offset,
        }
        m.dl.Unlock()

        // only when m is busy, write map is writable
        if m.IsBusy() {
                m.wl.Lock()
                m.write[key] = UserValue{
                        v:      value,
                        exp:    time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),
                        execAt: ext,
                        offset: offset,
                }
                m.wl.Unlock()

        }
}

// map.SetNx
// If key exist, do nothing, otherwise set key,value into map
func (m *UserMap) SetNx(key string, value User) {
        m.SetExNx(key, value, -1)
}

// map.SetEXNX
// If key exist, do nothing, otherwise set key,value into map
func (m *UserMap) SetExNx(key string, value User, seconds int) {
        if m.IsBusy() {
                m.dl.RLock()
                _, exist := m.dirty[key]
                m.dl.RUnlock()
                if exist {
                        return
                } else {
                        m.SetEx(key, value, seconds)
                        return
                }
        } else {
                m.l.RLock()
                _, exist := m.m[key]
                m.l.RUnlock()
                if exist {
                        return
                } else {
                        m.SetEx(key, value, seconds)
                        return
                }
        }
}

// If key is expired or not existed, return nil
func (m *UserMap) Get(key string) User {
        if !m.IsBusy() {
                return userGetFrom(m.l, m.m, key)
        } else {
                return userGetFrom(m.dl, m.dirty, key)
        }
}

// Delete
func (m *UserMap) Delete(key string) {
        offset := m.offsetIncr()
        ext := time.Now().UnixNano()
        if !m.IsBusy() {
                m.l.RLock()
                _, ok := m.m[key]
                m.l.RUnlock()

                if ok {
                        m.l.Lock()
                        delete(m.m, key)
                        m.l.Unlock()
                }
        }

        m.dl.RLock()
        _, ok2 := m.dirty[key]
        m.dl.RUnlock()

        if ok2 {
                m.dl.Lock()
                delete(m.dirty, key)
                m.dl.Unlock()
        }

        if m.IsBusy() {
                m.dll.Lock()
                m.del[key] = UserValue{
                        exp: 0,
                        v:  User{},

                        execAt: ext,
                        offset: offset,
                }
                m.dll.Unlock()
        }
}

func (m *UserMap) Detail() userMapView {
        var listMaxNum = 10
        var flag = 0

        mv := userMapView{
                M:     make(map[string]interface{}),
                Dirty: make(map[string]interface{}),
                Del:   make(map[string]interface{}),
                Write: make(map[string]interface{}),
        }
        mv.Offset = m.offset

        m.l.RLock()
        for k, _ := range m.m {
                flag++
                mv.M[k] = m.m[k].detail()
                if flag > listMaxNum {
                        mv.M["reach-max-detail"] = "end"
                        break
                }
        }
        m.l.RUnlock()

        m.dl.RLock()
        flag = 0
        for k, _ := range m.dirty {
                mv.Dirty[k] = m.dirty[k].detail()
                flag++
                if flag > listMaxNum {
                        mv.Dirty["reach-max-detail"] = "end"
                        break
                }
        }
        m.dl.RUnlock()

        m.wl.RLock()
        flag = 0
        for k, _ := range m.write {
                mv.Write[k] = m.write[k].detail()
                flag++
                if flag > listMaxNum {
                        mv.Write["reach-max-detail"] = "end"
                        break
                }
        }
        m.wl.RUnlock()

        flag = 0
        m.dll.RLock()
        for k, _ := range m.del {
                mv.Del[k] = m.del[k].detail()
                flag++
                if flag > listMaxNum {
                        mv.Del["reach-max-detail"] = "end"
                        break
                }
        }
        m.dll.RUnlock()
        return mv
}

func (m *UserMap) PrintDetail() string {
        b, e := json.MarshalIndent(m.Detail(), "", "  ")
        if e != nil {
                fmt.Println(e.Error())
                return ""
        }
        fmt.Println(string(b))
        return string(b)
}

func userGetFrom(l *sync.RWMutex, m map[string]UserValue, key string) User {
        l.RLock()
        value, ok := m[key]
        l.RUnlock()

        if !ok {
                return User{}
        }

        if value.exp == -1 {
                return value.v
        }

        if time.Now().UnixNano() >= value.exp {
                l.Lock()
                delete(m, key)
                l.Unlock()
                return User{}
        }

        return value.v
}

func (m *UserMap) setBusy() {
        m.l.Lock()
        defer m.l.Unlock()
        m.dl.Lock()
        defer m.dl.Unlock()
        m.wl.Lock()
        defer m.wl.Unlock()

        m.mode = USER_M_BUSY
}

func (m *UserMap) setFree() {
        m.l.Lock()
        defer m.l.Unlock()
        m.dl.Lock()
        defer m.dl.Unlock()
        m.wl.Lock()
        defer m.wl.Unlock()

        m.mode = USER_M_FREE
}

// Returns Map.m real length, not m.dirty or m.write.
func (m *UserMap) Len() int {
        m.l.RLock()
        defer m.l.RUnlock()
        return len(m.m)
}

// ClearExpireKeys clear expired keys, and it will not influence map write and read.
// When call m.ClearExpireKeys(), first will set m.mode=M_BUSY.
// At this moment, operation of write to Map.m is denied and instead data will be writen to Map.write which will sync to Map.m after clear job done.
// operation of read will use Map.dirty.
// After clear job has been done, Map.dirty will be cleared and copy from Map.m, Map.write will be unwritenable and data in Map.write will sync to Map.m.
func (m *UserMap) ClearExpireKeys() int {
        if m.IsBusy() {
                // on clearing job, another clear job do nothing
                // returned cleared key number is not concurrently consistent, because m.mode is not considered locked.
                return 0
        }
        // clear Map.m expired keys
        n := m.clearExpireKeys()

        // sync new writen data from Map.write
        m.l.Lock()

        m.wl.Lock()
        for k, v := range m.write {
                m.m[k] = v
        }
        m.write = make(map[string]UserValue)
        m.wl.Unlock()

        // sync deleted operation from Map.del
        m.dll.Lock()
        for k, v := range m.del {
                v2, ok := m.m[k]
                if !ok {
                        delete(m.del, k)
                        continue
                }
                if v.LatterThan(v2) {
                        delete(m.del, k)
                        delete(m.m, k)
                }
        }
        m.del = make(map[string]UserValue)
        m.dll.Unlock()

        m.l.Unlock()

        m.dl.Lock()
        m.dirty = make(map[string]UserValue)

        m.l.RLock()
        for key, _ := range m.m {
                m.dirty[key] = m.m[key]
        }
        m.l.RUnlock()
        m.dl.Unlock()

        return n
}
func (m *UserMap) clearExpireKeys() int {
        m.setBusy()
        defer m.setFree()
        var num int
        var shouldDelete = make([]string, 0, len(m.m))

        m.l.RLock()

        for k, v := range m.m {
                if v.exp == -1 {
                        continue
                }
                if v.exp < time.Now().UnixNano() {
                        shouldDelete = append(shouldDelete, k)
                }
        }
        m.l.RUnlock()

        m.l.Lock()
        for _, v := range shouldDelete {
                num++
                delete(m.m, v)
        }

        m.l.Unlock()
        return num
}

```
