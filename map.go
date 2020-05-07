package cmap

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// value saved
type Value struct {
	// value
	v interface{}
	// value will be expired in exp seconds, -1 means no time limit
	exp int64

	// Map's offset, when map exec set/delete, offset++
	offset int64
	// generated time when a value is set, unixnano
	execAt int64
}

// v is latter than v2 in time
// LatterThan helps judge set/del/sync make sense or not
func (v Value) LatterThan(v2 Value) bool {
	if v.execAt > v2.execAt {
		return true
	}

	if v.execAt < v2.execAt {
		return false
	}

	return v.offset > v2.offset
}

// v is former than v2 in time
func (v Value) FormerThan(v2 Value) bool {
	if v.execAt < v2.execAt {
		return true
	}
	if v.execAt > v2.execAt {
		return false
	}
	return v.offset < v2.offset
}

// make value readable
func (v Value) detail() map[string]interface{} {
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
	M_FREE = 0
	M_BUSY = 1
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
type Map struct {
	mode int

	l *sync.RWMutex
	m map[string]Value

	dl    *sync.RWMutex
	dirty map[string]Value

	wl    *sync.RWMutex
	write map[string]Value

	dll    *sync.RWMutex
	offset int64
	del    map[string]Value
}

// Help viewing map's detail.
//   b, e:= json.MarshalIndent(m.Detail(), "", "  ")
//   fmt.Println(string(b))
type mapView struct {
	Mode int

	M map[string]interface{}

	Dirty map[string]interface{}

	Write map[string]interface{}

	Offset int64
	Del    map[string]interface{}
}

func newMap() *Map {
	return &Map{
		mode: M_FREE,

		l: &sync.RWMutex{},
		m: make(map[string]Value),

		dl:    &sync.RWMutex{},
		dirty: make(map[string]Value),

		wl:    &sync.RWMutex{},
		write: make(map[string]Value),

		dll: &sync.RWMutex{},
		del: make(map[string]Value),
	}
}

// new a concurrent map
func NewMap() *Map {
	return newMap()
}

// Pause the world
// gLock and gUnlock must run in pair
func (m *Map) gLock() {
	m.l.Lock()
	m.wl.Lock()
	m.dl.Lock()
	m.dll.Lock()
}

// Continue
func (m *Map) gUnlock() {
	m.dll.Unlock()
	m.dl.Unlock()
	m.wl.Unlock()
	m.l.Unlock()
}

func (m *Map) gRLock() {
	m.l.RLock()
	m.wl.RLock()
	m.dl.RLock()
	m.dll.RLock()
}
func (m *Map) gRUnlock() {
	m.l.RUnlock()
	m.wl.RUnlock()
	m.dl.RUnlock()
	m.dll.RUnlock()
}

func (m *Map) IsBusy() bool {
	m.gRLock()

	defer m.gRUnlock()

	return m.mode == M_BUSY
}

// map.Set
func (m *Map) Set(key string, value interface{}) {
	ext := time.Now().UnixNano()
	offset := m.offsetIncr()

	// only when not busy, m is writable
	if !m.IsBusy() {
		m.l.Lock()

		m.m[key] = Value{
			v:   value,
			exp: -1,

			execAt: ext,
			offset: offset,
		}
		m.l.Unlock()

	}

	m.dl.Lock()
	m.dirty[key] = Value{
		v:      value,
		exp:    -1,
		execAt: ext,
		offset: offset,
	}
	m.dl.Unlock()

	// only when m is busy, write map is writable
	if m.IsBusy() {
		m.wl.Lock()
		m.write[key] = Value{
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
func (m *Map) offsetIncr() int64 {
	atomic.CompareAndSwapInt64(&m.offset, math.MaxInt64, 0)
	atomic.AddInt64(&m.offset, 1)
	return m.offset
}

// map.SetEX
// key-value will be put with expired time limit.
// If seconds are set -1, value will not be expired
// expired keys will be deleted as soon as calling m.Get(key), or calling m.ClearExpireKeys()
func (m *Map) SetEx(key string, value interface{}, seconds int) {
	ext := time.Now().UnixNano()
	offset := m.offsetIncr()

	// only when not busy, m is writable
	if !m.IsBusy() {
		m.l.Lock()

		m.m[key] = Value{
			v:   value,
			exp: time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),

			execAt: ext,
			offset: offset,
		}
		m.l.Unlock()

	}

	m.dl.Lock()

	m.dirty[key] = Value{
		v:      value,
		exp:    time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),
		execAt: ext,
		offset: offset,
	}
	m.dl.Unlock()

	// only when m is busy, write map is writable
	if m.IsBusy() {
		m.wl.Lock()
		m.write[key] = Value{
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
func (m *Map) SetNx(key string, value interface{}) {
	m.SetExNx(key, value, -1)
}

// map.SetEXNX
// If key exist, do nothing, otherwise set key,value into map
func (m *Map) SetExNx(key string, value interface{}, seconds int) {
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
func (m *Map) Get(key string) (interface{}, bool) {
	if !m.IsBusy() {
		return getFrom(m.l, m.m, key)
	} else {
		return getFrom(m.dl, m.dirty, key)
	}
}

// Delete
func (m *Map) Delete(key string) {
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
		m.del[key] = Value{
			exp: 0,
			v:   nil,

			execAt: ext,
			offset: offset,
		}
		m.dll.Unlock()
	}
}

func (m *Map) Detail() mapView {
	var listMaxNum = 10
	var flag = 0

	mv := mapView{
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

func (m *Map) PrintDetail() string {
	b, e := json.MarshalIndent(m.Detail(), "", "  ")
	if e != nil {
		fmt.Println(e.Error())
		return ""
	}
	fmt.Println(string(b))
	return string(b)
}

func getFrom(l *sync.RWMutex, m map[string]Value, key string) (interface{}, bool) {
	l.RLock()
	value, ok := m[key]
	l.RUnlock()

	if !ok {
		return nil, false
	}

	if value.exp == -1 {
		return value.v, true
	}

	if time.Now().UnixNano() >= value.exp {
		l.Lock()
		delete(m, key)
		l.Unlock()
		return nil, false
	}

	return value.v, true
}

func (m *Map) setBusy() {
	m.gLock()
	m.mode = M_BUSY
	m.gUnlock()
}

func (m *Map) setFree() {
	m.gLock()
	m.mode = M_FREE
	m.gUnlock()
}

// Returns Map.m real length, not m.dirty or m.write.
func (m *Map) Len() int {
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
func (m *Map) ClearExpireKeys() int {
	if m.IsBusy() {
		// on clearing job, another clear job do nothing
		// returned cleared key number is not concurrently consistent, because m.mode is not considered locked.
		return 0
	}
	// clear Map.m expired keys
	n := m.clearExpireKeys()

	m.l.Lock()

	m.wl.RLock()
	for k, v := range m.write {
		v2, ok := m.m[k]
		if !ok {
			m.m[k] = v
		} else {
			if v2.LatterThan(v) {
				continue
			}
		}
	}
	m.wl.RUnlock()

	m.wl.Lock()
	m.write = make(map[string]Value)
	m.wl.Unlock()

	m.l.Unlock()

	// sync deleted operation from Map.del
	m.l.Lock()

	m.dll.Lock()
	for k, v := range m.del {
		v2, ok := m.m[k]
		if !ok {
			continue
		}
		if v.LatterThan(v2) {
			delete(m.m, k)
		}
	}

	m.del = make(map[string]Value)
	m.dll.Unlock()

	m.l.Unlock()

	// When migrating data from m to drity, since while tmp is coping from m.m, m.m and m.dirty are still writable, tmp is relatively old.
	// So make sure data from old m.m's copied tmp will not influence latest writened data in dirty.
	m.l.RLock()

	m.dl.Lock()
	m.dirty = make(map[string]Value)
	for k, v := range m.m {
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

	m.l.RUnlock()
	return n
}

// Change to busy mode, now dirty provides read, write provides write, del provides delete.
// After clear expired keys in m, m will change into free auto..ly.
// in free mode, m.del and m.write will not provides read nor write, m.dirty will not read.
func (m *Map) clearExpireKeys() int {
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

// MLen
func (m *Map) MLen() int {
	m.l.RLock()
	leng := len(m.m)
	m.l.RUnlock()
	return leng
}
