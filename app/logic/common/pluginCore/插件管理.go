// Package pluginCore 原生插件(.so)内核: 加载、安装、升级、卸载、启停与按应用配置。
//
// 约束(Go plugin 机制决定,无法绕过):
//  1. 插件必须与宿主用同一 Go 版本、同一份 kyplugin SDK、相同构建参数(-trimpath)编译
//  2. 已加载的代码无法卸载,同一插件在一个进程内只能加载一次: 升级/卸载后需重启才彻底生效
//  3. 插件与宿主同进程、同权限,没有沙箱
package pluginCore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	goPlugin "plugin"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"kyplugin"

	"server/app/global"
	"server/app/logic/common/event"
	"server/app/models/dbm"
	"server/app/service"
)

const (
	常量_状态启用   = 1
	常量_状态停用   = 2
	常量_状态缓存秒  = 15
	常量_调用超时   = 30 * time.Second
	常量_升级后缀   = ".new"
	常量_备份后缀   = ".bak"
	常量_失败后缀   = ".failed"
	常量_插件目录名  = "plugins"
	常量_插件数据目录 = "data"
)

var 集_Key正则 = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}$`)

// Err需要登录 软件端动作要求登录但令牌未登录
var Err需要登录 = errors.New("未登录")

// L_插件 插件内核单例
var L_插件 = &插件管理{已加载: make(map[string]*插件实例)}

type 插件管理 struct {
	锁    sync.RWMutex
	已加载  map[string]*插件实例
	安装锁  sync.Mutex //安装/升级/卸载串行
	状态   atomic.Pointer[map[string]状态_插件]
	状态时间 atomic.Int64
}

type 插件实例 struct {
	信息    kyplugin.Info
	客户端动作 map[string]客户端动作
	管理端动作 map[string]kyplugin.AdminHandler
	事件    map[string][]kyplugin.EventHandler
	静态    fs.FS
}

type 客户端动作 struct {
	选项 kyplugin.ClientOption
	处理 kyplugin.ClientHandler
}

type 状态_插件 struct {
	启用 bool
	应用 map[int]状态_应用
}

type 状态_应用 struct {
	启用 bool
	配置 map[string]string
}

// Q取插件目录 运行目录/plugins,不存在则创建
func (j *插件管理) Q取插件目录() string {
	局_目录 := filepath.Join(global.GVA_CONFIG.Q取运行目录, 常量_插件目录名)
	_ = os.MkdirAll(局_目录, 0o755)
	return 局_目录
}

// J加载全部 宿主启动时调用一次: 应用待生效的升级文件,加载数据库中登记的全部插件
func (j *插件管理) J加载全部() {
	//订阅必须在判断数据库之前: 首次部署时宿主启动还没有数据库(在管理端初始化后才有),
	//不重启就安装的插件也要能收到事件。没有已加载的插件时 关注事件 直接返回 false,不访问数据库
	event.D订阅(j.派发事件, j.关注事件)
	if global.GVA_DB == nil {
		return
	}

	局_目录 := j.Q取插件目录()
	局_服务 := service.NewPlugin(global.Get局db())
	局_列表, 局_错误 := 局_服务.List()
	if 局_错误 != nil {
		global.GVA_LOG.Println("插件列表读取失败:" + 局_错误.Error())
		return
	}
	for _, 局_记录 := range 局_列表 {
		局_路径 := filepath.Join(局_目录, 局_记录.File)
		局_已升级 := false
		if 文件存在(局_路径 + 常量_升级后缀) {
			_ = os.Remove(局_路径 + 常量_备份后缀)
			_ = os.Rename(局_路径, 局_路径+常量_备份后缀)
			if 局_改名错误 := os.Rename(局_路径+常量_升级后缀, 局_路径); 局_改名错误 == nil {
				局_已升级 = true
			}
		}

		局_实例, 局_加载错误 := j.加载文件(局_路径, 局_记录.Key)
		if 局_加载错误 != nil {
			局_原因 := 局_加载错误.Error()
			if 局_已升级 && 文件存在(局_路径+常量_备份后缀) {
				_ = os.Remove(局_路径 + 常量_失败后缀)
				_ = os.Rename(局_路径, 局_路径+常量_失败后缀)
				_ = os.Rename(局_路径+常量_备份后缀, 局_路径)
				局_原因 += " | 新版本加载失败,已还原旧版本文件,再次重启后恢复旧版本"
			}
			global.GVA_LOG.Println("插件[" + 局_记录.Key + "]加载失败:" + 局_原因)
			_ = 局_服务.Update(局_记录.Key, map[string]interface{}{"load_error": 截断文本(局_原因, 1900)})
			continue
		}

		j.锁.Lock()
		j.已加载[局_记录.Key] = 局_实例
		j.锁.Unlock()

		局_更新 := map[string]interface{}{
			"name": 局_实例.信息.Name, "version": 局_实例.信息.Version, "author": 局_实例.信息.Author,
			"description": 局_实例.信息.Description, "load_error": "",
		}
		if 局_已升级 {
			局_更新["sha256"] = 文件Sha256(局_路径)
			局_更新["update_time"] = time.Now().Unix()
		}
		_ = 局_服务.Update(局_记录.Key, 局_更新)
		global.GVA_LOG.Println("插件[" + 局_记录.Key + "] v" + 局_实例.信息.Version + " 加载成功")
	}
	j.S刷新状态()
}

// A安装 安装一个新插件。临时文件 为已保存到插件目录下的上传文件,无论成败本函数负责处理它
func (j *插件管理) A安装(临时文件 string) (kyplugin.Info, error) {
	j.安装锁.Lock()
	defer j.安装锁.Unlock()

	局_插件, 局_信息, 局_错误 := j.打开文件(临时文件)
	if 局_错误 != nil {
		_ = os.Remove(临时文件)
		return 局_信息, 局_错误
	}
	局_服务 := service.NewPlugin(global.Get局db())
	if _, 局_查询错误 := 局_服务.Info(局_信息.Key); 局_查询错误 == nil {
		_ = os.Remove(临时文件)
		return 局_信息, errors.New("插件[" + 局_信息.Key + "]已安装,更新版本请在列表中使用「升级」")
	}

	局_实例, 局_错误 := j.初始化(局_插件, 局_信息)
	if 局_错误 != nil {
		_ = os.Remove(临时文件)
		return 局_信息, errors.New(局_错误.Error() + " (该文件的代码已载入本进程,修正后需重启宿主才能再次安装)")
	}

	局_文件名 := 局_信息.Key + ".so"
	局_路径 := filepath.Join(j.Q取插件目录(), 局_文件名)
	if 局_错误 = os.Rename(临时文件, 局_路径); 局_错误 != nil {
		_ = os.Remove(临时文件)
		return 局_信息, errors.New("保存插件文件失败:" + 局_错误.Error())
	}
	局_现在 := time.Now().Unix()
	局_错误 = 局_服务.Create(&dbm.DB_plugin{
		Key: 局_信息.Key, Name: 局_信息.Name, Version: 局_信息.Version, Author: 局_信息.Author,
		Description: 局_信息.Description, Status: 常量_状态启用, File: 局_文件名,
		Sha256: 文件Sha256(局_路径), CreateTime: 局_现在, UpdateTime: 局_现在,
	})
	if 局_错误 != nil {
		_ = os.Remove(局_路径)
		return 局_信息, errors.New("写入插件记录失败:" + 局_错误.Error())
	}

	j.锁.Lock()
	j.已加载[局_信息.Key] = 局_实例
	j.锁.Unlock()
	j.S刷新状态()
	global.GVA_LOG.Println("插件[" + 局_信息.Key + "] v" + 局_信息.Version + " 安装成功")
	return 局_信息, nil
}

// S升级 保存新版本文件,下次宿主启动时替换生效(加载失败会自动还原旧版本文件)
func (j *插件管理) S升级(key, 临时文件 string) error {
	j.安装锁.Lock()
	defer j.安装锁.Unlock()

	局_记录, 局_错误 := service.NewPlugin(global.Get局db()).Info(key)
	if 局_错误 != nil {
		_ = os.Remove(临时文件)
		return errors.New("插件不存在")
	}
	局_目标 := filepath.Join(j.Q取插件目录(), 局_记录.File) + 常量_升级后缀
	if 局_错误 = os.Rename(临时文件, 局_目标); 局_错误 != nil {
		_ = os.Remove(临时文件)
		return errors.New("保存升级文件失败:" + 局_错误.Error())
	}
	global.GVA_LOG.Println("插件[" + key + "]已上传升级文件,重启后生效")
	return nil
}

// X卸载 删除插件记录、应用配置和文件。已载入的代码要到重启才从内存消失,但立即不再被调用。
// 插件自建的数据表(plg_<key>_*)和数据目录不会删除
func (j *插件管理) X卸载(key string) error {
	j.安装锁.Lock()
	defer j.安装锁.Unlock()

	局_服务 := service.NewPlugin(global.Get局db())
	局_记录, 局_错误 := 局_服务.Info(key)
	if 局_错误 != nil {
		return errors.New("插件不存在")
	}
	if 局_错误 = 局_服务.Delete(key); 局_错误 != nil {
		return 局_错误
	}
	j.锁.Lock()
	delete(j.已加载, key)
	j.锁.Unlock()

	局_路径 := filepath.Join(j.Q取插件目录(), 局_记录.File)
	for _, 局_后缀 := range []string{"", 常量_升级后缀, 常量_备份后缀, 常量_失败后缀} {
		_ = os.Remove(局_路径 + 局_后缀)
	}
	j.S刷新状态()
	global.GVA_LOG.Println("插件[" + key + "]已卸载")
	return nil
}

// Z置状态 启用或停用插件(全局)
func (j *插件管理) Z置状态(key string, 状态 int) error {
	if 状态 != 常量_状态启用 && 状态 != 常量_状态停用 {
		return errors.New("状态只能是1或2")
	}
	局_服务 := service.NewPlugin(global.Get局db())
	if _, 局_错误 := 局_服务.Info(key); 局_错误 != nil {
		return errors.New("插件不存在")
	}
	if 局_错误 := 局_服务.Update(key, map[string]interface{}{"status": 状态, "update_time": time.Now().Unix()}); 局_错误 != nil {
		return 局_错误
	}
	j.S刷新状态()
	return nil
}

// X响应_插件项 管理端列表项
type X响应_插件项 struct {
	dbm.DB_plugin
	Loaded         bool                    `json:"loaded"`         //本进程是否已成功加载
	LoadedVersion  string                  `json:"loadedVersion"`  //内存中实际运行的版本
	PendingUpgrade bool                    `json:"pendingUpgrade"` //有升级文件等待重启生效
	Settings       []kyplugin.SettingField `json:"settings"`
	Menus          []kyplugin.Menu         `json:"menus"`
}

// Q取列表 管理端插件列表
func (j *插件管理) Q取列表() ([]X响应_插件项, error) {
	局_列表, 局_错误 := service.NewPlugin(global.Get局db()).List()
	if 局_错误 != nil {
		return nil, 局_错误
	}
	局_目录 := j.Q取插件目录()
	局_结果 := make([]X响应_插件项, 0, len(局_列表))
	for _, 局_记录 := range 局_列表 {
		局_项 := X响应_插件项{DB_plugin: 局_记录, Settings: []kyplugin.SettingField{}, Menus: []kyplugin.Menu{}}
		if 局_实例 := j.取实例(局_记录.Key); 局_实例 != nil {
			局_项.Loaded = true
			局_项.LoadedVersion = 局_实例.信息.Version
			if 局_实例.信息.Settings != nil {
				局_项.Settings = 局_实例.信息.Settings
			}
			if 局_实例.信息.Menus != nil {
				局_项.Menus = 局_实例.信息.Menus
			}
		}
		局_项.PendingUpgrade = 文件存在(filepath.Join(局_目录, 局_记录.File) + 常量_升级后缀)
		局_结果 = append(局_结果, 局_项)
	}
	return 局_结果, nil
}

// X响应_应用配置项 某插件在一个应用下的状态与配置
type X响应_应用配置项 struct {
	AppId    int               `json:"appId"`
	AppName  string            `json:"appName"`
	Status   int               `json:"status"` //1启用 2停用
	Settings map[string]string `json:"settings"`
}

// Q取应用配置 列出全部应用及该插件在其下的状态与配置(已合并默认值)
func (j *插件管理) Q取应用配置(key string) ([]X响应_应用配置项, error) {
	db := global.Get局db()
	if _, 局_错误 := service.NewPlugin(db).Info(key); 局_错误 != nil {
		return nil, errors.New("插件不存在")
	}
	var 局_应用列表 []dbm.DB_AppInfo
	if 局_错误 := db.Model(dbm.DB_AppInfo{}).Select("AppId", "AppName").Where("AppId > 10000").Order("AppId").Find(&局_应用列表).Error; 局_错误 != nil {
		return nil, 局_错误
	}
	j.S刷新状态()
	局_插件状态 := j.取状态()[key]
	局_结果 := make([]X响应_应用配置项, 0, len(局_应用列表))
	for _, 局_应用 := range 局_应用列表 {
		局_项 := X响应_应用配置项{AppId: 局_应用.AppId, AppName: 局_应用.AppName, Status: 常量_状态停用}
		if 局_插件状态.应用[局_应用.AppId].启用 {
			局_项.Status = 常量_状态启用
		}
		局_项.Settings = j.合并配置(key, 局_插件状态.应用[局_应用.AppId].配置)
		局_结果 = append(局_结果, 局_项)
	}
	return 局_结果, nil
}

// B保存应用配置 保存插件在某应用下的启用状态与配置。只保留插件声明过的配置键
func (j *插件管理) B保存应用配置(key string, AppId, 状态 int, 配置 map[string]string) error {
	if 状态 != 常量_状态启用 && 状态 != 常量_状态停用 {
		return errors.New("状态只能是1或2")
	}
	if AppId <= 10000 {
		return errors.New("AppId错误")
	}
	局_服务 := service.NewPlugin(global.Get局db())
	if _, 局_错误 := 局_服务.Info(key); 局_错误 != nil {
		return errors.New("插件不存在")
	}
	局_有效配置 := make(map[string]string, len(配置))
	if 局_实例 := j.取实例(key); 局_实例 != nil {
		for _, 局_字段 := range 局_实例.信息.Settings {
			if 局_值, 局_存在 := 配置[局_字段.Key]; 局_存在 {
				局_有效配置[局_字段.Key] = 局_值
			}
		}
	} else {
		局_有效配置 = 配置
	}
	局_json, _ := json.Marshal(局_有效配置)
	局_错误 := 局_服务.AppSave(&dbm.DB_pluginApp{
		Key: key, AppId: AppId, Status: 状态, Settings: string(局_json), UpdateTime: time.Now().Unix(),
	})
	if 局_错误 != nil {
		return 局_错误
	}
	j.S刷新状态()
	return nil
}

// S刷新状态 从数据库重建启用状态与配置快照
func (j *插件管理) S刷新状态() {
	局_新状态 := make(map[string]状态_插件)
	if global.GVA_DB != nil {
		局_服务 := service.NewPlugin(global.Get局db())
		局_插件列表, 局_错误1 := 局_服务.List()
		局_应用列表, 局_错误2 := 局_服务.AppList("")
		if 局_错误1 != nil || 局_错误2 != nil {
			j.状态时间.Store(time.Now().Unix()) //数据库异常时沿用旧快照,避免每个请求都打库
			return
		}
		for _, 局_记录 := range 局_插件列表 {
			局_新状态[局_记录.Key] = 状态_插件{启用: 局_记录.Status == 常量_状态启用, 应用: make(map[int]状态_应用)}
		}
		for _, 局_记录 := range 局_应用列表 {
			局_插件状态, 局_存在 := 局_新状态[局_记录.Key]
			if !局_存在 {
				continue
			}
			局_配置 := make(map[string]string)
			_ = json.Unmarshal([]byte(局_记录.Settings), &局_配置)
			局_插件状态.应用[局_记录.AppId] = 状态_应用{启用: 局_记录.Status == 常量_状态启用, 配置: 局_配置}
		}
	}
	j.状态.Store(&局_新状态)
	j.状态时间.Store(time.Now().Unix())
}

// 取状态 返回只读快照; 超过缓存时间则重建(兼顾多节点部署时其他节点的修改)
func (j *插件管理) 取状态() map[string]状态_插件 {
	if j.状态.Load() == nil || time.Now().Unix()-j.状态时间.Load() > 常量_状态缓存秒 {
		j.S刷新状态()
	}
	if 局_状态 := j.状态.Load(); 局_状态 != nil {
		return *局_状态
	}
	return map[string]状态_插件{}
}

func (j *插件管理) 取实例(key string) *插件实例 {
	j.锁.RLock()
	defer j.锁.RUnlock()
	return j.已加载[key]
}

// 取可用实例 已加载且全局启用才返回
func (j *插件管理) 取可用实例(key string) *插件实例 {
	局_实例 := j.取实例(key)
	if 局_实例 == nil || !j.取状态()[key].启用 {
		return nil
	}
	return 局_实例
}

func (j *插件管理) 应用已启用(key string, AppId int) bool {
	局_插件状态 := j.取状态()[key]
	return 局_插件状态.启用 && 局_插件状态.应用[AppId].启用
}

// 合并配置 已保存的值覆盖插件声明的默认值
func (j *插件管理) 合并配置(key string, 已保存 map[string]string) map[string]string {
	局_结果 := make(map[string]string, len(已保存))
	if 局_实例 := j.取实例(key); 局_实例 != nil {
		for _, 局_字段 := range 局_实例.信息.Settings {
			局_结果[局_字段.Key] = 局_字段.Default
		}
	}
	for 局_键, 局_值 := range 已保存 {
		局_结果[局_键] = 局_值
	}
	return 局_结果
}

// 加载文件 打开并初始化。期望key 非空时校验文件确实是该插件
func (j *插件管理) 加载文件(路径, 期望key string) (*插件实例, error) {
	局_插件, 局_信息, 局_错误 := j.打开文件(路径)
	if 局_错误 != nil {
		return nil, 局_错误
	}
	if 期望key != "" && 局_信息.Key != 期望key {
		return nil, errors.New("文件内的插件标识是[" + 局_信息.Key + "],与登记的[" + 期望key + "]不符")
	}
	return j.初始化(局_插件, 局_信息)
}

// 打开文件 载入 so 并取得、校验插件描述。注意: 载入即执行插件的 init 代码
func (j *插件管理) 打开文件(路径 string) (局_插件 kyplugin.Plugin, 局_信息 kyplugin.Info, 错误 error) {
	defer func() {
		if 局_异常 := recover(); 局_异常 != nil {
			错误 = fmt.Errorf("插件入口发生异常: %v", 局_异常)
		}
	}()
	局_so, 局_错误 := goPlugin.Open(路径)
	if 局_错误 != nil {
		return nil, 局_信息, 翻译打开错误(局_错误)
	}
	局_符号, 局_错误 := 局_so.Lookup(kyplugin.EntrySymbol)
	if 局_错误 != nil {
		return nil, 局_信息, errors.New("插件未导出入口函数 " + kyplugin.EntrySymbol)
	}
	局_入口, 局_类型正确 := 局_符号.(func() kyplugin.Plugin)
	if !局_类型正确 {
		return nil, 局_信息, errors.New("入口 " + kyplugin.EntrySymbol + " 的类型必须是 func() kyplugin.Plugin")
	}
	局_插件 = 局_入口()
	if 局_插件 == nil {
		return nil, 局_信息, errors.New("入口 " + kyplugin.EntrySymbol + " 返回了 nil")
	}
	局_信息 = 局_插件.Info()
	if !集_Key正则.MatchString(局_信息.Key) {
		return nil, 局_信息, errors.New("插件Key不合法: 需小写字母开头,仅含小写字母/数字/下划线,2-32位")
	}
	if 局_信息.SDKVersion != kyplugin.SDKVersion {
		return nil, 局_信息, fmt.Errorf("插件SDK版本(%d)与宿主(%d)不一致,请用宿主的 kyplugin 重新编译", 局_信息.SDKVersion, kyplugin.SDKVersion)
	}
	if 局_信息.Name == "" || 局_信息.Version == "" {
		return nil, 局_信息, errors.New("插件 Name 和 Version 不能为空")
	}
	return 局_插件, 局_信息, nil
}

// 初始化 调用插件 Init 完成注册
func (j *插件管理) 初始化(插件 kyplugin.Plugin, 信息 kyplugin.Info) (实例 *插件实例, 错误 error) {
	if j.取实例(信息.Key) != nil {
		return nil, errors.New("插件[" + 信息.Key + "]已加载")
	}
	局_实例 := &插件实例{
		信息:    信息,
		客户端动作: make(map[string]客户端动作),
		管理端动作: make(map[string]kyplugin.AdminHandler),
		事件:    make(map[string][]kyplugin.EventHandler),
	}
	局_宿主 := &插件宿主{key: 信息.Key, 实例: 局_实例}
	局_宿主.可注册.Store(true)
	defer 局_宿主.可注册.Store(false)
	defer func() {
		if 局_异常 := recover(); 局_异常 != nil {
			实例, 错误 = nil, fmt.Errorf("插件 Init 发生异常: %v", 局_异常)
		}
	}()
	if 局_错误 := 插件.Init(局_宿主); 局_错误 != nil {
		return nil, errors.New("插件 Init 失败: " + 局_错误.Error())
	}
	return 局_实例, nil
}

func 翻译打开错误(err error) error {
	局_文本 := err.Error()
	switch {
	case strings.Contains(局_文本, "different version of package"):
		return errors.New("插件与宿主的编译环境不一致。请用与宿主相同的 Go 版本、宿主仓库里的 kyplugin SDK、plugin-dev/build.sh 重新编译;插件引用的第三方库若宿主也在用,版本必须相同。原始错误: " + 局_文本)
	case strings.Contains(局_文本, "plugin already loaded"):
		return errors.New("同一插件(go.mod 的 module 名相同)已在本进程载入过。新装请重启宿主后再试;更新版本请在列表中使用「升级」")
	case strings.Contains(局_文本, "not implemented"):
		return errors.New("宿主未启用插件支持: 需在 Linux 下以 CGO_ENABLED=1 编译宿主")
	}
	return errors.New("插件文件载入失败: " + 局_文本)
}

func 文件存在(路径 string) bool {
	_, 局_错误 := os.Stat(路径)
	return 局_错误 == nil
}

func 文件Sha256(路径 string) string {
	局_文件, 局_错误 := os.Open(路径)
	if 局_错误 != nil {
		return ""
	}
	defer 局_文件.Close()
	局_哈希 := sha256.New()
	if _, 局_错误 = io.Copy(局_哈希, 局_文件); 局_错误 != nil {
		return ""
	}
	return hex.EncodeToString(局_哈希.Sum(nil))
}

func 截断文本(文本 string, 最大字符数 int) string {
	局_字符 := []rune(文本)
	if len(局_字符) <= 最大字符数 {
		return 文本
	}
	return string(局_字符[:最大字符数])
}
