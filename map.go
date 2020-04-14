package cmap

import (
	"sync"
	"time"
)

type Value struct {
	// value
	v interface{}
	// expire_in, -1-no time limit
	exp int64
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
}
func (m *Map) GUnlock() {
	m.l.Unlock()
	m.wl.Unlock()
	m.dl.Unlock()
}

// map.Set
func (m *Map) Set(key string, value interface{}) {

	// only when not busy, m is writable
	if m.mode != M_BUSY {
		m.l.Lock()

		m.m[key] = Value{
			v:   value,
			exp: -1,
		}
		m.l.Unlock()

	}

	m.dl.Lock()
	m.dirty[key] = Value{
		v:   value,
		exp: -1,
	}
	m.dl.Unlock()

	// only when m is busy, write map is writable
	if m.mode == M_BUSY {
		m.wl.Lock()
		m.write[key] = Value{
			v:   value,
			exp: -1,
		}
		m.wl.Unlock()
	}
}

// map.SetEX
// key-value will be put with expired time limit.
// expired keys will be deleted as soon as calling m.Get(key), or calling m.ClearExpireKeys()
func (m *Map) SetEx(key string, value interface{}, seconds int) {
	// only when not busy, m is writable
	if m.mode != M_BUSY {
		m.l.Lock()

		m.m[key] = Value{
			v:   value,
			exp: time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),
		}
		m.l.Unlock()

	}

	m.dl.Lock()

	m.dirty[key] = Value{
		v:   value,
		exp: time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),
	}
	m.dl.Unlock()

	// only when m is busy, write map is writable
	if m.mode == M_BUSY {
		m.wl.Lock()
		m.write[key] = Value{
			v:   value,
			exp: time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),
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

	m.l.Unlock()

	m.wl.Lock()
	m.write = make(map[string]Value)
	m.wl.Unlock()
	return num
}
