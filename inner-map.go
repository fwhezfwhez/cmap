package cmap

import "sync"

type innerMap struct {
	l *sync.RWMutex
	m map[string]Value
}

func newInnerMap() *innerMap {
	return &innerMap{
		l: &sync.RWMutex{},
		m: make(map[string]Value),
	}
}

// 设置map值
func (im *innerMap) setm(key string, value interface{}, ext int64, offset int64, exp int64, nx bool) {
	l := im.l
	m := im.m

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

func (im *innerMap) casm(key string, f func(old interface{}) (interface{}, error), exp int64, ext int64, offset int64) interface{} {
	l := im.l
	m := im.m

	l.Lock()
	defer l.Unlock()

	oldv, exist := m[key]
	if !exist {
		neww, e := f(oldv.v)
		if e != nil {
			return nil
		}

		newValue := Value{
			v:   neww,
			exp: exp,

			execAt: ext,
			offset: offset,
		}
		m[key] = newValue
		return newValue.v
	}

	neww, e := f(oldv.v)
	if e != nil {
		return nil
	}

	newValue := Value{
		v:   neww,
		exp: exp,

		execAt: ext,
		offset: offset,
	}
	m[key] = newValue

	return neww
}

func (im *innerMap) Len() int {
	im.l.RLock()
	defer im.l.RUnlock()

	return len(im.m)
}

func (im *innerMap) resetMap() {
	im.l.Lock()
	defer im.l.Unlock()

	im.m = make(map[string]Value)
}

func (im *innerMap) Range(f func(key string, value Value)) {
	im.l.RLock()
	defer im.l.RUnlock()

	for k, v := range im.m {
		f(k, v)
	}
}
