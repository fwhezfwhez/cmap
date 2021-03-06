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
	wg.Add(10000)

	var mv2 = NewMapV2(nil, 15, 10*5*time.Second)

	for i := 0; i < 10000; i ++ {

		go func(i int) {
			defer wg.Done()
			var ds, dg debugger
			var key = "hello" + strconv.Itoa(i)

			// mv2.Set(key, int(5), &ds)
			//v, _ := mv2.Get(key, &dg)

			mv2.slots[0].Set(key, int(5))
			v, _ := mv2.slots[0].Get(key)
			if v == nil {
				fmt.Println(mv2.slots[dg.slotIndex].PrintDetailOf(key))
				fmt.Println(key, dg, ds)
				os.Exit(1)
			}
			if v.(int) != 5 {
				panic("bad f")
			}
		}(i)

		//go func() {
		//	defer wg.Done()
		//
		//	mv2.SetEx("hello2", int(5), 15)
		//	v, _ := mv2.Get("hello2")
		//	if v.(int) != 5 {
		//		panic("bad f")
		//	}
		//}()
		//
		//go func() {
		//	defer wg.Done()
		//
		//	mv2.SetNx("hello3", int(5))
		//	v, _ := mv2.Get("hello3")
		//	if v.(int) != 5 {
		//		panic("bad f")
		//	}
		//
		//	mv2.SetNx("hello3", int(8))
		//	v, _ = mv2.Get("hello3")
		//	if v.(int) != 5 {
		//		panic("bad f")
		//	}
		//}()
		//
		//go func() {
		//	defer wg.Done()
		//
		//	mv2.SetNx("hello4", int(5))
		//
		//	mv2.Delete("hello4")
		//	_, exist := mv2.Get("hello4")
		//	if exist == true {
		//		panic("bad f4")
		//	}
		//}()

	}
	wg.Wait()

}

func TestNewMap1(t *testing.T) {
	var mv2 = NewMap()

	wg := sync.WaitGroup{}
	wg.Add(100001)

	for i := 0; i < 1; i ++ {
		func() {
			defer wg.Done()
			go mv2.ClearExpireKeys()
		}()
	}

	for i := 0; i < 100000; i ++ {

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
	for i := 0; i < b.N; i ++ {
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
