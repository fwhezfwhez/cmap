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

// Auto-generate by github.com/fwhezfwhez/cmap.GenerateTypeSyncMap()
// Note:
// You might put this auto-genereted file to the package where ${Model}{} is defined.
// How to use this auto-generate concurrently-safe ${Model}Map?
/*
    var ${model} ${Model}
    m := New${Model}Map()
    m.Set(fmt.Sprintf("%d", ${model}.${Model}Id), ${model})
    _ = m.Get(fmt.Sprintf("%d", ${model}.${Model}Id))
    m.Delete(fmt.Sprintf("%d", ${model}.${Model}Id))
    m.SetEx(fmt.Sprintf("%d", ${model}.${Model}Id), ${model}, 3*24*60*60)
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
type ${Model}Value struct {
	// value
	v ${Model}
	// value will be expired in exp seconds, -1 means no time limit
	exp int64

	// Map's offset, when map exec set/delete, offset++
	offset int64
	// generated time when a value is set, unixnano
	execAt int64
}

// v is latter than v2 in time
// LatterThan helps judge set/del/sync make sense or not
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

// make value readable
func (v ${Model}Value) detail() map[string]interface{} {
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
	${MODEL}_M_FREE = 0
	${MODEL}_M_BUSY = 1
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
//   b, e:= json.MarshalIndent(m.Detail(), "", "  ")
//   fmt.Println(string(b))
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

		l: &sync.RWMutex{},
		m: make(map[string]${Model}Value),

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

// Pause the world
// gLock and gUnlock must run in pair
func (m *${Model}Map) gLock() {
	m.l.Lock()
	m.wl.Lock()
	m.dl.Lock()
	m.dll.Lock()
}

// Continue
func (m *${Model}Map) gUnlock() {
	m.l.Unlock()
	m.wl.Unlock()
	m.dl.Unlock()
	m.dll.Unlock()
}

func (m *${Model}Map) gRLock() {
	m.l.RLock()
	m.wl.RLock()
	m.dl.RLock()
	m.dll.RLock()
}
func (m *${Model}Map) gRUnlock() {
	m.l.RUnlock()
	m.wl.RUnlock()
	m.dl.RUnlock()
	m.dll.RUnlock()
}

func (m *${Model}Map) IsBusy() bool {
	m.gRLock()

	defer m.gRUnlock()

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

// set,del,setnx,setex will increase map.offset.
// When offset reaches max int64 value, will be back to 0
// So to judege values former or latter, should compare v.execAt first and then comapre offset.
func (m *${Model}Map) offsetIncr() int64 {
	atomic.AddInt64(&m.offset, 1)
	atomic.CompareAndSwapInt64(&m.offset, math.MaxInt64, 0)
	return m.offset
}

// map.SetEX
// key-value will be put with expired time limit.
// If seconds are set -1, value will not be expired
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
			v:   ${Model}{},

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
			mv.M["reach-max-detail"] = ${Model}{}
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
			mv.Dirty["reach-max-detail"] = ${Model}{}
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
			mv.Write["reach-max-detail"] =${Model}{}
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
			mv.Del["reach-max-detail"] = ${Model}{}
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
	m.gLock()
	m.mode = ${MODEL}_M_BUSY
	m.gUnlock()
}

func (m *${Model}Map) setFree() {
	m.gLock()
	m.mode = ${MODEL}_M_FREE
	m.gUnlock()
}

// Returns Map.m real length, not m.dirty or m.write.
func (m *${Model}Map) Len() int {
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
func (m *${Model}Map) ClearExpireKeys() int {
	if m.IsBusy() {
		// on clearing job, another clear job do nothing
		// returned cleared key number is not concurrently consistent, because m.mode is not considered locked.
		return 0
	}
	// clear Map.m expired keys
	n := m.clearExpireKeys()

	// sync new writen data from Map.write
	var tmp = make(map[string]${Model}Value)
	m.wl.Lock()
	for k, v := range m.write {
		tmp[k] = v
	}
	m.write = make(map[string]${Model}Value)
	m.wl.Unlock()

	m.l.Lock()
	for k, v := range tmp {
		m.m[k] = v
	}
	m.l.Unlock()

	// sync deleted operation from Map.del
	tmp = make(map[string]${Model}Value)

	m.dll.Lock()

	for k, v := range m.del {
		tmp[k] = v
	}
	m.del = make(map[string]${Model}Value)
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

	tmp = make(map[string]${Model}Value)

	m.l.RLock()
	for k, v := range m.m {
		tmp[k] = v
	}
	m.l.RUnlock()

	// When migrating data from m to drity, since while tmp is coping from m.m, m.m and m.dirty are still writable, tmp is relatively old.
	// So make sure data from old m.m's copied tmp will not influence latest writened data in dirty.
	m.dl.Lock()
	m.dirty = make(map[string]${Model}Value)
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
