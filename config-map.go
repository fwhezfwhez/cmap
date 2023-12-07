package cmap

import (
	"fmt"
	"math"
	"time"
)

var DELETED = "cmap:DISABLE"

type ConfigMap struct {
	historyMap  *MapV2
	realtimeMap *MapV2
}

func NewConfigMap(hash func(string) int64, slotnum int, interval time.Duration) *ConfigMap {
	var cm = ConfigMap{
		historyMap:  NewMapV2(hash, slotnum, interval),
		realtimeMap: NewMapV2(hash, slotnum, interval),
	}

	return &cm
}
func (cm *ConfigMap) SetEx(key string, value interface{}, seconds int) {
	cm.refreshHistory(key, value)
	cm.realtimeMap.SetEx(key, value, seconds)
}

func (cm *ConfigMap) refreshHistory(key string, value interface{}) {
	cm.historyMap.Set(key, value)
	cm.historyMap.Delete(fmt.Sprintf("cmap:key_using_history_start_timeunix:%s", key))
}

func (cm *ConfigMap) onFirstUsingHistory(key string) {
	cm.historyMap.SetExNx(fmt.Sprintf("cmap:key_using_history_start_timeunix:%s", key), time.Now().Unix(), 60)
	cm.setLoadTimes(key)
}

// 获取载入过的次数
func (cm *ConfigMap) getLoadTimes(key string) int {
	rs, exist := cm.historyMap.Get(fmt.Sprintf("cmap:key_triger_load_times:%s", key))
	if exist {
		return getInt(rs)
	}
	return 0
}
func (cm *ConfigMap) setLoadTimes(key string) int64 {
	rs := cm.historyMap.IncrByEx(fmt.Sprintf("cmap:key_triger_load_times:%s", key), 1, 60*60*24)

	if rs >= math.MaxInt64-1000000 {
		cm.historyMap.SetEx(fmt.Sprintf("cmap:key_triger_load_times:%s", key), 100, 60*60*24)
	}
	return rs
}

// 对已经删除的配置，需要在外层显式调用SetDeleted，可以提高cmap性能,非必须
func (cm *ConfigMap) SetDeleted(key string) {
	cm.historyMap.SetEx(fmt.Sprintf("cmap:key_deleted:%s", key), DELETED, 15)
}

func (cm *ConfigMap) isDeleted(key string) bool {
	_, exist := cm.historyMap.Get(fmt.Sprintf("cmap:key_deleted:%s", key))

	if exist {
		return true
	}

	return false
}

func (cm *ConfigMap) getUsingHistoryStartUnix(key string) (int, bool) {
	startunix, exist := cm.historyMap.Get(fmt.Sprintf("cmap:key_using_history_start_timeunix:%s", key))
	if !exist {
		return 0, false
	}

	return getInt(startunix), true
}

// 返回 值，是否需要loading,是否获取到有效值
func (cm *ConfigMap) Get(key string) (interface{}, bool, bool) {
	// 对标记为删除的数据，60秒内直接返回无数据
	if cm.isDeleted(key) {
		return nil, false, false
	}

	rs, exist1 := cm.realtimeMap.Get(key)

	// 在实时map中找到了值
	if exist1 {
		return rs, false, true
	}

	var needloading bool
	oncecount := cm.historyMap.IncrByEx(fmt.Sprintf("cmap:key_loading_count:%s:%d", key, time.Now().Unix()/15), 1, 60)
	loadtimes := cm.getLoadTimes(key)

	needloading = countneedloading(oncecount, loadtimes)

	hist, exist2 := cm.historyMap.Get(key)

	// 在实时map里找不到，但是历史值里找到了，则使用历史值
	// 在使用历史值期间，外层调用方需要完成实时realtimemap的注入。
	if exist2 {

		startunix, existstarttime := cm.getUsingHistoryStartUnix(key)

		// 不存在则表示历史值第一次被使用,记录开始时间，并返回历史值
		if !existstarttime {
			cm.onFirstUsingHistory(key)
			return hist, needloading, true
		}

		// 如果从使用历史值开始，累计15秒没有完成注入，则历史值也应该被移除
		if time.Now().Unix() > int64(startunix+15) {
			cm.historyMap.Delete(key)

			return nil, needloading, false
		}

		return hist, needloading, true
	}

	// 如果都找不到，则第一条请求，触发载入会
	// 后续请求，直接返回 找不到
	return rs, needloading, false
}

func countneedloading(oncecount int64, loadtimes int) bool {

	// 当应用第一次启动，未载入配置时，所有请求允许直接击穿二级缓存，去读取源配置。(此场景，用redis来防止数据库击穿)
	// 之所以放开初次载入时的击穿，是因为初次启动时，并发下，非首个请求，可能会失败
	if loadtimes == 0 {
		return true
	}

	// 数据曾经载入过cmap，那么后续不再会出现击穿现象，因为会使用historymap来进行降级
	if oncecount == 1 {
		return true
	}

	return false
}

func getInt(i interface{}) int {
	switch v := i.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	}
	return 0
}
