package cmap
import (
	"fmt"
	"testing"
	"time"

	"github.com/fwhezfwhez/errorx"
)

func TestChanMap(t *testing.T) {
	cm := NewChanMap(2000)
	

	if e:= cm.Set("username", "chan-map"); e!=nil {
		fmt.Println(errorx.Wrap(e).Error())
		return
	}
	
	v , e :=cm.Get("username")
	if e!=nil {
		fmt.Println(errorx.Wrap(e).Error())
		return
	}
	fmt.Println(v)

    if e:=cm.Delete("username"); e!=nil {
		fmt.Println(errorx.Wrap(e).Error())
		return
	}
	v , e =cm.Get("username")
	if e!=nil {
		fmt.Println(errorx.Wrap(e).Error())
		return
	}
	fmt.Println(v)
}

// BenchmarkChanMapSet-4   	  500000	      4140 ns/op	    1043 B/op	      14 allocs/op
// go test -benchmem -run=^$ cmap -bench ^(BenchmarkChanMapSet)$
func BenchmarkChanMapSet(b *testing.B) {
	cm := NewChanMap(1000)

	b.ResetTimer()

	for i:=0;i <b.N;i ++ {
		if e:=cm.Set(fmt.Sprintf("key-%d", i), i); e!=nil {
			fmt.Println(e)
			b.Fail()
			return
		}
	}
}

// go test test -benchmem -run=^$ cmap -bench ^(BenchmarkChanMapGet)$
// BenchmarkChanMapGet-4   	  100000	     15670 ns/op	    6112 B/op	      14 allocs/op
func BenchmarkChanMapGet(b *testing.B) {
	const n = 1000000
	
	var mp = make(map[string]Value)
	for i:=0;i <n;i ++ {
		mp[fmt.Sprintf("key-%d", i)] = Value {
			v : i,
			exp : -1,
			execAt:  time.Now().UnixNano(),
			offset: int64(i),
		}
	}
	
	cm := newChanMapWithExistedMap(100000, mp)

	b.ResetTimer()

	for i:=0;i <b.N;i ++ {
		index := randomInt(n)
		v,e  := cm.Get(fmt.Sprintf("key-%d", index))
		if e!=nil {
			fmt.Println(e)
			b.Fail()
			return
		}
		if v ==nil {
			b.Fail()
			return
		}
	}
}

// go test -benchmem -run=^$ cmap -bench ^(BenchmarkChanMapSetParallel)$
// BenchmarkChanMapSetParallel-4   	  300000	      6186 ns/op	    7164 B/op	      51 allocs/op
func BenchmarkChanMapSetParallel(b *testing.B) {
	cm := NewChanMap(1)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			func() {
				randomStr := randomString(40)
				if e:=cm.Set(randomStr, 1); e!=nil {
					fmt.Println(e)
					b.Fail()
					return
				}
			}()
		}
	})
}

// BenchmarkChanMapGetParallel-4   	  500000	      5483 ns/op	    6111 B/op	      14 allocs/op
// go test -benchmem -run=^$ cmap -bench ^(BenchmarkChanMapGetParallel)$
func BenchmarkChanMapGetParallel(b *testing.B) {
	const n = 1000000
	
	var mp = make(map[string]Value)
	for i:=0;i <n;i ++ {
		mp[fmt.Sprintf("key-%d", i)] = Value {
			v : i,
			exp : -1,
			execAt:  time.Now().UnixNano(),
			offset: int64(i),
		}
	}
	
	cm := newChanMapWithExistedMap(10000, mp)

	b.ResetTimer()


	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			func() {
				index := randomInt(n)
				v,e  := cm.Get(fmt.Sprintf("key-%d", index))
				if e!=nil {
					fmt.Println(e)
					b.Fail()
					return
				}
				if v ==nil {
					b.Fail()
					return
				}
			}()
		}
	})
}