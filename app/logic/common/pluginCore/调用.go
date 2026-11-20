package pluginCore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"runtime/debug"
	"strings"

	"kyplugin"

	"server/app/global"
	"server/app/logic/common/event"
)

var 错误_插件不可用 = errors.New("插件不存在或未启用")

// D调用客户端 软件端调用插件动作。未登录而动作要求登录时返回 Err需要登录
func (j *插件管理) D调用客户端(key, 动作名 string, 上下文 *kyplugin.ClientContext) (any, error) {
	局_实例 := j.取可用实例(key)
	if 局_实例 == nil || !j.应用已启用(key, 上下文.AppId) {
		return nil, 错误_插件不可用
	}
	局_动作, 局_存在 := 局_实例.客户端动作[动作名]
	if !局_存在 {
		return nil, errors.New("插件动作不存在")
	}
	if 局_动作.选项.RequireLogin && !上下文.LoggedIn {
		return nil, Err需要登录
	}
	局_ctx, 局_取消 := context.WithTimeout(context.Background(), 常量_调用超时)
	defer 局_取消()
	上下文.Ctx = 局_ctx
	return 安全执行(key, 动作名, func() (any, error) { return 局_动作.处理(上下文) })
}

// D调用管理端 管理端调用插件动作。插件停用时仍不可调用
func (j *插件管理) D调用管理端(key, 动作名 string, 上下文 *kyplugin.AdminContext) (any, error) {
	局_实例 := j.取可用实例(key)
	if 局_实例 == nil {
		return nil, 错误_插件不可用
	}
	局_处理, 局_存在 := 局_实例.管理端动作[动作名]
	if !局_存在 {
		return nil, errors.New("插件动作不存在")
	}
	局_ctx, 局_取消 := context.WithTimeout(context.Background(), 常量_调用超时)
	defer 局_取消()
	上下文.Ctx = 局_ctx
	return 安全执行(key, 动作名, func() (any, error) { return 局_处理(上下文) })
}

// Q取静态文件 读取插件注册的管理端静态文件
func (j *插件管理) Q取静态文件(key, 路径 string) ([]byte, error) {
	局_实例 := j.取实例(key)
	if 局_实例 == nil || 局_实例.静态 == nil {
		return nil, fs.ErrNotExist
	}
	路径 = strings.TrimLeft(路径, "/")
	if 路径 == "" {
		路径 = "index.html"
	}
	if !fs.ValidPath(路径) {
		return nil, fs.ErrNotExist
	}
	return fs.ReadFile(局_实例.静态, 路径)
}

// 安全执行 隔离插件 panic,并把返回值规范成纯 JSON 结构
func 安全执行(key, 动作名 string, 执行 func() (any, error)) (结果 any, 错误 error) {
	defer func() {
		if 局_异常 := recover(); 局_异常 != nil {
			global.GVA_LOG.Printf("[插件:%s] 动作[%s]发生异常: %v\n%s", key, 动作名, 局_异常, debug.Stack())
			结果, 错误 = nil, errors.New("插件内部错误")
		}
	}()
	局_返回, 局_错误 := 执行()
	if 局_错误 != nil {
		return nil, 局_错误
	}
	if 局_返回 == nil {
		return map[string]any{}, nil
	}
	局_json, 局_错误 := json.Marshal(局_返回)
	if 局_错误 != nil {
		return nil, fmt.Errorf("插件返回值无法序列化: %w", 局_错误)
	}
	var 局_规范值 any
	局_解码器 := json.NewDecoder(bytes.NewReader(局_json))
	if 局_错误 = 局_解码器.Decode(&局_规范值); 局_错误 != nil {
		return nil, 局_错误
	}
	return 局_规范值, nil
}

// 事件基名 "api.after:UserLogin" => "api.after"
func 事件基名(名称 string) string {
	if 局_位置 := strings.IndexByte(名称, ':'); 局_位置 > 0 {
		return 名称[:局_位置]
	}
	return 名称
}

// 取事件处理 收集一个插件对该事件(精确名+基名)的处理函数
func (j *插件实例) 取事件处理(名称 string) []kyplugin.EventHandler {
	局_处理 := j.事件[名称]
	if 局_基名 := 事件基名(名称); 局_基名 != 名称 {
		局_处理 = append(append([]kyplugin.EventHandler{}, 局_处理...), j.事件[局_基名]...)
	}
	return 局_处理
}

// 取事件订阅实例 已加载、已启用、(AppId>0时)对该应用启用 且订阅了该事件的插件
func (j *插件管理) 取事件订阅实例(名称 string, AppId int) map[string]*插件实例 {
	局_结果 := make(map[string]*插件实例)
	j.锁.RLock()
	for 局_key, 局_实例 := range j.已加载 {
		if len(局_实例.事件) > 0 {
			局_结果[局_key] = 局_实例
		}
	}
	j.锁.RUnlock()
	if len(局_结果) == 0 {
		return 局_结果
	}
	局_状态 := j.取状态()
	for 局_key, 局_实例 := range 局_结果 {
		局_插件状态 := 局_状态[局_key]
		if !局_插件状态.启用 || (AppId > 0 && !局_插件状态.应用[AppId].启用) || len(局_实例.取事件处理(名称)) == 0 {
			delete(局_结果, 局_key)
		}
	}
	return 局_结果
}

func (j *插件管理) 关注事件(名称 string, AppId int) bool {
	return len(j.取事件订阅实例(名称, AppId)) > 0
}

// 派发事件 事件总线回调(已在独立协程中)
func (j *插件管理) 派发事件(e event.S事件) {
	for 局_key, 局_实例 := range j.取事件订阅实例(e.M名称, e.AppId) {
		for _, 局_处理 := range 局_实例.取事件处理(e.M名称) {
			j.执行事件处理(局_key, 局_处理, e)
		}
	}
}

func (j *插件管理) 执行事件处理(key string, 处理 kyplugin.EventHandler, e event.S事件) {
	defer func() {
		if 局_异常 := recover(); 局_异常 != nil {
			global.GVA_LOG.Printf("[插件:%s] 事件[%s]处理发生异常: %v\n%s", key, e.M名称, 局_异常, debug.Stack())
		}
	}()
	局_ctx, 局_取消 := context.WithTimeout(context.Background(), 常量_调用超时)
	defer 局_取消()
	处理(局_ctx, kyplugin.Event{Name: e.M名称, AppId: e.AppId, Payload: e.S数据})
}
