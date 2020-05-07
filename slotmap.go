package cmap

import (
	"fmt"
	"sync"
	"time"
)

const (
	Sl_Is_Extending = 2
)

// SlotMap consist of many map slots
// It can auto reduce data to those slots by s.hash().
// If sm.autoExtend = true, slots will work in extendable size of slots concurrently safe.
type SlotMap struct {

	// not used then
	// if autoExtend = true, slots size will adjust autonomously, about 200000 per slot
	// autoExtend is not changeable in runtime, decide auto-extendable or not when call newSlotMap()
	autoExtend       bool
	extendingState   int
	slotOverWeighNum int

	// only when autoExtend=true, checkIneval will make sense.
	checkInteval time.Duration

	l     *sync.RWMutex
	slots []Map
}

func NewSlotMap(slotNum int, autoExtend bool, checkIneval time.Duration) *SlotMap {
	s := &SlotMap{
		autoExtend:       autoExtend,
		checkInteval:     checkIneval,
		slotOverWeighNum: 200000,

		l:     &sync.RWMutex{},
		slots: make([]Map, slotNum, 2*slotNum),
	}
	if autoExtend {
		go func() {
			for {
				fmt.Println("[extend]Start daily extend spying")
				s.extend()
				time.Sleep(s.checkInteval)
			}
		}()
	}
	return s
}

func (s *SlotMap) isExtending() bool {
	s.rLock()
	state := s.extendingState
	s.rUnlock()
	return state == Sl_Is_Extending
}

func (s *SlotMap) setExtendingState() {
	s.lock()
	s.extendingState = Sl_Is_Extending
	s.unlock()
}
func (s *SlotMap) freeExtendingState() {
	s.lock()
	s.extendingState = 0
	s.unlock()
}

func (s *SlotMap) shouldExend() bool {
	s.l.RLock()
	slotNum := len(s.slots)
	maxOverSlotNum := slotNum / 3

	var numOverSlotNum = 0
	for i, _ := range s.slots {
		if s.slots[i].IsBusy() {
			continue
		} else {
			if s.slots[i].MLen() > s.slotOverWeighNum {
				numOverSlotNum++
				if numOverSlotNum >= maxOverSlotNum {
					return true
				} else {
					continue
				}
			}
		}
	}
	s.l.RUnlock()

	return false
}
func (s *SlotMap) extend() {
	if s.isExtending() {
		fmt.Println("[extend]Another extend job is working, no need to extend twice")
		return
	}
	if !s.shouldExend() {
		fmt.Println("[extend]Slots have healthy length, no need to extend")
		return
	}
	s.setExtendingState()
	defer s.freeExtendingState()
	s.lock()

	s.unlock()
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

func (s *SlotMap) SetEx(key string, value interface{}, seconds int) {
	s.rLock()
	slot := s.slots[s.hash(key)]
	s.rUnlock()

	slot.SetEx(key, value, seconds)
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
