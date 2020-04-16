package cmap

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"reflect"

	"github.com/fwhezfwhez/errorx"
)

type ChanMapRepsonse struct {
	Response interface{}
	Err      error
}

func errOf(e error) ChanMapRepsonse {
	return ChanMapRepsonse{
		Response: nil,
		Err:      errorx.Wrap(e),
	}
}
func successOf(resp interface{}) ChanMapRepsonse {
	return ChanMapRepsonse{
		Response: resp,
		Err:      nil,
	}
}

type OperationI interface {
	Command() string
	Values() []interface{}
	Response() chan ChanMapRepsonse
}

type ChanMap struct {
	l *sync.RWMutex
	m map[string]Value

	ol         *sync.RWMutex
	operations chan OperationI

	consumed bool
	offset   int64

	forceClear chan struct{}
}

func NewChanMap(cap int) *ChanMap {
	cm := &ChanMap{
		l: &sync.RWMutex{},
		m: make(map[string]Value),

		ol:         &sync.RWMutex{},
		operations: make(chan OperationI, cap),

		forceClear: make(chan struct{}, 1),
	}

	cm.autoConsume()
	return cm
}

func newChanMapWithExistedMap(cap int, m map[string]Value) *ChanMap {
	cm := &ChanMap{
		l: &sync.RWMutex{},
		m: m,

		ol:         &sync.RWMutex{},
		operations: make(chan OperationI, cap),

		forceClear: make(chan struct{}, 1),
	}

	cm.autoConsume()
	return cm
}

func (cm *ChanMap) recevOperation(o OperationI) {
	cm.operations <- o
}
func (cm *ChanMap) offsetIncr() int64 {
	new := atomic.AddInt64(&cm.offset, 1)
	atomic.CompareAndSwapInt64(&cm.offset, math.MaxInt64, 0)
	return new
}

func (cm *ChanMap) autoConsume() {
	if cm.consumed {
		return
	}
	cm.consumed = true
	go func(cm *ChanMap) {
	L:
		for {
			select {
			// forcely stop all acitivity of cm
			case <-cm.forceClear:
				break L
			case v, ok := <-cm.operations:
				// cm.operations has been closed, deny writing but readable
				if !ok {
					// consume over
					if v == nil {
						break L
					}
				} else {
					cm.handle(v.Response(), v.Command(), v.Values()...)
				}
			}
		}

	}(cm)
}

// ForceClear will stop chanMap consuming regardless of existence of data remained in chanMap.operations chanel
func (cm *ChanMap) forceStop() {
	cm.forceClear <- struct{}{}
}

// close operations make it unwritable but readable
func (cm *ChanMap) gracefulStop() {
	close(cm.operations)
}

func (cm *ChanMap) writeResponse(response chan ChanMapRepsonse, resp ChanMapRepsonse) {
	go func() {
		select {
		case <-time.After(10 * time.Second):
			fmt.Println("write reponse time out(10s)")
			return
		case response <- resp:
			return
		}
	}()
}

// handle command
func (cm *ChanMap) handle(response chan ChanMapRepsonse, command string, values ...interface{}) {
	switch command {
	case "set", "SET":
		if e := cm.set(values...); e != nil {
			cm.writeResponse(response, errOf(errorx.Wrap(e)))
			return
		}
		cm.writeResponse(response, successOf(nil))

	case "get", "GET":
		v, e := cm.get(values...)
		if e != nil {
			cm.writeResponse(response, errOf(errorx.Wrap(e)))
			return
		}
		cm.writeResponse(response, successOf(v))

	case "del", "DEL":
		if e := cm.delete(values...); e != nil {
			cm.writeResponse(response, errOf(errorx.Wrap(e)))
			return
		}
		cm.writeResponse(response, successOf(nil))
	}
}

// set
func (cm *ChanMap) set(values ...interface{}) error {
	offset := cm.offsetIncr()
	if len(values)%2 == 0 && len(values) >= 2 {
		for i := 0; i < len(values)-1; i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return errorx.NewFromStringf("key(values[%d]) must be a string type but got %s", i, reflect.TypeOf(values[i]).Name())
			}
			cm.m[key] = Value{
				v:      values[i+1],
				offset: offset,
				execAt: time.Now().UnixNano(),
				exp:    -1,
			}
		}
	} else {
		return errorx.NewFromStringf("command 'set' should have at least 2 values and value in pair, but got %v", values)
	}
	return nil
}

// get
func (cm *ChanMap) get(values ...interface{}) (interface{}, error) {
	if len(values) != 1 {
		return nil, errorx.NewFromStringf("command 'get' should have only one value, but got %v", values)
	}
	k, ok := values[0].(string)
	if !ok {
		return nil, errorx.NewFromStringf("command 'get' key requires string type but got '%v', typed '%s'", values[0], reflect.TypeOf(values[0]).Name())
	}
	return cm.m[k].v, nil
}

// delete
func (cm *ChanMap) delete(values ...interface{}) error {
	for i, v := range values {
		k, ok := v.(string)
		if !ok {
			return errorx.NewFromStringf("command 'delete' values should be string type but values[%d] got '%v' typed '%s'", i, v, reflect.TypeOf(v).Name())
		}
		delete(cm.m, k)
	}
	return nil
}

type Operation struct {
	command  string
	values   []interface{}
	response chan ChanMapRepsonse
}

func (set Operation) Command() string {
	return set.command
}
func (set Operation) Values() []interface{} {
	return set.values
}
func (set Operation) Response() chan ChanMapRepsonse {
	return set.response
}

func (cm *ChanMap) Set(key string, value interface{}) error {
	operation := Operation{
		command:  "SET",
		values:   []interface{}{key, value},
		response: make(chan ChanMapRepsonse, 1),
	}
	cm.recevOperation(operation)
	select {
	case <-time.After(10 * time.Second):
		return errorx.NewFromStringf("set command time out, no reponse")
	case v := <-operation.Response():
		if v.Err != nil {
			return errorx.Wrap(v.Err)
		}
		return nil
	}
}

func (cm *ChanMap) Get(key string) (interface{}, error) {
	operation := Operation{
		command:  "GET",
		values:   []interface{}{key},
		response: make(chan ChanMapRepsonse, 1),
	}
	cm.recevOperation(operation)
	select {
	case <-time.After(10 * time.Second):
		return nil, errorx.NewFromStringf("get command time out, no reponse")
	case v := <-operation.Response():
		if v.Err != nil {
			return nil, errorx.Wrap(v.Err)
		}
		return v.Response, nil
	}
}

func (cm *ChanMap) Delete(key string) error {
	operation := Operation{
		command:  "DEL",
		values:   []interface{}{key},
		response: make(chan ChanMapRepsonse, 1),
	}
	cm.recevOperation(operation)
	select {
	case <-time.After(10 * time.Second):
		return errorx.NewFromStringf("get command time out, no reponse")
	case v := <-operation.Response():
		if v.Err != nil {
			return errorx.Wrap(v.Err)
		}
		return nil
	}
}
