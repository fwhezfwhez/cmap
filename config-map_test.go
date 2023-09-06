package cmap

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

var value int64 = 1

func dbValue() int64 {
	return value
}

func printvalue() {
	fmt.Println(dbValue())
}

// 每隔1秒，自增1
func changing() {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			atomic.AddInt64(&value, 1)
		}
	}()
}

var cm = NewConfigMap(nil, 4, 30*time.Minute)

func GetDBValue() (int64, bool) {
	var key = fmt.Sprintf("testconfigmap:%s", "dbvalue")
	rs, needloading, exist := cm.Get(key)

	if needloading {
		v := dbValue()
		fmt.Println("load db")

		cm.SetEx(key, v, 15)

		return v, true
	}

	if exist {
		return rs.(int64), true
	}
	return 0, false
}
func TestNewConfigMap(t *testing.T) {

	go changing()

	GetDBValue()

	time.Sleep(120 * time.Second)

	//go func() {
	//	for {
	//		time.Sleep(1 * time.Second)
	//		printvalue()
	//	}
	//}()

	go func() {
		var i = 0
		for {
			i++
			time.Sleep(500 * time.Millisecond)
			r, exist := GetDBValue()

			fmt.Println(r, exist)

		}
	}()
	go func() {
		for {
			time.Sleep(200 * time.Millisecond)

			r, exist := GetDBValue()

			fmt.Println(r, exist)
		}
	}()

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)

			r, exist := GetDBValue()

			fmt.Println(r, exist)
		}
	}()

	select {}
}
