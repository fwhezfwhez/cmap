package cmap

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestMapHA(t *testing.T) {
	m := newMap()

	go func() {
		for i := 0; true; {
			i += 1
			go func(i int) {
				m.SetEx(fmt.Sprintf("hello_%d", i), i, 15)
				m.Get(fmt.Sprintf("hello_%d", i-1))
			}(i)
		}
	}()

	go func() {
		for i := 0; true; {
			i += 1
			go func(i int) {
				m.Delete(fmt.Sprintf("hello_%d", i))
			}(i)
		}
	}()

	go func() {
		for {
			go func() {
				m.ClearExpireKeys()
			}()
		}
	}()

	time.Sleep(5 * time.Second) // 先并读发写5秒\

	m.StopWrite() // 停止写入

	time.Sleep(2 * time.Second)
	m.StopBoostClear() // 停止执行新的clear工作

	time.Sleep(10 * time.Second)

	mv := m.Detail() // 期待存在长度

	t.Log(JSONString(mv))

	time.Sleep(7 * time.Second) // 确保覆盖测试用例15秒

	m.clearExpireKeysInternal()

	mv = m.Detail() // 期待输出长度全部为0

	t.Log(JSONString(mv))

}

func TestMapV2HA(t *testing.T) {

	OpenLog = true
	m := NewNamedMapV2("TestMapV2HA", 1, 2*time.Second)

	go func() {
		for {
			m.SetEx(fmt.Sprintf("hello_%d", rand.Intn(100000)), 7, 5)
			m.Get(fmt.Sprintf("hello"))
		}
	}()

	go func() {
		for {
			m.SetEx(fmt.Sprintf("hello_%d", rand.Intn(100000)), 7, 5)
			m.Get(fmt.Sprintf("hello"))
		}
	}()

	go func() {
		for {
			m.Delete("")
			_ = m.MetaStrings()
		}
	}()

	go func() {
		for {
			m.Delete("")
			_ = m.MetaStrings()
		}
	}()

	go func() {
		for {
			m.IncrByEx(fmt.Sprintf("hello_incr_%d", rand.Intn(100000)), 1, 2)
			m.Get(fmt.Sprintf("hello"))
		}
	}()
	go func() {
		for {
			m.IncrByEx(fmt.Sprintf("hello_incr_%d", rand.Intn(100000)), 1, 2)
			m.Get(fmt.Sprintf("hello"))
		}
	}()

	time.Sleep(10 * time.Second)
	m.slots[0].StopWrite()

	select {}
}
