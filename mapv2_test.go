package cmap

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestNewMapV2(t *testing.T) {
	wg := sync.WaitGroup{}
	var mv2 = NewMapV2(nil, 15, 10*5*time.Second)

	for i := 0; i < 10; i++ {

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var key = "hello" + strconv.Itoa(i)

			mv2.Set(key, int(5))
			v, _ := mv2.Get(key)
			if v == nil {
				fmt.Println(mv2.PrintDetailOf(key))
				os.Exit(1)
			}
			if v.(int) != 5 {
				panic("bad f1")
			}
		}(i)

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			var key = "hello2" + strconv.Itoa(i)

			mv2.SetEx(key, int(5), 1)
			v, _ := mv2.Get(key)
			if v.(int) != 5 {
				panic("bad f2 5")
			}

			time.Sleep(1 * time.Second)

			v, _ = mv2.Get(key)
			if v != nil {
				fmt.Println(key)
				fmt.Println(v)
				fmt.Println(mv2.PrintDetailOf(key))
				panic("bad f2 nil")
			}

		}(i)

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			key := fmt.Sprintf("%s:%d", "hello3", i)
			mv2.SetNx(key, int(5))
			v, _ := mv2.Get(key)
			if v.(int) != 5 {
				panic("bad f3 5")
			}

			mv2.SetNx(key, int(8))
			v, _ = mv2.Get(key)
			if v.(int) != 5 {
				panic("bad f3 8")
			}

			mv2.SetExNx(key, int(10), 1)

			v, _ = mv2.Get(key)
			if v.(int) != 5 {
				panic("bad f3 10")
			}

			time.Sleep(1 * time.Second)
			v, _ = mv2.Get(key)
			if v.(int) != 5 {
				panic("bad f3 10")
			}

			key4 := key + "exnx" + strconv.Itoa(i)
			mv2.SetExNx(key4, int(11), 2)

			v, _ = mv2.Get(key4)
			if v.(int) != 11 {
				panic("bad f4 11")
			}

			mv2.SetExNx(key4, int(12), 9)
			v, _ = mv2.Get(key4)
			if v.(int) != 11 {
				panic("bad f4 11")
			}

			time.Sleep(2 * time.Second)
			v, _ = mv2.Get(key4)
			if v != nil {
				panic("bad f4 11")
			}
		}(i)
		//
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			key := fmt.Sprintf("%s:%d", "hello5", i)
			mv2.SetEx(key, 15, 7)
			mv2.SetNx(key, 15)

			time.Sleep(7 * time.Second)
			v, _ := mv2.Get(key)
			if v != nil {
				panic("bad f5 15")
			}
			mv2.SetEx(key, 15, 1)
			mv2.Set(key, 16)
			time.Sleep(1 * time.Second)
			v, _ = mv2.Get(key)
			if v.(int) != 16 {
				panic("bad f5 16")
			}

			mv2.Delete(key)
			_, exist := mv2.Get(key)
			if exist == true {
				panic("bad delete")
			}
		}(i)

	}
	wg.Wait()
}

func TestNewMap1(t *testing.T) {
	var mv2 = NewMap()

	wg := sync.WaitGroup{}
	wg.Add(100001)

	for i := 0; i < 1; i++ {
		func() {
			defer wg.Done()
			go mv2.ClearExpireKeys()
		}()
	}

	for i := 0; i < 100000; i++ {

		go func(i int) {
			defer wg.Done()

			// var circletimes int32 =0
			// bug: 在busy时，set
			//      在free时，get
			mv2.Set("hello"+strconv.Itoa(i), int(5))

			v, _ := mv2.Get("hello" + strconv.Itoa(i))

			if v == nil {

				mv2.PrintDetailOf("hello" + strconv.Itoa(i))
				panic(fmt.Errorf("nil"))
				os.Exit(1)
			}
			if v.(int) != 5 {
				panic("bad f")
			}
		}(i)
	}
	wg.Wait()
	// time.Sleep(10 * time.Second)
}

func BenchmarkNewMap1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var mv2 = NewMap()

		func(i int) {
			mv2.Set("hello"+strconv.Itoa(i), int(5))
			v, _ := mv2.Get("hello" + strconv.Itoa(i))

			if v.(int) != 5 {
				panic("bad f")
			}
		}(i)
	}
}

func TestIncrEx(t *testing.T) {
	m := NewMapV2(nil, 2, 5*time.Minute)

	var once = func(key string, seconds int) bool {
		rs := m.IncrByEx(key, 1, seconds)

		if rs == 1 {
			return true
		}

		return false
	}

	fmt.Println(once("1111", 3)) // true
	fmt.Println(once("1111", 3)) // false
	time.Sleep(4 * time.Second)
	fmt.Println(once("1111", 3)) // true
	fmt.Println(once("1111", 3)) // false
}
