package cmap

import "strings"

func GenerateTypeSyncMap(Model string, args map[string]string) string {
	args["${Model}"] = Model
	initArgs(args)

	format := `
package ${package_name}

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type ${Model}Value struct {
	//
	IsNil bool
	// value
	v ${Model}
	// expire_in, -1-no time limit
	exp int64

	// Map's offset, when map exec set/delete offset++
	offset int64
	// exec time unixnano
	execAt int64
}

// v is latter than v2 in time
func (v ${Model}Value) LatterThan(v2 ${Model}Value) bool {
	if v.execAt > v2.execAt {
		return true
	}

	if v.execAt < v2.execAt {
		return false
	}

	return v.offset > v2.offset
}

// v is former than v2 in time
func (v ${Model}Value) FormerThan(v2 ${Model}Value) bool {
	if v.execAt < v2.execAt {
		return true
	}
	if v.execAt > v2.execAt {
		return false
	}
	return v.offset < v2.offset
}

func (v ${Model}Value) detail() map[string]interface{} {
	m := map[string]interface{}{
		"v":       v.v,
		"exp":     v.exp,
		"offset":  v.offset,
		"exec_at": v.execAt,
	}
	return m
}

const (
	${MODEL}_M_FREE = 0
	${MODEL}_M_BUSY = 1
)

type ${Model}Map struct {
	mode int

	l *sync.RWMutex
	m map[string]${Model}Value

	dl    *sync.RWMutex
	dirty map[string]${Model}Value

	wl    *sync.RWMutex
	write map[string]${Model}Value

	dll    *sync.RWMutex
	offset int64
	del    map[string]${Model}Value
}

// Help viewing map's detail.
type ${model}MapView struct {
	Mode int

	M map[string]interface{}

	Dirty map[string]interface{}

	Write map[string]interface{}

	Offset int64
	Del    map[string]interface{}
}

func new${Model}Map() *${Model}Map {
	return &${Model}Map{
		mode: ${MODEL}_M_FREE,
		l:    &sync.RWMutex{},
		m:    make(map[string]${Model}Value),

		dl:    &sync.RWMutex{},
		dirty: make(map[string]${Model}Value),

		wl:    &sync.RWMutex{},
		write: make(map[string]${Model}Value),

		dll: &sync.RWMutex{},
		del: make(map[string]${Model}Value),
	}
}

// new a concurrent map
func New${Model}Map() *${Model}Map {
	return new${Model}Map()
}
func (m *${Model}Map) GLock() {
	m.l.Lock()
	m.wl.Lock()
	m.dl.Lock()
	m.dll.Lock()
}
func (m *${Model}Map) GUnlock() {
	m.l.Unlock()
	m.wl.Unlock()
	m.dl.Unlock()
	m.dll.Unlock()
}

func (m *${Model}Map) GRLock() {
	m.l.RLock()
	m.wl.RLock()
	m.dl.RLock()
	m.dll.RLock()
}
func (m *${Model}Map) GRUnlock() {
	m.l.RUnlock()
	m.wl.RUnlock()
	m.dl.RUnlock()
	m.dll.RUnlock()
}

func (m *${Model}Map) IsBusy() bool {
	m.GRLock()

	defer m.GRUnlock()

	return m.mode == ${MODEL}_M_BUSY
}

// map.Set
func (m *${Model}Map) Set(key string, value ${Model}) {
	ext := time.Now().UnixNano()
	offset := m.offsetIncr()

	// only when not busy, m is writable
	if !m.IsBusy() {
		m.l.Lock()

		m.m[key] = ${Model}Value{
			v:   value,
			exp: -1,

			execAt: ext,
			offset: offset,
		}
		m.l.Unlock()

	}

	m.dl.Lock()
	m.dirty[key] = ${Model}Value{
		v:      value,
		exp:    -1,
		execAt: ext,
		offset: offset,
	}
	m.dl.Unlock()

	// only when m is busy, write map is writable
	if m.IsBusy() {
		m.wl.Lock()
		m.write[key] = ${Model}Value{
			v:   value,
			exp: -1,

			execAt: ext,
			offset: offset,
		}
		m.wl.Unlock()
	}
}

func (m *${Model}Map) offsetIncr() int64 {
	atomic.AddInt64(&m.offset, 1)
	atomic.CompareAndSwapInt64(&m.offset, math.MaxInt64, 0)
	return m.offset
}

// map.SetEX
// key-value will be put with expired time limit.
// expired keys will be deleted as soon as calling m.Get(key), or calling m.ClearExpireKeys()
func (m *${Model}Map) SetEx(key string, value ${Model}, seconds int) {
	ext := time.Now().UnixNano()
	offset := m.offsetIncr()

	// only when not busy, m is writable
	if !m.IsBusy() {
		m.l.Lock()

		m.m[key] = ${Model}Value{
			v:   value,
			exp: time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),

			execAt: ext,
			offset: offset,
		}
		m.l.Unlock()

	}

	m.dl.Lock()

	m.dirty[key] = ${Model}Value{
		v:      value,
		exp:    time.Now().Add(time.Duration(seconds) * time.Second).UnixNano(),
		execAt: ext,
		offset: offset,
	}
	m.dl.Unlock()

	// only when m is busy, write map is writable
	if m.IsBusy() {
		m.wl.Lock()
		m.write[key] = ${Model}Value{
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
func (m *${Model}Map) SetNx(key string, value ${Model}) {
	m.SetExNx(key, value, -1)
}

// map.SetEXNX
// If key exist, do nothing, otherwise set key,value into map
func (m *${Model}Map) SetExNx(key string, value ${Model}, seconds int) {
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
func (m *${Model}Map) Get(key string) ${Model} {
	if !m.IsBusy() {
		return ${model}GetFrom(m.l, m.m, key)
	} else {
		return ${model}GetFrom(m.dl, m.dirty, key)
	}
}

// Delete
func (m *${Model}Map) Delete(key string) {
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
		m.del[key] = ${Model}Value{
			exp: 0,
			v:  ${Model}{},

			execAt: ext,
			offset: offset,
		}
		m.dll.Unlock()
	}
}

func (m *${Model}Map) Detail() ${model}MapView {
	var listMaxNum = 10
	var flag = 0

	mv := ${model}MapView{
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

func (m *${Model}Map) PrintDetail() string {
	b, e := json.MarshalIndent(m.Detail(), "", "  ")
	if e != nil {
		fmt.Println(e.Error())
		return ""
	}
	fmt.Println(string(b))
	return string(b)
}

func ${model}GetFrom(l *sync.RWMutex, m map[string]${Model}Value, key string) ${Model} {
	l.RLock()
	value, ok := m[key]
	l.RUnlock()

	if !ok {
		return ${Model}{}
	}

	if value.exp == -1 {
		return value.v
	}

	if time.Now().UnixNano() >= value.exp {
		l.Lock()
		delete(m, key)
		l.Unlock()
		return ${Model}{}
	}

	return value.v
}

func (m *${Model}Map) setBusy() {
	m.l.Lock()
	defer m.l.Unlock()
	m.dl.Lock()
	defer m.dl.Unlock()
	m.wl.Lock()
	defer m.wl.Unlock()

	m.mode = ${MODEL}_M_BUSY
}

func (m *${Model}Map) setFree() {
	m.l.Lock()
	defer m.l.Unlock()
	m.dl.Lock()
	defer m.dl.Unlock()
	m.wl.Lock()
	defer m.wl.Unlock()

	m.mode = ${MODEL}_M_FREE
}

// Returns Map.m real length, not m.dirty or m.write.
func (m *${Model}Map) Len() int {
	m.l.RLock()
	defer m.l.RUnlock()
	return len(m.m)
}

// ClearExpireKeys clear expired keys, and it will not influence map write and read.
// When call m.ClearExpireKeys(), first will set m.mode=M_BUSY.
// At this moment, operation of write to Map.m is denied and instead data will be writen to Map.write which will sync to Map.m after clear job done.
// operation of read will use Map.dirty.
// After clear job has been done, Map.dirty will be cleared and copy from Map.m, Map.write will be unwritenable and data in Map.write will sync to Map.m.
func (m *${Model}Map) ClearExpireKeys() int {
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
	m.write = make(map[string]${Model}Value)
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
	m.del = make(map[string]${Model}Value)
	m.dll.Unlock()

	m.l.Unlock()

	m.dl.Lock()
	m.dirty = make(map[string]${Model}Value)

	m.l.RLock()
	for key, _ := range m.m {
		m.dirty[key] = m.m[key]
	}
	m.l.RUnlock()
	m.dl.Unlock()

	return n
}
func (m *${Model}Map) clearExpireKeys() int {
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

	`
	return replace(format, args)
}

func initArgs(args map[string]string) {
	if len(args) == 0 {
		args = map[string]string{}
	}

	if args["${package_name}"] == "" {
		args["${package_name}"] = "model"
	}

	if args["${Model}"] == "" {
		args["${Model}"] = "Model"
	}

	if args["${MODEL}"] == "" {
		args["${MODEL}"] = strings.ToUpper(args["${Model}"])
	}
	if args["${model}"] == "" {
		args["${model}"] = strings.ToLower(args["${Model}"])
	}

}

func replace(format string, args map[string]string) string {
	var rs = format
	// rs = strings.Replace(format, "${package_name}", "model", -1)
	for k, v := range args {
		rs = strings.Replace(rs, k, v, -1)
	}
	return rs
}
