package userSafetyApi

import (
	"encoding/json"
	"errors"

	"github.com/gin-gonic/gin"
	"kyplugin"

	"server/app/controller/userSafetyApi/response"
	"server/app/logic/common/pluginCore"
	"server/app/models/constant"
)

// UserApi_插件调用 软件端调用原生插件注册的动作
// {"Api":"PluginCall","Plugin":"ads","Action":"getAds","Data":{...}}
// 成功响应的 Data 即插件动作的返回值
func UserApi_插件调用(c *gin.Context) {
	局_ctx := 取上下文(c)
	局_插件key := 局_ctx.Q请求明文.Get("Plugin").String()
	局_动作名 := 局_ctx.Q请求明文.Get("Action").String()
	if 局_插件key == "" || 局_动作名 == "" {
		response.FailMsg(c, constant.Status_参数错误, "Plugin和Action不能为空")
		return
	}

	var 局_数据 json.RawMessage
	if 局_Data := 局_ctx.Q请求明文.Get("Data"); !局_Data.IsNil() {
		局_数据, _ = json.Marshal(局_Data.Val())
	}
	局_在线 := 局_ctx.Z在线信息
	局_返回, err := pluginCore.L_插件.D调用客户端(局_插件key, 局_动作名, &kyplugin.ClientContext{
		AppId:    局_ctx.AppInfo.AppId,
		AppName:  局_ctx.AppInfo.AppName,
		LoggedIn: 检测用户登录在线正常(&局_ctx.Z在线信息),
		Online: kyplugin.Online{
			TokenId: 局_在线.Id, Uid: 局_在线.Uid, User: 局_在线.User, Key: 局_在线.Key,
			Ip: 局_在线.Ip, AppVer: 局_在线.AppVer, Tab: 局_在线.Tab, AgentUid: 局_在线.AgentUid,
		},
		Data: 局_数据,
	})
	if errors.Is(err, pluginCore.Err需要登录) {
		response.Fail(c, constant.Status_未登录)
		return
	}
	if err != nil {
		response.FailMsg(c, constant.Status_操作失败, err.Error())
		return
	}
	response.OkData(c, 局_返回)
}
