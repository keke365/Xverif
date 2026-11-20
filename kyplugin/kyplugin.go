// Package kyplugin 飞鸟快验插件 SDK。
//
// 本包是宿主(server)与插件(.so)之间唯一的契约,只依赖 Go 标准库。
// 宿主和插件必须编译同一份本包源码: 本包任何改动都会让旧插件无法加载,必须重新编译。
// 改动导出接口时请同时递增 SDKVersion。
//
// 插件入口(package main)必须导出:
//
//	func KyPlugin() kyplugin.Plugin
//
// 开发规范见 plugin-dev/README.md
package kyplugin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
)

// SDKVersion 契约版本号。插件 Info().SDKVersion 必须等于宿主的该值。
const SDKVersion = 1

// EntrySymbol 插件必须导出的入口函数名,类型为 func() Plugin
const EntrySymbol = "KyPlugin"

// Plugin 插件必须实现的接口
type Plugin interface {
	// Info 返回插件静态描述。必须是纯函数: 不访问数据库、不依赖 Init
	Info() Info
	// Init 插件加载时调用一次(安装时、以及之后每次宿主启动时)。
	// 在这里建表、注册接口/事件/定时任务。返回错误则插件加载失败。
	// host 可以保存下来在之后的处理函数里使用。
	Init(host Host) error
}

// Info 插件描述
type Info struct {
	Key         string // 唯一标识: 小写字母开头,仅含小写字母/数字/下划线,2-32位。同时是表名前缀和接口路径
	Name        string // 显示名称
	Version     string // 版本号,如 1.0.0
	Author      string
	Description string
	SDKVersion  int // 固定填 kyplugin.SDKVersion

	Settings []SettingField // 按应用的配置项,管理端自动生成表单
	Menus    []Menu         // 管理端页面入口
}

// SettingField 配置项定义。值统一以字符串存取
type SettingField struct {
	Key     string   // 配置键
	Label   string   // 显示名称
	Type    string   // text | textarea | number | switch | select
	Default string   // 默认值。switch 用 "1"/"0"
	Options []string // Type=select 时的可选值
	Tip     string   // 说明文字
}

// Menu 管理端页面入口,指向插件 Static 注册的静态文件
type Menu struct {
	Title string // 菜单标题
	Path  string // 相对静态目录的路径,如 "index.html"
}

// Host 宿主提供给插件的能力。
// Register* 方法只能在 Init 内调用; 其余方法在 Init 和处理函数内都可调用,且并发安全。
type Host interface {
	// ---------- 注册(仅 Init 内) ----------

	// RegisterClientAction 注册软件端可调用的动作。
	// 软件端通过加密接口调用: {"Api":"PluginCall","Plugin":"<key>","Action":"<name>","Data":{...}}
	// 只有对该应用启用了本插件才会被调用。
	RegisterClientAction(name string, opt ClientOption, fn ClientHandler)
	// RegisterAdminAction 注册管理端可调用的动作(已校验管理员令牌)。
	// POST /{管理入口}/plugin/call/<key>/<name>  body 为任意 JSON
	RegisterAdminAction(name string, fn AdminHandler)
	// RegisterEvent 订阅宿主事件,事件名见 Event* 常量。处理函数被异步调用,不能影响宿主原流程
	RegisterEvent(event string, fn EventHandler)
	// RegisterCron 注册定时任务。spec 为 6 位(含秒) cron 表达式,如 "0 */5 * * * *"
	RegisterCron(name, spec string, fn func(ctx context.Context)) error
	// RegisterStatic 注册管理端静态页面(通常是 embed.FS)。
	// 访问地址 /{管理入口}/plugin/static/<key>/<path>,无鉴权,不要放敏感内容
	RegisterStatic(files fs.FS)

	// ---------- 数据 ----------

	// DB 宿主的 MySQL 连接池。插件只应操作 TableName 返回的自有表
	DB() *sql.DB
	// TableName 返回插件自有表的完整表名: plg_<key>_<name>
	TableName(name string) string
	// Setting 读取本插件在某应用下的配置(已合并默认值)。应用未配置时返回默认值
	Setting(appId int) Settings
	// AppEnabled 本插件是否对该应用启用
	AppEnabled(appId int) bool
	// DataDir 插件专属数据目录(已创建),可读写文件
	DataDir() string

	// ---------- 宿主业务 ----------

	// GetUser 取账号信息(账号模式应用)。uid 为 Online.Uid
	GetUser(uid int) (User, error)
	// GetAppUser 取软件用户信息(到期时间/点数/积分等)
	GetAppUser(appId, uid int) (AppUser, error)
	// AddBalance 增减账号余额,amount 为负数表示扣减,余额不足返回错误。会写余额日志
	AddBalance(uid int, amount float64, note string) (newBalance float64, err error)
	// AddVipNumber 增减软件用户积分,amount 为负数表示扣减。会写积分日志
	AddVipNumber(appId, uid int, amount float64, note string) error
	// AddVipTime 增减到期时间(秒)或点数(计点应用),amount 为负数表示扣减。会写日志
	AddVipTime(appId, uid int, amount int64, note string) error
	// WsSend 向一个 WebSocket 在线连接推送消息,tokenId 为在线记录 Id
	WsSend(tokenId int, message []byte) error

	// Log 输出日志,自动带插件标识前缀
	Log(format string, args ...any)
}

// Settings 配置值集合
type Settings map[string]string

// Get 取字符串
func (s Settings) Get(key string) string { return s[key] }

// Bool switch 类型: "1"/"true" 为真
func (s Settings) Bool(key string) bool { return s[key] == "1" || s[key] == "true" }

// Int 取整数,非法值返回 0
func (s Settings) Int(key string) int {
	var 局_值 int
	_ = json.Unmarshal([]byte(s[key]), &局_值)
	return 局_值
}

// ClientOption 软件端动作选项
type ClientOption struct {
	RequireLogin bool // true: 未登录的令牌直接被宿主拒绝
}

// Online 发起请求的在线令牌信息。未登录时 Uid=0
type Online struct {
	TokenId  int    // 在线记录 Id,可用于 WsSend
	Uid      int    // 账号模式=账号 Id,卡号模式=卡号 Id
	User     string // 账号或卡号
	Key      string // 绑定信息(机器码)
	Ip       string
	AppVer   string
	Tab      string // 动态标签
	AgentUid int
}

// ClientContext 软件端调用上下文
type ClientContext struct {
	Ctx      context.Context
	AppId    int
	AppName  string
	LoggedIn bool
	Online   Online
	Data     json.RawMessage // 请求里的 Data 字段原文,可能为空
}

// Bind 把 Data 解析到 v。Data 为空时不改动 v
func (c *ClientContext) Bind(v any) error { return 解析数据(c.Data, v) }

// AdminContext 管理端调用上下文
type AdminContext struct {
	Ctx       context.Context
	AdminUid  int
	AdminUser string
	Data      json.RawMessage // 请求 body 原文
}

// Bind 把 Data 解析到 v。Data 为空时不改动 v
func (c *AdminContext) Bind(v any) error { return 解析数据(c.Data, v) }

func 解析数据(data json.RawMessage, v any) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if 局_错误 := json.Unmarshal(data, v); 局_错误 != nil {
		return errors.New("参数格式错误: " + 局_错误.Error())
	}
	return nil
}

// ClientHandler 软件端动作处理函数。返回值会被 JSON 序列化后放进响应的 Data;
// 返回 error 时其文本作为失败提示返回给软件端,不要包含敏感信息
type ClientHandler func(ctx *ClientContext) (any, error)

// AdminHandler 管理端动作处理函数,约定同 ClientHandler
type AdminHandler func(ctx *AdminContext) (any, error)

// EventHandler 事件处理函数
type EventHandler func(ctx context.Context, e Event)

// Event 宿主事件
type Event struct {
	Name    string
	AppId   int             // 相关应用,无关时为 0。>0 时仅派发给对该应用启用的插件
	Payload json.RawMessage // 事件数据 JSON,结构见各事件常量说明
}

// Bind 把 Payload 解析到 v
func (e Event) Bind(v any) error { return 解析数据(e.Payload, v) }

const (
	// EventApiAfter 任意软件端接口处理完成后触发。Payload 为 ApiAfterPayload。
	// 也可以订阅 "api.after:<Api名>" 只接收某一个接口,如 "api.after:UserLogin"
	EventApiAfter = "api.after"
	// EventPaySuccess 支付订单处理成功后触发。Payload 为 PaySuccessPayload
	EventPaySuccess = "pay.success"
)

// EventApiAfterOf 返回只订阅某个接口的事件名
func EventApiAfterOf(api string) string { return EventApiAfter + ":" + api }

// ApiAfterPayload EventApiAfter 的数据
type ApiAfterPayload struct {
	Api      string          `json:"api"`      // 接口名,如 UserLogin / NewUserInfo / UseKa
	Online   Online          `json:"online"`   // 请求时的令牌信息
	Request  json.RawMessage `json:"request"`  // 请求明文
	Response json.RawMessage `json:"response"` // 响应明文,Status=10000 等于客户端期望的成功码时为成功
}

// PaySuccessPayload EventPaySuccess 的数据
type PaySuccessPayload struct {
	PayOrder       string  `json:"payOrder"` // 订单号
	Uid            int     `json:"uid"`
	User           string  `json:"user"`
	AppId          int     `json:"appId"`
	Rmb            float64 `json:"rmb"`            // 支付金额
	PayType        string  `json:"payType"`        // 支付通道
	ProcessingType int     `json:"processingType"` // 订单类型: 余额充值/购卡直冲/支付购卡等
}

// User 账号信息
type User struct {
	Id           int
	User         string
	Phone        string
	Email        string
	Qq           string
	Status       int // 1正常 2冻结
	Rmb          float64
	RegisterTime int64
}

// AppUser 软件用户信息
type AppUser struct {
	Id           int
	Uid          int
	Status       int     // 1正常 2冻结
	Key          string  // 绑定信息
	VipTime      int64   // 到期时间戳或剩余点数
	VipNumber    float64 // 积分
	Note         string
	MaxOnline    int
	UserClassId  int
	RegisterTime int64
	AgentUid     int
}
