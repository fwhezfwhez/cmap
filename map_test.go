package cmap

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestMap(t *testing.T) {
	m := newMap()
	m.Set("username1", "fengtao")
	m.SetEx("username", "fengtao", 5)
	m.SetEx("hehe", "xxx", 6)

	time.Sleep(6 * time.Second)

	fmt.Println(m.ClearExpireKeys())

	fmt.Println(m.Len())
}

func TestWRCMap(t *testing.T) {
	m := newMap()
	for i := 0; i < 10000; i++ {
		go func(i int) {
			defer func() {
				if e := recover(); e != nil {
					fmt.Println(e)
					os.Exit(-1)
				}
			}()
			m.Set(fmt.Sprintf("key-%d", i), i)
			m.SetEx(fmt.Sprintf("ex-key-%d", i), i, 3+i)

		}(i)
	}
	for i := 0; i < 10000; i++ {
		go func(i int) {
			defer func() {
				if e := recover(); e != nil {
					fmt.Println(e)
					os.Exit(-1)
				}
			}()
			_ = m.Get(fmt.Sprintf("key-%d", i))
			_ = m.Get(fmt.Sprintf("ex-key-%d", i))
		}(i)
	}

	time.Sleep(10 * time.Second)

	for i := 0; i < 100; i++ {
		go func(i int) {
			defer func() {
				if e := recover(); e != nil {
					fmt.Println(e)
					os.Exit(-1)
				}
			}()
			n := m.ClearExpireKeys()
			fmt.Println(n)
		}(i)
	}
	for i := 0; i < 10000; i++ {
		go func(i int) {
			defer func() {
				if e := recover(); e != nil {
					fmt.Println(e)
					os.Exit(-1)
				}
			}()
			m.Set(fmt.Sprintf("key-%d", i), i)
			m.SetEx(fmt.Sprintf("ex-key-%d", i), i, 3+i)
			m.SetNx(fmt.Sprintf("nx-key-%d", i), i)
			m.SetExNx(fmt.Sprintf("nx-key-%d", i), i, 5)
		}(i)
	}
	for i := 0; i < 10000; i++ {
		go func(i int) {
			defer func() {
				if e := recover(); e != nil {
					fmt.Println(e)
					os.Exit(-1)
				}
			}()
			_ = m.Get(fmt.Sprintf("key-%d", i))
			_ = m.Get(fmt.Sprintf("ex-key-%d", i))
			_ = m.Get(fmt.Sprintf("nx-key-%d", i))
		}(i)
	}
	select {
	case <-time.After(20 * time.Second):
		return
	}
}

// 1482 ns/op
func BenchmarkMapSet(b *testing.B) {
	m := newMap()
	for i := 0; i < b.N; i++ {
		m.Set(fmt.Sprintf("username-%d", i), fmt.Sprintf("cmap-%d", i))
	}
}

// 1709 ns/op
func BenchmarkSyncMapSet(b *testing.B) {
	m := sync.Map{}

	for i := 0; i < b.N; i++ {
		m.Store(fmt.Sprintf("username-%d", i), fmt.Sprintf("cmap-%d", i))
	}
}

// 173 ns/op
func BenchmarkMapGet(b *testing.B) {
	m := newMap()
	for i := 0; i < 10000; i++ {
		m.Set(fmt.Sprintf("username-%d", i), fmt.Sprintf("cmap-%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = m.Get(fmt.Sprintf("username-%d", i))
	}
}

// 186 ns/op
func BenchmarkSyncMapGet(b *testing.B) {
	m := sync.Map{}
	for i := 0; i < 10000; i++ {
		m.Store(fmt.Sprintf("username-%d", i), fmt.Sprintf("cmap-%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = m.Load(fmt.Sprintf("username-%d", i))
	}
}
