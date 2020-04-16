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
| cmap | 5000000 | 345 ns/op |24 B/op | 1 allocs/op | [cmap](https://github.com/fwhezfwhez/cmap/blob/aabf39042164d251011b20273a2ccba7639df915/map_test.go#L227) |
| sync.map | 3000000 | 347 ns/op | 24 B/op | 2 allocs/op | [sync.map](https://github.com/fwhezfwhez/cmap/blob/aabf39042164d251011b20273a2ccba7639df915/map_test.go#L242) |
| chan-map | 100000 |15670 ns/op | 6112 B/op | 14 allocs/op | [chan-map]() |

GET-parallel-pb

| cases | n | ns/op | B/ob | allocs/op | link |
| ---- | --- | --- | -- | --- |----- |
| cmap | 500000 | 3409 ns/op | 5399 B/op | 3 allocs/op | [cmap](https://github.com/fwhezfwhez/cmap/blob/aabf39042164d251011b20273a2ccba7639df915/map_test.go#L227) |
| sync.map | 200000 | 5359 ns/op | 5399 B/op | 3 allocs/op | [sync.map](https://github.com/fwhezfwhez/cmap/blob/aabf39042164d251011b20273a2ccba7639df915/map_test.go#L242) |
| chan-map | 500000	| 5483 ns/op | 6111 B/op | 14 allocs/op | [chan-map]() |

SET-benchmark-b.N

| cases | n | ns/op | B/ob | allocs/op | link |
| ---- | --- | --- | -- | --- |----- |
| cmap | 1000000 | 1820 ns/op | 617,B/op | 5 allocs/op | [cmap](https://github.com/fwhezfwhez/cmap/blob/aabf39042164d251011b20273a2ccba7639df915/map_test.go#L227) |
| sync.map | 1000000 | 1931 ns/op | 243 B/op | 9 allocs/op | [sync.map](https://github.com/fwhezfwhez/cmap/blob/aabf39042164d251011b20273a2ccba7639df915/map_test.go#L242) |
| chan-map | 500000	| 4140 ns/op | 1043 B/op | 14 allocs/op | [chan-map]() |

SET-parallel-pb

| cases | n | ns/op | B/ob | allocs/op | link |
| ---- | --- | --- | -- | --- |----- |
| cmap | 500000 | 4020 ns/op | 6434 B/op | 40 allocs/op | [cmap](https://github.com/fwhezfwhez/cmap/blob/aabf39042164d251011b20273a2ccba7639df915/map_test.go#L227) |
| sync.map | 500000 | 4100 ns/op | 6464 B/op | 42 allocs/op | [sync.map](https://github.com/fwhezfwhez/cmap/blob/aabf39042164d251011b20273a2ccba7639df915/map_test.go#L242) |
| chan-map | 300000 | 6186 ns/op | 7164 B/op | 51 allocs/op | [chan-map]() |

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

## Auto-generate
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
// You might put this auto-genereted file to the package where Teacher{} is defined.
// How to use this auto-generate concurrently-safe TeacherMap?
/*
   var teacher Teacher
   m := NewTeacherMap()
   m.Set(fmt.Sprintf("%d", teacher.TeacherId), teacher)
   _ = m.Get(fmt.Sprintf("%d", teacher.TeacherId))
   m.Delete(fmt.Sprintf("%d", teacher.TeacherId))
   m.SetEx(fmt.Sprintf("%d", teacher.TeacherId), teacher, 3*24*60*60)
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
// value saved
type TeacherValue struct {
	// value
	v Teacher
	// value will be expired in exp seconds, -1 means no time limit
	exp int64

	// Map's offset, when map exec set/delete, offset++
	offset int64
	// generated time when a value is set, unixnano
	execAt int64
}

// v is latter than v2 in time
// LatterThan helps judge set/del/sync make sense or not
func (v TeacherValue) LatterThan(v2 TeacherValue) bool {
	if v.execAt > v2.execAt {
		return true
	}

	if v.execAt < v2.execAt {
		return false
	}

	return v.offset > v2.offset
}

// v is former than v2 in time
func (v TeacherValue) FormerThan(v2 TeacherValue) bool {
	if v.execAt < v2.execAt {
		return true
	}
	if v.execAt > v2.execAt {
		return false
	}
	return v.offset < v2.offset
}

// make value readable
func (v TeacherValue) detail() map[string]interface{} {
	m := map[string]interface{}{
		"v":       v.v,
		"exp":     v.exp,
		"offset":  v.offset,
		"exec_at": v.execAt,
	}
	return m
}

// M_FREE and M_BUSY are signal of map.m state.
// When map.mode == M_FREE, data are writable and readable using map.m.
// When map.mode == M_BUSY, data are writable and readble using map.dirty, writable map.write and writable map.del
const (
	TEACHER_M_FREE = 0
	TEACHER_M_BUSY = 1
)

// Map is concurrently safe, and faster than sync.Map.
// Map consist of m, dirty, write, del, these 4 data register.
// In M_FREE mode, map.m is writable/readable to users.
// When calling map.ClearExpireKeys(), map will change mode into M_BUSY.
// In B_BUSY mode, map.m is unreadable and unwritable. users read data from dirty.
// At this moment,
//    writing operation will write data to map.dirty, map.write
//    deleting operation will write to map.del
// As soon as clearing job done, mode set to M_FREE,
// At this moment,
//    map.m is return to job, writed and read
//    map.write change to read-only and data in m.write will be sync to map.m then cleared
//    map.del change to read-only and data in m.del will sync to map.m then cleared
//    map.dirty are first be cleared and then synchronizing data from map.m and block new write request, after sync job done, finish all writing request.
type TeacherMap struct {
	mode int

	l *sync.RWMutex
	m map[string]TeacherValue

	dl    *sync.RWMutex
	dirty map[string]TeacherValue

	wl    *sync.RWMutex
	write map[string]TeacherValue

	dll    *sync.RWMutex
	offset int64
	del    map[string]TeacherValue
}

// Help viewing map's detail.
//   b, e:= json.MarshalIndent(m.Detail(), "", "  ")
//   fmt.Println(string(b))
type teacherMapView struct {
	Mode int

	M map[string]interface{}

	Dirty map[string]interface{}

	Write map[string]interface{}

	Offset int64
	Del    map[string]interface{}
}

func newTeacherMap() *TeacherMap {
	return &TeacherMap{
		mode: TEACHER_M_FREE,

		l: &sync.RWMutex{},
		m: make(map[string]TeacherValue),

		dl:    &sync.RWMutex{},
		dirty: make(map[string]TeacherValue),

		wl:    &sync.RWMutex{},
		write: make(map[string]TeacherValue),

		dll: &sync.RWMutex{},
		del: make(map[string]TeacherValue),
	}
}

// new a concurrent map
func NewTeacherMap() *TeacherMap {
	return newTeacherMap()
}

// Pause the world
// gLock and gUnlock must run in pair
func (m *TeacherMap) gLock() {
	m.l.Lock()
	m.wl.Lock()
	m.dl.Lock()
	m.dll.Lock()
}

// Continue
func (m *TeacherMap) gUnlock() {
	m.l.Unlock()
	m.wl.Unlock()
	m.dl.Unlock()
	m.dll.Unlock()
}

func (m *TeacherMap) gRLock() {
	m.l.RLock()
	m.wl.RLock()
	m.dl.RLock()
	m.dll.RLock()
}
func (m *TeacherMap) gRUnlock() {
	m.l.RUnlock()
	m.wl.RUnlock()
	m.dl.RUnlock()
	m.dll.RUnlock()
}

func (m *TeacherMap) IsBusy() bool {
	m.gRLock()

	defer m.gRUnlock()

	return m.mode == TEACHER_M_BUSY
}

// map.Set
func (m *TeacherMap) Set(key string, value Teacher) {
	ext := time.Now().UnixNano()
	offset := m.offsetIncr()

	// only when not busy, m is writable
	if !m.IsBusy() {
		m.l.Lock()

		m.m[key] = TeacherValue{
			v:   value,
			exp: -1,

			execAt: ext,
			offset: offset,
		}
		m.l.Unlock()

	}

	m.dl.Lock()
	m.dirty[key] = TeacherValue{
		v:      value,
		exp:    -1,
		execAt: ext,
		offset: offset,
	}
	m.dl.Unlock()

	// only when m is busy, write map is writable
	if m.IsBusy() {
		m.wl.Lock()
		m.write[key] = TeacherValue{
			v:   value,
			exp: -1,

			execAt: ext,
			offset: offset,
		}
		m.wl.Unlock()
	}
}

// set,del,setnx,setex will increase map.offset.
// When offset reaches max int64 value, will be back to 0
// So to judege values former or latter, should compare v.execAt first and then comapre offset.
func (m *TeacherMap) offsetIncr() int64 {
	atomic.AddInt64(&m.offset, 1)
	atomic.CompareAndSwapInt64(&m.offset, math.MaxInt64, 0)
	return m.offset
}

// map.SetEX
// key-value will be put with expired time limit.
// If seconds are set -1, value will not be expired
// expired keys will be deleted as soon as calling m.Get(key), or calling m.ClearExpireKeys()
func (m *TeacherMap) SetEx(key string, value Teacher, seconds int) {
	ext := time.Now().UnixNano()
	offset := m.offsetIncr()

	// only when not busy, m is writable
	if !m.IsBusy() {
		m.l.Lock()

		m.m[key] = TeacherValue{
			v:   value,
			exp: time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),

			execAt: ext,
			offset: offset,
		}
		m.l.Unlock()

	}

	m.dl.Lock()

	m.dirty[key] = TeacherValue{
		v:      value,
		exp:    time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),
		execAt: ext,
		offset: offset,
	}
	m.dl.Unlock()

	// only when m is busy, write map is writable
	if m.IsBusy() {
		m.wl.Lock()
		m.write[key] = TeacherValue{
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
func (m *TeacherMap) SetNx(key string, value Teacher) {
	m.SetExNx(key, value, -1)
}

// map.SetEXNX
// If key exist, do nothing, otherwise set key,value into map
func (m *TeacherMap) SetExNx(key string, value Teacher, seconds int) {
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
func (m *TeacherMap) Get(key string) Teacher {
	if !m.IsBusy() {
		return teacherGetFrom(m.l, m.m, key)
	} else {
		return teacherGetFrom(m.dl, m.dirty, key)
	}
}

// Delete
func (m *TeacherMap) Delete(key string) {
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
		m.del[key] = TeacherValue{
			exp: 0,
			v:   Teacher{},

			execAt: ext,
			offset: offset,
		}
		m.dll.Unlock()
	}
}

func (m *TeacherMap) Detail() teacherMapView {
	var listMaxNum = 10
	var flag = 0

	mv := teacherMapView{
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
			mv.M["reach-max-detail"] = Teacher{}
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
			mv.Dirty["reach-max-detail"] = Teacher{}
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
			mv.Write["reach-max-detail"] = Teacher{}
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
			mv.Del["reach-max-detail"] = Teacher{}
			break
		}
	}
	m.dll.RUnlock()
	return mv
}

func (m *TeacherMap) PrintDetail() string {
	b, e := json.MarshalIndent(m.Detail(), "", "  ")
	if e != nil {
		fmt.Println(e.Error())
		return ""
	}
	fmt.Println(string(b))
	return string(b)
}

func teacherGetFrom(l *sync.RWMutex, m map[string]TeacherValue, key string) Teacher {
	l.RLock()
	value, ok := m[key]
	l.RUnlock()

	if !ok {
		return Teacher{}
	}

	if value.exp == -1 {
		return value.v
	}

	if time.Now().UnixNano() >= value.exp {
		l.Lock()
		delete(m, key)
		l.Unlock()
		return Teacher{}
	}

	return value.v
}

func (m *TeacherMap) setBusy() {
	m.gLock()
	m.mode = TEACHER_M_BUSY
	m.gUnlock()
}

func (m *TeacherMap) setFree() {
	m.gLock()
	m.mode = TEACHER_M_FREE
	m.gUnlock()
}

// Returns Map.m real length, not m.dirty or m.write.
func (m *TeacherMap) Len() int {
	m.l.RLock()
	length := len(m.m)
	m.l.RUnlock()
	return length
}

// ClearExpireKeys clear expired keys, and it will not influence map write and read.
// When call m.ClearExpireKeys(), first will set m.mode=M_BUSY.
// At this moment, operation of write to Map.m is denied and instead data will be writen to Map.write which will sync to Map.m after clear job done.
// operation of read will use Map.dirty.
// After clear job has been done, Map.dirty will be cleared and copy from Map.m, Map.write will be unwritenable and data in Map.write will sync to Map.m.
func (m *TeacherMap) ClearExpireKeys() int {
	if m.IsBusy() {
		// on clearing job, another clear job do nothing
		// returned cleared key number is not concurrently consistent, because m.mode is not considered locked.
		return 0
	}
	// clear Map.m expired keys
	n := m.clearExpireKeys()

	// sync new writen data from Map.write
	var tmp = make(map[string]TeacherValue)
	m.wl.Lock()
	for k, v := range m.write {
		tmp[k] = v
	}
	m.write = make(map[string]TeacherValue)
	m.wl.Unlock()

	m.l.Lock()
	for k, v := range tmp {
		m.m[k] = v
	}
	m.l.Unlock()

	// sync deleted operation from Map.del
	tmp = make(map[string]TeacherValue)

	m.dll.Lock()

	for k, v := range m.del {
		tmp[k] = v
	}
	m.del = make(map[string]TeacherValue)
	m.dll.Unlock()

	m.l.Lock()
	for k, v := range tmp {
		v2, ok := m.m[k]
		if !ok {
			continue
		}
		if v.LatterThan(v2) {
			delete(m.m, k)
		}
	}
	m.l.Unlock()

	tmp = make(map[string]TeacherValue)

	m.l.RLock()
	for k, v := range m.m {
		tmp[k] = v
	}
	m.l.RUnlock()

	// When migrating data from m to drity, since while tmp is coping from m.m, m.m and m.dirty are still writable, tmp is relatively old.
	// So make sure data from old m.m's copied tmp will not influence latest writened data in dirty.
	m.dl.Lock()
	m.dirty = make(map[string]TeacherValue)
	for k, v := range tmp {
		v2, ok := m.dirty[k]
		if ok && v2.LatterThan(v) {
			// make sure existed latest writen data will not be replaced by the old
			continue
		} else {
			// replace old data
			m.dirty[k] = v
		}
	}
	m.dl.Unlock()

	return n
}

// Change to busy mode, now dirty provides read, write provides write, del provides delete.
// After clear expired keys in m, m will change into free auto..ly.
// in free mode, m.del and m.write will not provides read nor write, m.dirty will not read.
func (m *TeacherMap) clearExpireKeys() int {
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
