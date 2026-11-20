package controller

import (
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"kyplugin"

	"server/app/controller/Common"
	"server/app/logic/common/pluginCore"
	"server/app/models/old/response"
)

const 插件_上传最大字节 = 200 << 20

// Plugin 原生插件(.so)管理
type Plugin struct {
	Common.Common
}

func NewPluginController() *Plugin {
	return &Plugin{}
}

type 请求_pluginKey struct {
	Key string `json:"key" binding:"required,min=2,max=32"`
}

type 请求_pluginSetStatus struct {
	Key    string `json:"key" binding:"required,min=2,max=32"`
	Status int    `json:"status" binding:"required,oneof=1 2"`
}

type 请求_pluginSaveApp struct {
	Key      string            `json:"key" binding:"required,min=2,max=32"`
	AppId    int               `json:"appId" binding:"required,min=10001"`
	Status   int               `json:"status" binding:"required,oneof=1 2"`
	Settings map[string]string `json:"settings"`
}

// GetList 已安装插件列表
func (j *Plugin) GetList(c *gin.Context) {
	局_列表, err := pluginCore.L_插件.Q取列表()
	if err != nil {
		response.FailWithMessage("获取失败:"+err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"list": 局_列表, "sdkVersion": kyplugin.SDKVersion}, "获取成功", c)
}

// Upload 上传 .so 文件。表单字段 file; 带 key 字段为升级已安装插件,否则为安装新插件
func (j *Plugin) Upload(c *gin.Context) {
	局_文件, err := c.FormFile("file")
	if err != nil {
		response.FailWithMessage("请选择插件文件", c)
		return
	}
	if filepath.Ext(局_文件.Filename) != ".so" {
		response.FailWithMessage("只支持 .so 插件文件", c)
		return
	}
	if 局_文件.Size <= 0 || 局_文件.Size > 插件_上传最大字节 {
		response.FailWithMessage("插件文件大小必须在 200MB 以内", c)
		return
	}
	//文件名由服务端生成,不使用上传的文件名
	局_临时文件 := filepath.Join(pluginCore.L_插件.Q取插件目录(), "upload-"+strconv.FormatInt(time.Now().UnixNano(), 10)+".tmp")
	if err = c.SaveUploadedFile(局_文件, 局_临时文件); err != nil {
		response.FailWithMessage("保存上传文件失败:"+err.Error(), c)
		return
	}

	if 局_key := c.PostForm("key"); 局_key != "" {
		if err = pluginCore.L_插件.S升级(局_key, 局_临时文件); err != nil {
			response.FailWithMessage("升级失败:"+err.Error(), c)
			return
		}
		response.OkWithMessage("升级文件已保存,重启系统后生效", c)
		return
	}
	局_信息, err := pluginCore.L_插件.A安装(局_临时文件)
	if err != nil {
		response.FailWithMessage("安装失败:"+err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"key": 局_信息.Key, "name": 局_信息.Name, "version": 局_信息.Version}, "安装成功", c)
}

// SetStatus 全局启用/停用
func (j *Plugin) SetStatus(c *gin.Context) {
	var 局_请求 请求_pluginSetStatus
	if !j.ToJSON(c, &局_请求) {
		return
	}
	if err := pluginCore.L_插件.Z置状态(局_请求.Key, 局_请求.Status); err != nil {
		response.FailWithMessage("操作失败:"+err.Error(), c)
		return
	}
	response.OkWithMessage("操作成功", c)
}

// Uninstall 卸载
func (j *Plugin) Uninstall(c *gin.Context) {
	var 局_请求 请求_pluginKey
	if !j.ToJSON(c, &局_请求) {
		return
	}
	if err := pluginCore.L_插件.X卸载(局_请求.Key); err != nil {
		response.FailWithMessage("卸载失败:"+err.Error(), c)
		return
	}
	response.OkWithMessage("已卸载,插件代码在重启系统后从内存移除", c)
}

// GetAppList 插件在各应用下的启用状态与配置
func (j *Plugin) GetAppList(c *gin.Context) {
	var 局_请求 请求_pluginKey
	if !j.ToJSON(c, &局_请求) {
		return
	}
	局_列表, err := pluginCore.L_插件.Q取应用配置(局_请求.Key)
	if err != nil {
		response.FailWithMessage("获取失败:"+err.Error(), c)
		return
	}
	response.OkWithDetailed(gin.H{"list": 局_列表}, "获取成功", c)
}

// SaveApp 保存插件在某应用下的启用状态与配置
func (j *Plugin) SaveApp(c *gin.Context) {
	var 局_请求 请求_pluginSaveApp
	if !j.ToJSON(c, &局_请求) {
		return
	}
	if err := pluginCore.L_插件.B保存应用配置(局_请求.Key, 局_请求.AppId, 局_请求.Status, 局_请求.Settings); err != nil {
		response.FailWithMessage("保存失败:"+err.Error(), c)
		return
	}
	response.OkWithMessage("保存成功", c)
}

// Call 调用插件注册的管理端动作  POST plugin/call/:key/:action  body为任意json
func (j *Plugin) Call(c *gin.Context) {
	局_数据, err := c.GetRawData()
	if err != nil {
		response.FailWithMessage("参数读取失败", c)
		return
	}
	局_返回, err := pluginCore.L_插件.D调用管理端(c.Param("key"), c.Param("action"), &kyplugin.AdminContext{
		AdminUid:  c.GetInt("Uid"),
		AdminUser: c.GetString("User"),
		Data:      局_数据,
	})
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithData(局_返回, c)
}

// UI 插件管理页(独立静态页,接口调用时才校验管理员令牌)
func (j *Plugin) UI(c *gin.Context) {
	插件_输出静态(c, "index.html", pluginCore.W管理页_首页)
}

// SdkJs 插件管理端页面使用的前端辅助脚本
func (j *Plugin) SdkJs(c *gin.Context) {
	插件_输出静态(c, "sdk.js", pluginCore.W管理页_脚本)
}

// Static 插件自带的管理端静态文件  GET plugin/static/:key/*filepath
func (j *Plugin) Static(c *gin.Context) {
	局_路径 := c.Param("filepath")
	局_内容, err := pluginCore.L_插件.Q取静态文件(c.Param("key"), 局_路径)
	if err != nil {
		c.String(http.StatusNotFound, "404 not found")
		return
	}
	if 局_路径 == "" || 局_路径 == "/" {
		局_路径 = "index.html"
	}
	插件_输出静态(c, 局_路径, 局_内容)
}

func 插件_输出静态(c *gin.Context, 文件名 string, 内容 []byte) {
	局_类型 := mime.TypeByExtension(path.Ext(文件名))
	if 局_类型 == "" {
		局_类型 = "application/octet-stream"
	}
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, 局_类型, 内容)
}
