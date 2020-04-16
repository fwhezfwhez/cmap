package cmap

import (
	"sync"
)

// SlotMap consist of many map slots
// It can auto reduce data to those slots by s.hash().
// If sm.autoExtend = true, slots will work in extendable size of slots concurrently safe.
type SlotMap struct {
	autoExtend bool

	l     *sync.RWMutex
	slots []Map
}

func NewSlotMap(slotNum int, autoExtend bool) *SlotMap {
	return &SlotMap{
		autoExtend: autoExtend,
		l:          &sync.RWMutex{},
		slots:      make([]Map, slotNum, 2*slotNum),
	}
}

func (s *SlotMap) lock() {
	if s.autoExtend {
		s.l.Lock()
	}
}

func (s *SlotMap) unlock() {
	if s.autoExtend {
		s.l.Unlock()
	}
}
func (s *SlotMap) rLock() {
	if s.autoExtend {
		s.l.RLock()
	}
}

func (s *SlotMap) rUnlock() {
	if s.autoExtend {
		s.l.RUnlock()
	}
}

func (s *SlotMap) SlotNum() int {
	s.rLock()
	num := len(s.slots)
	s.rUnlock()
	return num
}

func (s SlotMap) hash(key string) int {
	var slotNum int
	s.rLock()
	slotNum = len(s.slots)
	s.rUnlock()
	return UsMBCRC16([]byte(key)) % slotNum
}

func (s *SlotMap) Set(key string, value interface{}) {
	s.rLock()
	slot := s.slots[s.hash(key)]
	s.rUnlock()

	slot.Set(key, value)
	return
}

func (s *SlotMap) Get(key string) interface{} {
	s.rLock()
	slot := s.slots[s.hash(key)]
	s.rUnlock()

	return slot.Get(key)
}

func (s *SlotMap) Delete(key string) {
	s.rLock()
	slot := s.slots[s.hash(key)]
	s.rUnlock()

	slot.Delete(key)
	return
}
