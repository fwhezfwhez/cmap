package cmap

import (
	"hash/crc32"
	"sync"
)

// SlotMap consist of many map slots
// It can auto reduce data to those slots by s.hash().
// If sm.autoExtend = true, slots will work in extendable size of slots concurrently safe.
type SlotMap struct{
	autoExtend bool

	l *sync.RWMutex
	slots []Map
}

func NewSlotMap(slotNum int, autoExtend bool) *SlotMap {
	return &SlotMap{
	autoExtend: autoExtend,
	l: &sync.RWMutex{},
	slots: make([]Map, slotNum, 2 *slotNum)
	}
}

func (s *slotMap) lock() {
    if s.autoExtend {
		s.l.Lock()
	}
}

func (s *slotMap) unlock() {
	if s.autoExtend {
		s.l.UnLock()
	}
}
func (s *slotMap) rLock() {
    if s.autoExtend {
		s.l.RLock()
	}
}

func (s *slotMap) rUnlock() {
	if s.autoExtend {
		s.l.RUnLock()
	}
}

func (s *slotMap) SlotNum() int {
	s.rlock()
    num := len(s.slots)
	s.rUnlock()
	return num
}


func (s slotMap) hash(key string) int {
	return UsMBCRC16([]byte(key)) % s.slotNum
}

func (s *slotMap) Set(key string, value interface{}) {
	s.rlock()
	slot := slots[s.hash(key)]
	s.rUnlock()
	
	slot.Set(key, value)
	return
}

func (s *slotMap) Get(key string) interface{} {
	s.rlock()
	slot := slots[s.hash(key)]
	s.rUnlock()

	return slot.Get(key)
}

func (s *slotMap) Delete(key string) {
	s.rlock()
	slot := slots[s.hash(key)]
	s.rUnlock()

	slot.Delete(key)
	return
}



