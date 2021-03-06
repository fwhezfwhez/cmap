package cmap

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMap(t *testing.T) {
	m := newMap()
	m.Set("username1", "fengtao")

	v, _ := m.Get("username1")
	if v.(string) != "fengtao" {
		panic("setapi ircorrect")
	}

	m.SetEx("username", "fengtao", 5)
	m.SetExNx("username", "xxx", 9)

	time.Sleep(1 * time.Second)
	v, _ = m.Get("username")
	if v.(string) != "fengtao" {
		panic("setexnxapi ircorrect")
	}
	time.Sleep(6 * time.Second)

	fmt.Println(m.ClearExpireKeys())

	v, _ = m.Get("username")
	if v != nil {
		panic("setexapi ircorrect")
		return
	}

	m.ClearExpireKeys()

	if len(m.m) != 1 {
		panic("clear irc")
	}
	m.PrintDetail()

}

func TestIncr(t *testing.T) {
	m := NewMap()

	rs := m.Incr("user-incr")

	if rs != 1 {
		panic("incr wrong")
	}

	rs = m.IncrBy("user-incr-by", 10)
	if rs != 10 {
		panic("incr-by wrong")
	}

	rs = m.Decr("user-decr")

	if rs != -1 {
		panic("decr wrong")
	}

	rs = m.DecrBy("user-decr-by", 13)

	if rs != -13 {
		panic("decr wrong")
	}

	w := sync.WaitGroup{}

	w.Add(100000)
	for i:=0;i<100000;i++ {
		go func(){
			defer w.Done()
			m.ClearExpireKeys()
		}()
	}


	w.Add(100000*2)
	for i := 0; i < 100000; i++ {
		go func(i int) {
			defer w.Done()

			var done = make(chan bool, 1)

			m.IncrByEx(fmt.Sprintf("%d", i), 13, 5)

			go func(i int) {
				defer w.Done()
				<-done
				v, exist := m.Get(fmt.Sprintf("%d", i))

				if v.(int) != 13 || !exist {
					panic("incr wrong")
				}

				time.Sleep(6 * time.Second)
				v, exist = m.Get(fmt.Sprintf("%d", i))

				if v != nil || exist {
					panic("incr wrong")
				}
			}(i)

			done <- true
		}(i)
	}
	w.Wait()

}

func TestDelete2(t *testing.T) {
	m := newMap()
	m.Set("username1", "fengtao")
	m.SetEx("username", "fengtao", 5)
	m.SetEx("hehe", "xxx", 6)

	// m.PrintDetail()

	m.Delete("username")
	m.Delete("username1")
	m.Delete("hehe")

	time.Sleep(1 * time.Second)

	if len(m.m) != 0 && len(m.dirty) != 0 {
		panic("del irc")
		return
	}

	// 	m.PrintDetail()
	// return

	// select {}
	// return

	wg := sync.WaitGroup{}
	wg.Add(100000*2 + 1)
	for i := 0; i < 100000; i++ {
		go func(i int) {
			defer wg.Done()
			var setdone = make(chan bool, 1)
			go func(i int, setdone chan bool) {
				defer wg.Done()
				<-setdone
				//time.Sleep(1 * time.Second)
				m.Delete(strconv.Itoa(i))

				v, _ := m.Get(strconv.Itoa(i))
				if v != nil {
					fmt.Println(m.PrintDetailOf(strconv.Itoa(i)))
					panic("sd del api irccect")
					os.Exit(1)
				}

				v, _ = m.Get(strconv.Itoa(i) + "set")
				if v.(int) != int(i) {
					fmt.Println(m.PrintDetailOf(strconv.Itoa(i)))
					panic("sg del api irccect")
					os.Exit(1)
				}

			}(i, setdone)

			m.Set(strconv.Itoa(i), i)
			m.Set(strconv.Itoa(i)+"set", int(i))
			setdone <- true
		}(i)
	}
	for i := 0; i < 1; i++ {
		go func(i int) {
			defer wg.Done()
			m.ClearExpireKeys()
		}(i)
	}
	wg.Wait()

	//m.ClearExpireKeys()
	//m.PrintDetail()
}

func TestDelete3(t *testing.T) {
	m := sync.Map{}
	m.Store("username1", "fengtao")

	// m.PrintDetail()

	m.Delete("username1")

	time.Sleep(1 * time.Second)

	// 	m.PrintDetail()
	// return

	// select {}
	// return

	wg := sync.WaitGroup{}
	wg.Add(100000 * 2)
	for i := 0; i < 100000; i++ {
		go func(i int) {
			defer wg.Done()
			var setdone = make(chan bool, 1)
			go func(i int, setdone chan bool) {
				defer wg.Done()
				<-setdone
				time.Sleep(1 * time.Second)
				m.Delete(strconv.Itoa(i))

				v, _ := m.Load(strconv.Itoa(i))
				if v != nil {
					panic("del api irccect")
				}
			}(i, setdone)

			m.Store(strconv.Itoa(i), i)
			setdone <- true
		}(i)
	}
	wg.Wait()

	//m.ClearExpireKeys()
	//m.PrintDetail()
}

func TestMap3(t *testing.T) {
	m := newMap()
	var i = 1
	m.Set(fmt.Sprintf("key-%d", i), i)
	m.SetEx(fmt.Sprintf("ex-key-%d", i), i, 3+i)

	m.Delete(fmt.Sprintf("key-%d", i))
	m.Delete(fmt.Sprintf("ex-key-%d", i))

	m.PrintDetail()
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
			_, _ = m.Get(fmt.Sprintf("key-%d", i))
			_, _ = m.Get(fmt.Sprintf("ex-key-%d", i))
		}(i)
	}

	time.Sleep(5 * time.Second)

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
			_, _ = m.Get(fmt.Sprintf("key-%d", i))
			_, _ = m.Get(fmt.Sprintf("ex-key-%d", i))
			_, _ = m.Get(fmt.Sprintf("nx-key-%d", i))
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
	case <-time.After(7 * time.Second):
		return
	}
}

func TestDelete(t *testing.T) {
	m := newMap()
	var wg sync.WaitGroup
	wg.Add(10 + 100000)
	for i := 0; i < 100000; i++ {
		go func(i int) {
			defer wg.Done()
			defer func() {
				if e := recover(); e != nil {
					fmt.Println(e)
					os.Exit(-1)
				}
			}()

			m.Set(fmt.Sprintf("key-%d", i), i)
			m.SetEx(fmt.Sprintf("ex-key-%d", i), i, 3+i)

			m.Delete(fmt.Sprintf("key-%d", i))
			m.Delete(fmt.Sprintf("ex-key-%d", i))
		}(i)
	}

	for i := 0; i < 10; i++ {
		go func(i int) {
			defer wg.Done()

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

	wg.Wait()

	b, e := json.MarshalIndent(m.Detail(), "", "  ")
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Println(string(b))
}

// go test -run ^BenchmarkMapSet$ -bench ^BenchmarkMapSet$ -benchmem
// BenchmarkMapSet-4   	 1000000	      1820 ns/op	     617 B/op	       5 allocs/op
func BenchmarkMapSet(b *testing.B) {
	m := newMap()
	for i := 0; i < b.N; i++ {
		m.Set(fmt.Sprintf("username-%d", i), fmt.Sprintf("cmap-%d", i))
	}
}

// go test -run ^BenchmarkSyncMapSet$ -bench ^BenchmarkSyncMapSet$ -benchmem
// BenchmarkSyncMapSet-4   	 1000000	      1931 ns/op	     243 B/op	       9 allocs/op
func BenchmarkSyncMapSet(b *testing.B) {
	m := sync.Map{}

	for i := 0; i < b.N; i++ {
		m.Store(fmt.Sprintf("username-%d", i), fmt.Sprintf("cmap-%d", i))
	}
}

// BenchmarkMapGet-4   	 5000000	       345 ns/op	      24 B/op	       1 allocs/op
// go test -run ^BenchmarkMapGet$ -bench ^BenchmarkMapGet$ -benchmem
func BenchmarkMapGet(b *testing.B) {
	m := newMap()
	for i := 0; i < 1000000; i++ {
		m.Set(fmt.Sprintf("username-%d", i), fmt.Sprintf("cmap-%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = m.Get(fmt.Sprintf("username-%d", i))
	}
}

// BenchmarkMapv2Get-4      3000000               478 ns/op              25 B/op          2 allocs/op
// go test -run ^BenchmarkMapv2Get$ -bench ^BenchmarkMapv2Get$ -benchmem
func BenchmarkMapv2Get(b *testing.B) {
	m := NewMapV2(nil, 8, time.Minute)
	time.Sleep(1 * time.Second)
	for i := 0; i < 1000000; i++ {
		m.Set(fmt.Sprintf("username-%d", i), fmt.Sprintf("cmap-%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = m.Get(fmt.Sprintf("username-%d", i))
	}
}

// BenchmarkSyncMapGet-4   	 3000000	       347 ns/op	      24 B/op	       2 allocs/op
// go test -benchmem -run=^$ cmap -bench ^(BenchmarkSyncMapGet)$
func BenchmarkSyncMapGet(b *testing.B) {
	m := sync.Map{}
	for i := 0; i < 1000000; i++ {
		m.Store(fmt.Sprintf("username-%d", i), fmt.Sprintf("cmap-%d", i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = m.Load(fmt.Sprintf("username-%d", i))
	}
}

// BenchmarkMapSetParallel-4   	  500000	      4020 ns/op	    6434 B/op	      40 allocs/op
// go test -run ^BenchmarkMapSetParallel$ -bench ^BenchmarkMapSetParallel$ -benchmem
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

// BenchmarkSyncMapSetParallel-4   	  500000	      4100 ns/op	    6464 B/op	      42 allocs/op
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

// BenchmarkMapv2SetParallel-4       300000              6718 ns/op            6484 B/op         41 allocs/op
// go test -run ^BenchmarkMapv2SetParallel$ -bench ^BenchmarkMapv2SetParallel$ -benchmem
func BenchmarkMapv2SetParallel(b *testing.B) {
	m := NewMapV2(nil, 8, 10*time.Minute)
	time.Sleep(1 * time.Second)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			func() {
				randomStr := randomString(40)
				m.Set(randomStr, 1)
			}()
		}
	})
}

// BenchmarkMapGetParallel-4   	  500000	      3409 ns/op	    5399 B/op	       3 allocs/op
// go test -run ^BenchmarkMapGetParallel$ -bench ^BenchmarkMapGetParallel$ -benchmem
func BenchmarkMapGetParallel(b *testing.B) {
	m := newMap()
	for i := 0; i < 1000000; i++ {
		m.Set(fmt.Sprintf("key-%d", i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			func() {
				_, _ = m.Get(fmt.Sprintf("key-%d", randomInt(100000)))
			}()
		}
	})
}

// BenchmarkMapGetParallel-4        500000              6480 ns/op            5399 B/op          3 allocs/op
// go test -run ^BenchmarkMapv2GetParallel$ -bench ^BenchmarkMapv2GetParallel$ -benchmem
func BenchmarkMapv2GetParallel(b *testing.B) {
	m := NewMapV2(nil, 8, 5*time.Minute)
	for i := 0; i < 1000000; i++ {
		m.Set(fmt.Sprintf("key-%d", i), i)
	}

	time.Sleep(1 * time.Second)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			func() {
				_, _ = m.Get(fmt.Sprintf("key-%d", randomInt(100000)))
			}()
		}
	})
}

// BenchmarkSyncMapGetParallel-4   	  200000	      5359 ns/op	    5399 B/op	       3 allocs/op
// go test -run ^BenchmarkSyncMapGetParallel$ -bench ^BenchmarkSyncMapGetParallel$ -benchmem
func BenchmarkSyncMapGetParallel(b *testing.B) {
	m := sync.Map{}
	for i := 0; i < 1000000; i++ {
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

// BenchmarkMapClearExpireKeys-4   	  500000	      3343 ns/op
// go test -run ^BenchmarkMapClearExpireKeys$ -bench ^BenchmarkMapClearExpireKeys$ -benchmem
func BenchmarkMapClearExpireKeys(b *testing.B) {
	m := newMap()
	for i := 0; i < 1000000; i++ {
		m.Set(fmt.Sprintf("key-%d", i), i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			func() {
				m.ClearExpireKeys()
			}()
		}
	})
}

func TestMapClearExpireKey(t *testing.T) {
	m := newMap()
	for i := 0; i < 1000000; i++ {
		m.Set(fmt.Sprintf("key-%d", i), i)
	}
	m.ClearExpireKeys()
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

func TestSelect(t *testing.T) {
	var a = make(chan int)
	select {
	case <-time.After(5 * time.Second):
		fmt.Println("time out ")
	case a <- 5:
	}
}

func TestConcurrentatomic(t *testing.T) {
	var a int32 = 0
	var m = make(map[string]int, 0)

	var times int32
	wg := sync.WaitGroup{}
	f := func() {
		defer wg.Done()
		rs := atomic.AddInt32(&a, 1)
		defer atomic.AddInt32(&a, -1)

		if rs != 1 {
			return
		}

		atomic.AddInt32(&times, 1)
		m["1"] = 1
		_ = m["1"]
	}

	wg.Add(200000)
	for i := 0; i < 100000; i++ {
		go f()
	}
	for i := 0; i < 100000; i++ {
		go f()
	}

	wg.Wait()

	fmt.Println(times)
}
