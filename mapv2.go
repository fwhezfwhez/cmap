package cmap

import (
	"time"
)

// mapv2 is upgraded basing on map
// map now is fast and concurrently safe,but all keys will be put into a common race env. Apparently it's not proper if two irrelevant keys are operated at meanwhile to share a common lock.
// Thus, mapv2 is a combination of <hash, map>.All mapv2 api will be designed alike map.

// mapv2's slots are fixed once set right.It should never changed in runtime.

// comparing with slot-map, mapv2 is much smarter.
type MapV2 struct {
	hash  func(string) int64 // default hash is crc16 mechanism. Users can set your own hash function by `mv2.SetHash = func(string) int`
	slots []*Map             // slots are all maps. Keys will first get hashed and then decide to read/write which slots
	len   int

	clear chan struct{} // close mapv2 will send clear to finish mapd goroutine

}

func NewMapV2(hash func(string) int64, slotNum int, intervald time.Duration) *MapV2 {
	var mv2 = &MapV2{
		hash:  hash,
		slots: make([]*Map, slotNum, slotNum),
		clear: make(chan struct{}, 1),
	}

	for i, _ := range mv2.slots {
		mv2.slots[i] = newMap()
	}

	if mv2.hash == nil {
		mv2.hash = func(s string) int64 {
			return int64(UsMBCRC16([]byte(s)))
		}
	}

	mv2.len = slotNum

	mv2.mapd(intervald)
	return mv2
}

type debugger struct {
	slotIndex int
	hashN     int
}

func (mv2 *MapV2) Clear() {
	mv2.clear <- struct{}{}
}

func (mv2 *MapV2) Set(key string, value interface{}) {
	n := mv2.hash(key)
	mv2.slots[n%int64(mv2.len)].Set(key, value)
}
func (mv2 *MapV2) SetEx(key string, value interface{}, seconds int) {
	mv2.getslot(key).SetEx(key, value, seconds)
}
func (mv2 *MapV2) SetNx(key string, value interface{}) {
	mv2.getslot(key).SetNx(key, value)
}

func (mv2 *MapV2) SetExNx(key string, value interface{}, seconds int) {
	mv2.getslot(key).SetExNx(key, value, seconds)
}
func (mv2 *MapV2) Get(key string) (interface{}, bool) {

	n := mv2.hash(key)

	return mv2.slots[n%int64(mv2.len)].Get(key)
}

func (mv2 *MapV2) Incr(key string) int64 {
	return mv2.getslot(key).Incr(key)
}
func (mv2 *MapV2) IncrBy(key string, delta int) int64 {
	return mv2.getslot(key).IncrBy(key, delta)
}
func (mv2 *MapV2) IncrByEx(key string, delta int, seconds int) int64 {
	return mv2.getslot(key).IncrByEx(key, delta, seconds)
}

func (mv2 *MapV2) Decr(key string) int64 {
	return mv2.getslot(key).Decr(key)
}
func (mv2 *MapV2) DecrBy(key string, delta int) int64 {
	return mv2.getslot(key).DecrBy(key, delta)
}
func (mv2 *MapV2) DecrByEx(key string, delta int, seconds int) int64 {
	return mv2.getslot(key).DecrByEx(key, delta, seconds)
}

func (mv2 *MapV2) Delete(key string) {
	mv2.getslot(key).Delete(key)
}

func (mv2 *MapV2) getslot(key string) *Map {
	n := mv2.hash(key)

	i := n % int64(mv2.len)
	return mv2.slots[i]
}

// keep
func (mv2 *MapV2) mapd(interval time.Duration) {
	go func() {
		for {
			select {
			case <-time.After(interval):
				for i, _ := range mv2.slots {
					mv2.slots[i].ClearExpireKeys()
					time.Sleep(10 * time.Second)
				}
			case <-mv2.clear:
				return
			}
		}
	}()
}

func (mv2 *MapV2) PrintDetailOf(key string) string {
	n := mv2.hash(key)

	i := n % int64(mv2.len)
	return mv2.slots[i].PrintDetailOf(key)
}
