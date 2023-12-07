package cmap

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type clist struct {
	arr []interface{}
	l   *sync.RWMutex

	rpushTimes  int64
	rpushnTimes int64
	lpopTimes   int64
	lpopnTimes  int64
	lrangeTimes int64
	llenTimes   int64
}

func newclist() *clist {
	return &clist{
		arr: make([]interface{}, 0, 10),
		l:   &sync.RWMutex{},
	}
}

func (m *clist) String() string {
	m.l.RLock()
	defer m.l.RUnlock()
	return fmt.Sprintf("%s clist len=%d rpush_times=%d rpushn_times=%d lpop_times=%d lpopn_times=%d lrange_times=%d llen_times=%d",
		time.Now().Format("2006-01-02 15:04:05"),
		len(m.arr),
		m.rpushTimes,
		m.rpushnTimes,
		m.lpopTimes,
		m.lpopnTimes,
		m.lrangeTimes,
		m.llenTimes,
	)
}

func (m *clist) RPush(value interface{}) {
	m.l.Lock()
	defer m.l.Unlock()

	atomic.AddInt64(&m.rpushTimes, 1)
	m.arr = append(m.arr, value)
}
func (m *clist) RPushN(values ...interface{}) {
	m.l.Lock()
	defer m.l.Unlock()
	atomic.AddInt64(&m.rpushnTimes, 1)

	m.arr = append(m.arr, values...)
}

func (m *clist) LPop() (interface{}, bool) {
	m.l.Lock()
	defer m.l.Unlock()
	atomic.AddInt64(&m.lpopTimes, 1)

	if len(m.arr) == 0 {
		return nil, false
	}
	rs := m.arr[0]

	m.arr = m.arr[1:]

	return rs, true
}

func (m *clist) LPopN(n int) []interface{} {
	if n <= 0 {
		return []interface{}{}
	}

	m.l.Lock()
	defer m.l.Unlock()
	atomic.AddInt64(&m.lpopnTimes, 1)

	if len(m.arr) == 0 {
		return []interface{}{}
	}

	if n >= len(m.arr) {

		rs := copySlice(m.arr)
		m.arr = []interface{}{}
		return rs
	}

	rs := copySlice(m.arr[0:n])
	m.arr = m.arr[n:]
	return rs
}
func (m *clist) LRange(start int, end int) []interface{} {
	if start > end {
		return []interface{}{}
	}

	m.l.RLock()
	defer m.l.RUnlock()

	atomic.AddInt64(&m.lrangeTimes, 1)

	//                                start**********end
	//   arr[0]------------arr[len-1]
	if start > len(m.arr)-1 {
		return []interface{}{}
	}

	// start**********end
	//                    arr[0]------------
	if end < 0 {
		return []interface{}{}
	}

	if end-start >= len(m.arr) {
		return copySlice(m.arr)
	}

	if len(m.arr) == 0 {
		return []interface{}{}
	}

	// start**********end
	//         arr[0]------------len-1
	if start <= 0 && 0 <= end && end <= len(m.arr)-1 {
		return copySlice(m.arr[0 : end+1])
	}

	//           start**********end
	// arr[0]------------arr[len-1]
	if start <= len(m.arr)-1 && len(m.arr)-1 <= end {
		return copySlice(m.arr[start:len(m.arr)])
	}

	//           start**********end
	//       arr[0]------------arr[len-1]

	return copySlice(m.arr[start : end+1])
}

func (m *clist) LLen() int {
	m.l.RLock()
	defer m.l.RUnlock()

	atomic.AddInt64(&m.llenTimes, 1)

	return len(m.arr)
}

func copySlice(arr []interface{}) []interface{} {
	var rs = make([]interface{}, 0, 10)

	for _, v := range arr {
		rs = append(rs, v)
	}
	return rs
}
