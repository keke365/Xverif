// Package event 宿主事件总线(叶子包,不依赖任何业务包,任何层都可以发布事件)
// 目前唯一的订阅者是插件内核 logic/common/pluginCore
package event

import (
	"encoding/json"
	"sync/atomic"
)

// S事件 一次事件
type S事件 struct {
	M名称   string
	AppId int             //相关应用,无关填0
	S数据   json.RawMessage //事件数据json
}

type 订阅者 struct {
	处理 func(S事件)
	关注 func(名称 string, AppId int) bool
}

var 集_订阅者 atomic.Pointer[订阅者]

// D订阅 设置事件订阅者(后设置的覆盖先设置的)。关注 用于在发布前快速判断是否有人需要该事件
func D订阅(处理 func(S事件), 关注 func(名称 string, AppId int) bool) {
	集_订阅者.Store(&订阅者{处理: 处理, 关注: 关注})
}

// G关注 是否有人需要该事件。热路径上先判断,避免白白构造事件数据
func G关注(名称 string, AppId int) bool {
	局_订阅者 := 集_订阅者.Load()
	return 局_订阅者 != nil && 局_订阅者.关注(名称, AppId)
}

// F发布 异步发布事件,永不阻塞也永不影响调用方
func F发布(名称 string, AppId int, 数据 any) {
	局_订阅者 := 集_订阅者.Load()
	if 局_订阅者 == nil || !局_订阅者.关注(名称, AppId) {
		return
	}
	局_json, 局_错误 := json.Marshal(数据)
	if 局_错误 != nil {
		return
	}
	go func() {
		defer func() { _ = recover() }()
		局_订阅者.处理(S事件{M名称: 名称, AppId: AppId, S数据: 局_json})
	}()
}
