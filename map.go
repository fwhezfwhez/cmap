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

func (v Value) isExpire() bool {
	if v.exp == -1 {
		return false
	}

	if v.exp < time.Now().UnixNano() {
		return true
	}
	return false
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
	M_FREE1 = 2 // free1-同步中, m 同步write和del
	M_FREE2 = 0 // free2-同步完成, m可以独立承担读写
	M_BUSY  = 1
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
	clearing int32

	deltal *sync.RWMutex // 在增量复制时的锁

	modl *sync.RWMutex
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
		deltal: &sync.RWMutex{},
		modl:   &sync.RWMutex{},
		mode:   M_FREE2,

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
	m.modl.RLock()
	defer m.modl.RUnlock()

	return m.mode == M_BUSY
}

func (m *Map) IsFree1() bool {
	m.modl.RLock()
	defer m.modl.RUnlock()

	return m.mode == M_FREE1
}

func (m *Map) IsFree2() bool {
	m.modl.RLock()
	defer m.modl.RUnlock()

	return m.mode == M_FREE2
}

func (m *Map) isBusyWrapedBymodl() bool {
	return m.mode == M_BUSY
}

func (m *Map) isFree1WrapedBymodl() bool {
	return m.mode == M_FREE1
}

func (m *Map) isFree2WrapedBymodl() bool {
	return m.mode == M_FREE2
}

// map.Set
func (m *Map) Set(key string, value interface{}) {
	m.set(key, value, -1, false)
}

func (m *Map) set(key string, value interface{}, seconds int, nx bool) {
	var exp int64
	if seconds == -1 {
		exp = -1
	} else {
		exp = time.Now().Add(time.Duration(seconds) * time.Second).UnixNano()
	}

	ext := time.Now().UnixNano()
	offset := m.offsetIncr()

	// 发生set时，不会出现状态切换
	m.modl.RLock()
	defer m.modl.RUnlock()

	// free2 时，写入m，写入dir
	if m.isFree2WrapedBymodl() {
		setm(m.l, m.m, key, value, ext, offset, exp, nx)

		func(ext int64, offset int64) {
			setm(m.dl, m.dirty, key, value, ext, offset, exp, nx)
		}(ext, offset)
		return
	}

	// 同步中时，写dir，阻塞m
	if m.isFree1WrapedBymodl() {
		m.deltal.RLock()
		setm(m.l, m.m, key, value, ext, offset, exp, nx)
		m.deltal.RUnlock()
		func(ext int64, offset int64) {
			setm(m.dl, m.dirty, key, value, ext, offset, exp, nx)
		}(ext, offset)
		return
	}

	// busy时，写入dir和write
	setm(m.dl, m.dirty, key, value, ext, offset, exp, nx)
	setm(m.wl, m.write, key, value, ext, offset, exp, nx)
}

// set,del,setnx,setex will increase map.offset.
// When offset reaches max int64 value, will be back to 0
// So to judege values former or latter, should compare v.execAt first and then comapre offset.
func (m *Map) offsetIncr() int64 {
	atomic.CompareAndSwapInt64(&m.offset, math.MaxInt64-10000, 0)
	atomic.AddInt64(&m.offset, 1)
	return m.offset
}

// map.SetEX
// key-value will be put with expired time limit.
// If seconds are set -1, value will not be expired
// expired keys will be deleted as soon as calling m.Get(key), or calling m.ClearExpireKeys()
func (m *Map) SetEx(key string, value interface{}, seconds int) {
	m.set(key, value, seconds, false)
}

// map.SetNx
// If key exist, do nothing, otherwise set key,value into map
func (m *Map) SetNx(key string, value interface{}) {
	m.set(key, value, -1, true)
}

// map.SetEXNX
// If key exist, do nothing, otherwise set key,value into map
func (m *Map) SetExNx(key string, value interface{}, seconds int) {
	m.set(key, value, seconds, true)
}

// If key is expired or not existed, return nil
func (m *Map) Get(key string) (interface{}, bool) {

	// Get过程中。
	// m 必须在mod保护态下，才能get
	// dirty不论何时，都可以被get
	m.modl.RLock()

	var shouldRUnlock bool = true

	defer func() {
		if shouldRUnlock {
			m.modl.RUnlock()
		}
	}()

	// m free时，读取m
	if m.isFree2WrapedBymodl() {
		return getFrom(m.l, m.m, key)
	}

	m.modl.RUnlock()
	shouldRUnlock = false
	return getFrom(m.dl, m.dirty, key)
	//// m繁忙时，读取dirty
	//if m.isBusyWrapedBymodl() {
	//	return getFrom(m.dl, m.dirty, key)
	//}
	//
	//// m 处于迁移中,读取dir
	//if m.isFree1WrapedBymodl() {
	//	return getFrom(m.dl, m.dirty, key)
	//}

	return nil, false
}

// Delete todo, bug delete fail
func (m *Map) Delete(key string) {
	offset := m.offsetIncr()
	ext := time.Now().UnixNano()

	m.modl.RLock()
	defer m.modl.RUnlock()
	if m.isFree2WrapedBymodl() {
		deletem(m.l, m.m, key, ext)

		func(ext int64) {
			//fmt.Printf("del dir %s \n", key)
			deletem(m.dl, m.dirty, key, ext)

			// deletem(m.wl, m.write, key, ext)

		}(ext)
		return
	}

	if m.isFree1WrapedBymodl() {
		// free1时，m1不能提供使用
		func() {
			m.deltal.RLock()
			deletem(m.l, m.m, key, ext)
			m.deltal.RUnlock()
		}()
		func() {
			// 	deletem(m.wl, m.write, key, ext)
			deletem(m.dl, m.dirty, key, ext)
		}()
		return
	}
	// deletem(m.wl, m.write, key, ext)

	// busy时，要删除dir, 并且追加命令进del
	deletem(m.dl, m.dirty, key, ext)

	setm(m.dll, m.del, key, "waiting-deleted", ext, offset, -1, false)
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

func (m *Map) DetailOf(key string) mapView {
	mv := mapView{
		Mode:  m.mode,
		M:     make(map[string]interface{}),
		Dirty: make(map[string]interface{}),
		Del:   make(map[string]interface{}),
		Write: make(map[string]interface{}),
	}

	m.gRLock()
	defer m.gRUnlock()

	mv.M = m.m[key].detail()
	mv.Dirty = m.dirty[key].detail()
	mv.Del = m.del[key].detail()
	mv.Write = m.write[key].detail()
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

func (m *Map) PrintDetailOf(key string) string {
	b, e := json.MarshalIndent(m.DetailOf(key), "", "  ")
	if e != nil {
		fmt.Println(e.Error())
		return ""
	}
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
	m.modl.Lock()
	defer m.modl.Unlock()
	m.mode = M_BUSY
	// m.gUnlock()
}

func (m *Map) setFree1() {
	m.modl.Lock()
	defer m.modl.Unlock()

	m.mode = M_FREE1
}

func (m *Map) setFree2() {
	m.modl.Lock()
	defer m.modl.Unlock()

	m.mode = M_FREE2
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

	// 利用atomic，确保高并发下，只会有一个ClearExpireKeys被执行
	v := atomic.AddInt32(&m.clearing, 1)
	defer atomic.AddInt32(&m.clearing, -1)

	if v != 1 {
		return 0
	}

	// 将模式切到busy下
	// 因为在读写删进行时，mod是禁切换的。
	// 所以能够切到busy,必然是读写删都在入口阻住了。
	m.setBusy()

	// 置为busy时，读写删自动放行。读取写入删除都切换到对应的模式。m此时将保持不对外不可用

	n := m.clearExpireKeys()

	m.modl.Lock()
	m.mode = M_FREE1

	m.deltal.Lock()
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

	m.l.Unlock()

	// sync deleted operation from Map.del
	m.l.Lock()

	m.dll.RLock()
	for k, v := range m.del {
		v2, ok := m.m[k]
		if !ok {
			continue
		}
		if v.LatterThan(v2) {
			delete(m.m, k)
		}
	}

	m.dll.RUnlock()

	m.l.Unlock()
	m.deltal.Unlock()

	m.modl.Unlock()

	m.setFree2()

	// 进入free2时，清理write和del,dir
	m.dll.Lock()
	m.del = make(map[string]Value)
	m.dll.Unlock()

	m.wl.Lock()
	m.write = make(map[string]Value)
	m.wl.Unlock()

	clearExpire(m.dl, m.dirty)

	return n
}

// Change to busy mode, now dirty provides read, write provides write, del provides delete.
// After clear expired keys in m, m will change into free auto..ly.
// in free mode, m.del and m.write will not provides read nor write, m.dirty will not read.
func (m *Map) clearExpireKeys() int {
	return m.clearExpireKeysWithDepth(-1)
}

// Change to busy mode, now dirty provides read, write provides write, del provides delete.
// After clear expired keys in m, m will change into free auto..ly.
// in free mode, m.del and m.write will not provides read nor write, m.dirty will not read.
func (m *Map) clearExpireKeysWithDepth(depth int) int {
	var num int

	// keys should be deleted
	var shouldDelete = make([]string, 0, len(m.m))

	m.l.RLock()

	var offset int
L:
	for k, v := range m.m {
		if v.exp == -1 {
			continue
		}
		if v.exp < time.Now().UnixNano() {
			shouldDelete = append(shouldDelete, k)

			// If hit depth of delete times, will stop range
			offset ++
			if depth > 0 && offset >= depth {
				break L
			}
		}
	}
	m.l.RUnlock()

	m.l.Lock()
	for _, v := range shouldDelete {
		delete(m.m, v)
		num++
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

// range function returns bool value
// if false,  will stop range process
func (m *Map) Range(f func(key string, value interface{}) bool) {
	if m.IsFree2() {
		rangem(m.l, m.m, f)
	} else {
		rangem(m.dl, m.dirty, f)
	}
}

// RealLength returns cmap's real length.
// If it's busy, it returns length of dirty
// If free, returns length of m
func (m *Map) RealLength() int {
	if m.IsBusy() {
		return lengthOf(m.l, m.m)
	}
	return lengthOf(m.dl, m.dirty)
}

func rangem(l *sync.RWMutex, mp map[string]Value, f func(key string, value interface{}) bool) {
	l.RLock()
	defer l.RUnlock()

	for k, v := range mp {
		if !f(k, v.v) {
			break
		}
	}
}

func lengthOf(l *sync.RWMutex, mp map[string]Value) int {
	l.RLock()
	defer l.RUnlock()
	return len(mp)
}

func clearExpire(l *sync.RWMutex, m map[string]Value) int {
	var shouldDelete = make([]string, 0, 10)

	l.RLock()
	for k, v := range m {
		if v.isExpire() {
			shouldDelete = append(shouldDelete, k)
		}
	}
	l.RUnlock()

	l.Lock()
	for _, v := range shouldDelete {
		delete(m, v)
	}
	l.Unlock()

	return len(shouldDelete)
}

func setm(l *sync.RWMutex, m map[string]Value, key string, value interface{}, ext int64, offset int64, exp int64, nx bool) {
	l.Lock()
	defer l.Unlock()

	newValue := Value{
		v:   value,
		exp: exp,

		execAt: ext,
		offset: offset,
	}

	v, exist := m[key]

	// 当未过期，并且已存在时，nx不操作
	if exist && !v.isExpire() && nx == true {
		return
	}

	// 不存在时，设置新值
	if !exist {
		m[key] = newValue
		return
	}

	// 已失效时，设置新值
	if v.isExpire() {
		m[key] = newValue
		return
	}

	// 比新key后执行，则设置新值
	if v.FormerThan(newValue) {
		m[key] = newValue
		return
	}

	// 否则不操作
	return
}

func deletem(l *sync.RWMutex, m map[string]Value, key string, ext int64) {

	l.RLock()
	_, exist := m[key]
	l.RUnlock()

	if !exist {
		return
	}

	l.Lock()
	delete(m, key)
	l.Unlock()

	//if v.isExpire() {
	//	delete(m, key)
	//	return
	//}
	//
	//// 存储值的执行时间，小于ext时，才允许删。大于时，不能删
	//if v.execAt < ext {
	//	delete(m, key)
	//	return
	//}
	return
}

func (m *Map) mirrorOf(key string) (interface{}, bool) {

	m.gRLock()
	mv, mexist := m.m[key]

	wv, wexist := m.write[key]

	dv, dexist := m.del[key]
	m.gRUnlock()

	if !mexist && !wexist {
		return nil, false
	}

	nv := newestV(mv, wv)

	if !dexist {
		return nv.v, true
	}

	if dv.FormerThan(nv) {
		return nv.v, true
	}

	return nil, false
}

func newestV(v1, v2 Value) Value {
	if v1.FormerThan(v2) {
		return v2
	}
	return v1
}
