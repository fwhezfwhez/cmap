package cmap

import (
	"fmt"
	"testing"
	"time"
)

func TestClist(t *testing.T) {
	list := newclist()
	list.RPush(1)
	list.RPushN(2, 3, 4, 5, 6, 7)

	fmt.Println(list.arr) //1-7

	fmt.Println(list.LPop()) // 1 true
	fmt.Println(list.arr)    // 2-7

	fmt.Println(list.LPop()) // 2 true
	fmt.Println(list.arr)    // 3-7

	fmt.Println(list.LPopN(3)) // 3,4,5
	fmt.Println(list.arr)      // 6,7

	fmt.Println(list.LRange(0, 0))        // 6
	fmt.Println(list.LRange(1, 1))        // 7
	fmt.Println(list.LRange(0, 5000))     // 6,7
	fmt.Println(list.LRange(-10, 1))      // 6,7
	fmt.Println(list.LRange(-10, 10))     // 6,7
	fmt.Println(list.LRange(-10, 0))      // 6,7
	fmt.Println(list.LRange(-10, -13))    // []
	fmt.Println(list.LRange(6000, 10000)) // []

	fmt.Println(list.LPopN(10)) // 6,7
	fmt.Println(list.arr)       //[]

	list.RPushN(8, 9, 10)
}

func TestConrurency(t *testing.T) {
	list := newclist()
	go func() {
		for {
			list.RPush(1)
		}
	}()
	go func() {
		for {
			list.RPushN(2, 3)
		}
	}()

	go func() {
		for {
			list.LPopN(1)
		}
	}()
	go func() {
		for {
			list.LPop()
		}
	}()
	go func() {
		for {
			list.LRange(3, 6)
		}
	}()
	go func() {
		for {
			list.LRange(-9, 6)
		}
	}()

	go func() {
		for {
			list.LLen()
		}
	}()

	time.Sleep(5 * time.Second)
	fmt.Println(list.String())
}
