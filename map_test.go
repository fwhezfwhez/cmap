package cmap

import (
	"fmt"
	"math/rand"
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
	for i := 0; i < 10000; i++ {
		go func(i int) {
			defer func() {
				if e := recover(); e != nil {
					fmt.Println(e)
					os.Exit(-1)
				}
			}()
			_ = m.Detail()
		}(i)
	}
	select {
	case <-time.After(10 * time.Second):
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
	for i := 0; i < 100000; i++ {
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
	for i := 0; i < 100000; i++ {
		m.Store(fmt.Sprintf("username-%d", i), fmt.Sprintf("cmap-%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = m.Load(fmt.Sprintf("username-%d", i))
	}
}

// length, allocs/op
// 40      3773 ns/op
// go test -benchmem -run=^$ cmap -bench ^(BenchmarkSyncMapSetParallel)$
func BenchmarkMapSetParallel(b *testing.B) {
	m := newMap()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			func() {
				randomStr := randomString(40)
				m.Set(randomStr, 1)
			}()
		}
	})
}

// length, allocs/op
// 40      4494 ns/op
// test -benchmem -run=^$ cmap -bench ^(BenchmarkSyncMapSetParallel)$
func BenchmarkSyncMapSetParallel(b *testing.B) {
	m := sync.Map{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			func() {
				randomStr := randomString(40)
				m.Store(randomStr, 1)
			}()
		}
	})
}

// BenchmarkMapGetParallel-4   	  300000	      3333 ns/op	    5399 B/op	       3 allocs/op
// go test -benchmem -run=^$ cmap -bench ^(BenchmarkMapGetParallel)$
func BenchmarkMapGetParallel(b *testing.B) {
	m := newMap()
	for i := 0; i < 100000; i++ {
		m.Set(fmt.Sprintf("key-%d", i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			func() {
				_ = m.Get(fmt.Sprintf("key-%d", randomInt(100000)))
			}()
		}
	})
}

// BenchmarkSyncMapGetParallel-4   	  300000	      3733 ns/op	    5455 B/op	       5 allocs/op
// go test -benchmem -run=^$ cmap -bench ^(BenchmarkSyncMapGetParallel)$
func BenchmarkSyncMapGetParallel(b *testing.B) {
	m := sync.Map{}
	for i := 0; i < 100000; i++ {
		m.Store(fmt.Sprintf("key-%d", i), i)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			func() {
				_, _ = m.Load(fmt.Sprintf("key-%d", randomInt(100000)))
			}()
		}
	})
}

func randomString(length int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var r *rand.Rand

	r = rand.New(rand.NewSource(time.Now().UnixNano()))

	var result string
	for i := 0; i < length; i++ {
		result += string(str[r.Intn(len(str))])
	}
	return result
}

func randomInt(max int) int {
	var r *rand.Rand

	r = rand.New(rand.NewSource(time.Now().UnixNano()))

	return r.Intn(max)
}
