package cmap

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type Value struct {
	// value
	v interface{}
	// expire_in, -1-no time limit
	exp int64

	// Map's offset, when map exec set/delete offset++
	offset int64
	// exec time unixnano
	execAt int64
}

// v is latter than v2 in time
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

func (v Value) detail() string {
	m := map[string]interface{}{
		"v":       v.v,
		"exp":     v.exp,
		"offset":  v.offset,
		"exec_at": v.execAt,
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return string(b)
}

const (
	M_FREE = 0
	M_BUSY = 1
)

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
		l:    &sync.RWMutex{},
		m:    make(map[string]Value),

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
func (m *Map) GLock() {
	m.l.Lock()
	m.wl.Lock()
	m.dl.Lock()
	m.dll.Lock()
}
func (m *Map) GUnlock() {
	m.l.Unlock()
	m.wl.Unlock()
	m.dl.Unlock()
	m.dll.Unlock()
}

// map.Set
func (m *Map) Set(key string, value interface{}) {
	ext := time.Now().UnixNano()
	offset := m.offsetIncr()

	// only when not busy, m is writable
	if m.mode != M_BUSY {
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
	if m.mode == M_BUSY {
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

func (m *Map) offsetIncr() int64 {
	atomic.AddInt64(&m.offset, 1)
	atomic.CompareAndSwapInt64(&m.offset, math.MaxInt64, 0)
	return m.offset
}

// map.SetEX
// key-value will be put with expired time limit.
// expired keys will be deleted as soon as calling m.Get(key), or calling m.ClearExpireKeys()
func (m *Map) SetEx(key string, value interface{}, seconds int) {
	ext := time.Now().UnixNano()
	offset := m.offsetIncr()

	// only when not busy, m is writable
	if m.mode != M_BUSY {
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
	if m.mode == M_BUSY {
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
	if m.mode == M_BUSY {
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
func (m *Map) Get(key string) interface{} {
	if m.mode != M_BUSY {
		return getFrom(m.l, m.m, key)
	} else {
		return getFrom(m.dl, m.dirty, key)
	}
}

// Delete
func (m *Map) Delete(key string) {
	offset := m.offsetIncr()
	ext := time.Now().UnixNano()
	if m.mode == M_FREE {
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

	if m.mode == M_BUSY {
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

func (m *Map) Detail() string {
	m.GLock()
	defer m.GUnlock()

	var listMaxNum = 10
	var flag = 0

	mv := mapView{
		M:     make(map[string]interface{}),
		Dirty: make(map[string]interface{}),
		Del:   make(map[string]interface{}),
		Write: make(map[string]interface{}),
	}
	mv.Offset = m.offset

	for k, _ := range m.m {
		flag++
		mv.M[k] = m.m[k].detail()
		if flag > listMaxNum {
			mv.M["reach-max-detail"] = "end"
			break
		}
	}
	flag = 0
	for k, _ := range m.dirty {
		mv.Dirty[k] = m.dirty[k].detail()
		flag++
		if flag > listMaxNum {
			mv.Dirty["reach-max-detail"] = "end"
			break
		}
	}

	flag = 0
	for k, _ := range m.write {
		mv.Write[k] = m.write[k].detail()
		flag++
		if flag > listMaxNum {
			mv.Write["reach-max-detail"] = "end"
			break
		}
	}

	flag = 0
	for k, _ := range m.del {
		mv.Del[k] = m.del[k].detail()
		flag++
		if flag > listMaxNum {
			mv.Del["reach-max-detail"] = "end"
			break
		}
	}

	b, e := json.MarshalIndent(mv, "", "  ")
	if e != nil {
		fmt.Println(e.Error())
		return ""
	}
	return string(b)
}

func getFrom(l *sync.RWMutex, m map[string]Value, key string) interface{} {
	l.RLock()
	value, ok := m[key]
	l.RUnlock()

	if !ok {
		return nil
	}

	if value.exp == -1 {
		return value.v
	}

	if time.Now().UnixNano() >= value.exp {
		l.Lock()
		delete(m, key)
		l.Unlock()
		return nil
	}

	return value.v
}

func (m *Map) setBusy() {
	m.l.Lock()
	defer m.l.Unlock()
	m.dl.Lock()
	defer m.dl.Unlock()
	m.wl.Lock()
	defer m.wl.Unlock()

	m.mode = M_BUSY
}

func (m *Map) setFree() {
	m.l.Lock()
	defer m.l.Unlock()
	m.dl.Lock()
	defer m.dl.Unlock()
	m.wl.Lock()
	defer m.wl.Unlock()

	m.mode = M_FREE
}

// Returns Map.m real length, not m.dirty or m.write.
func (m *Map) Len() int {
	m.l.RLock()
	defer m.l.RUnlock()
	return len(m.m)
}

// ClearExpireKeys clear expired keys, and it will not influence map write and read.
// When call m.ClearExpireKeys(), first will set m.mode=M_BUSY.
// At this moment, operation of write to Map.m is denied and instead data will be writen to Map.write which will sync to Map.m after clear job done.
// operation of read will use Map.dirty.
// After clear job has been done, Map.dirty will be cleared and copy from Map.m, Map.write will be unwritenable and data in Map.write will sync to Map.m.
func (m *Map) ClearExpireKeys() int {
	if m.mode == M_BUSY {
		// on clearing job, another clear job do nothing
		// returned cleared key number is not concurrently consistent, because m.mode is not considered locked.
		return 0
	}

	n := m.clearExpireKeys()

	m.dl.Lock()
	m.dirty = make(map[string]Value)

	m.l.RLock()
	for key, _ := range m.m {
		m.dirty[key] = m.m[key]
	}
	m.l.RUnlock()
	m.dl.Unlock()

	return n
}
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

	m.wl.RLock()
	for k, v := range m.write {
		m.m[k] = v
	}
	m.wl.RUnlock()

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
	m.dll.Unlock()

	m.l.Unlock()

	m.wl.Lock()
	m.write = make(map[string]Value)
	m.wl.Unlock()

	m.dll.Lock()
	m.del = make(map[string]Value)
	m.dll.Unlock()

	return num
}
