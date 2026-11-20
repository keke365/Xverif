# 飞鸟快验 原生插件开发规范

插件是一个用 Go 编写、编译成 `.so` 的动态库。在管理端上传即安装,无需改动宿主代码。
一个插件可以同时提供:软件端接口、管理端接口、管理端页面、按应用的配置项、自己的数据表、事件处理、定时任务。

完整示例见 [examples/ads](examples/ads)(广告投放)。

## 1. 先了解三条硬限制

这三条来自 Go 的 plugin 机制,宿主无法绕过。

| 限制 | 影响 | 应对 |
|---|---|---|
| 插件与宿主必须"同一环境"编译 | Go 版本、构建参数、`kyplugin` SDK 源码、双方都用到的第三方库版本,任何一项不同都会加载失败 | 只用 [build.sh](build.sh) 编译宿主和插件;插件尽量只依赖标准库 |
| 代码载入后无法卸载 | 安装即时生效;**升级、卸载后要重启宿主**才彻底生效 | 停用是即时的;升级文件在下次启动时替换,加载失败自动还原旧文件 |
| 没有沙箱 | 插件在宿主进程内运行,能读写数据库、文件、内存中的一切 | **只安装审核过源码的插件。** 拿到别人的 `.so` 不要直接装,要源码自己编译 |

只支持 Linux(以及 macOS/FreeBSD),不支持 Windows。服务器需要安装 `gcc`。

## 2. 快速开始

```bash
# 1. 编译宿主(必须用这个脚本,不要直接 go build; make run / make build 内部调用的也是它)
plugin-dev/build.sh host

# 2. 复制示例作为起点
cp -r plugin-dev/examples/ads plugin-dev/examples/myplugin
#    修改 go.mod 的 module 名、main.go 里的 Key

# 3. 编译插件 -> plugin-dev/examples/myplugin/myplugin.so
plugin-dev/build.sh plugin plugin-dev/examples/myplugin

# 4. 管理后台「二开扩展 → 插件管理」,上传 .so
#    备用入口(功能相同的独立页): http://服务器/{管理入口}/plugin/ui ,需先在同一浏览器登录管理后台
# 5. 在「应用配置」里对需要的应用启用
```

宿主重新编译后(升级 Go、更新宿主代码、修改 SDK),**所有插件都要重新编译**。

## 3. 目录结构

```
myplugin/
├── go.mod        # module 名全局唯一,建议 kyplugin_<Key>
├── main.go       # package main,导出 KyPlugin
└── static/       # 可选,管理端页面,用 embed 打进 .so
    └── index.html
```

`go.mod`:

```
module kyplugin_myplugin

go 1.25.0

require kyplugin v0.0.0

replace kyplugin => ../../../kyplugin   // 指向宿主仓库的 kyplugin 目录
```

两个插件的 module 名相同会被 Go 当成同一个插件,后装的无法加载。

## 4. 入口与生命周期

```go
package main

import "kyplugin"

type 我的插件 struct{ host kyplugin.Host }

// 名称和签名固定
func KyPlugin() kyplugin.Plugin { return &我的插件{} }

func (j *我的插件) Info() kyplugin.Info {
	return kyplugin.Info{
		Key: "myplugin", Name: "我的插件", Version: "1.0.0",
		SDKVersion: kyplugin.SDKVersion,
	}
}

func (j *我的插件) Init(host kyplugin.Host) error {
	j.host = host
	// 建表、注册动作/事件/定时任务
	return nil
}
```

| 时机 | 发生什么 |
|---|---|
| 上传安装 | 载入 `.so` → `Info()` → 校验 → `Init()` → 登记入库,立即可用 |
| 宿主每次启动 | 对每个已登记插件:载入 → `Info()` → `Init()` |
| 停用 / 对某应用停用 | 不再调用动作和事件,定时任务空跑。即时生效 |
| 升级 | 新文件存为 `<key>.so.new`,下次启动替换。新版加载失败则还原旧文件,再重启一次恢复 |
| 卸载 | 删除登记、应用配置和文件;立即不再被调用。**数据表和数据目录保留** |

规则:

- `Info()` 必须是纯函数,不访问数据库,不依赖 `Init`。
- `Init()` 每次启动都会执行,里面的操作必须可重复执行(建表用 `CREATE TABLE IF NOT EXISTS`)。
- `Init()` 返回错误则加载失败,原因显示在管理页。
- `Register*` 方法只能在 `Init()` 内调用,之后调用会 panic。
- 没有 `Stop` 回调。不要在插件里启动无法结束的后台协程;周期性工作用 `RegisterCron`。
- `Key`:小写字母开头,只含小写字母、数字、下划线,2–32 位。发布后不要修改。

## 5. 宿主能力(kyplugin.Host)

权威定义见 [kyplugin/kyplugin.go](../kyplugin/kyplugin.go)。

### 5.1 软件端动作

```go
host.RegisterClientAction("getAds", kyplugin.ClientOption{RequireLogin: false}, func(ctx *kyplugin.ClientContext) (any, error) {
	var 请求 struct{ Id int `json:"id"` }
	if err := ctx.Bind(&请求); err != nil {
		return nil, err
	}
	return map[string]any{"list": []string{}}, nil
})
```

软件端通过现有的加密接口调用,加密、签名、令牌校验都由宿主完成:

```json
{"Api":"PluginCall","Plugin":"ads","Action":"getAds","Data":{"id":1}}
```

成功响应里的 `Data` 就是处理函数的返回值。

- 只有对该应用启用了插件才会被调用。
- `RequireLogin: true` 时未登录令牌由宿主直接拒绝(状态码 210)。
- `ctx.Online.Uid`:账号模式是账号 Id,卡号模式是卡号 Id;未登录为 0。
- 返回的 `error` 文本会原样发给软件端,不要带数据库错误等内部信息。
- 单次调用的 `ctx.Ctx` 30 秒超时,数据库操作请传入它。

### 5.2 管理端动作

```go
host.RegisterAdminAction("list", func(ctx *kyplugin.AdminContext) (any, error) { ... })
```

地址:`POST /{管理入口}/plugin/call/<key>/<动作名>`,请求体为任意 JSON,宿主已校验管理员令牌。
响应:`{"code":10000,"data":<返回值>,"msg":"ok"}`,失败时 `code` 为 200。

### 5.3 管理端页面

```go
//go:embed static
var 静态 embed.FS

sub, _ := fs.Sub(静态, "static")
host.RegisterStatic(sub)
// Info().Menus 里声明入口: {Title: "广告管理", Path: "index.html"}
```

页面地址:`/{管理入口}/plugin/static/<key>/<路径>`,入口按钮显示在插件管理页(管理后台里点击后在弹窗内打开,也可以新窗口打开)。

页面里引入宿主脚本后即可调用自己的管理端动作:

```html
<script src="../../sdk.js"></script>
<script>
  kyplugin.call('list', {page: 1}).then(function (data) { ... }).catch(function (e) { alert(e.message) });
</script>
```

- 静态文件本身**不鉴权**(和宿主前端一样),不要把密钥等敏感内容写进页面;数据一律通过动作获取。
- 页面里显示数据用 `textContent`,不要拼 `innerHTML`。
- 页面是普通静态页,可以用任何前端技术,把构建产物放进 `static/` 即可。

### 5.4 按应用的配置

在 `Info().Settings` 声明,管理页会自动生成表单:

```go
Settings: []kyplugin.SettingField{
	{Key: "max", Label: "数量", Type: "number", Default: "5"},
	{Key: "open", Label: "开关", Type: "switch", Default: "1"},
	{Key: "mode", Label: "模式", Type: "select", Options: []string{"a", "b"}, Default: "a"},
},
```

读取:`host.Setting(appId).Get("mode")` / `.Int("max")` / `.Bool("open")`。
值统一按字符串存储;`Type` 可选 `text` `textarea` `number` `switch` `select`。未保存过时返回默认值。

### 5.5 数据存储

```go
表 := host.TableName("item")   // plg_<key>_item
host.DB().ExecContext(ctx, "INSERT INTO `"+表+"` (`title`) VALUES (?)", 标题)
```

- `host.DB()` 是标准库 `*sql.DB`,与宿主共用连接池(MySQL)。
- 只操作 `host.TableName()` 得到的自有表。需要宿主数据时用下面的业务方法,不要直接改宿主的表。
- **所有外部输入必须用 `?` 占位符**,只有表名可以拼接。
- 需要按应用隔离的数据,表里加 `app_id` 列,查询和更新都带上它。
- 文件存到 `host.DataDir()`(`运行目录/plugins/data/<key>/`)。

### 5.6 宿主业务

| 方法 | 说明 |
|---|---|
| `GetUser(uid)` | 账号信息(账号模式应用) |
| `GetAppUser(appId, uid)` | 软件用户:到期时间/点数、积分、绑定信息、用户类型等 |
| `AddBalance(uid, 金额, 备注)` | 增减余额,负数为扣减,不足时返回错误,自动写余额日志 |
| `AddVipNumber(appId, uid, 数量, 备注)` | 增减积分,自动写日志 |
| `AddVipTime(appId, uid, 数量, 备注)` | 增减到期时间(秒)或点数,自动写日志 |
| `WsSend(tokenId, 消息)` | 向 WebSocket 在线连接推送 |
| `AppEnabled(appId)` | 本插件是否对该应用启用 |
| `Log(格式, 参数...)` | 写宿主日志,自动带 `[插件:<key>]` 前缀 |

### 5.7 事件

```go
host.RegisterEvent(kyplugin.EventPaySuccess, func(ctx context.Context, e kyplugin.Event) {
	var 数据 kyplugin.PaySuccessPayload
	if e.Bind(&数据) != nil { return }
})
```

| 事件名 | 触发时机 | 数据 |
|---|---|---|
| `api.after:<Api名>` | 某个软件端接口处理完成,如 `api.after:UserLogin`、`api.after:NewUserInfo`、`api.after:UseKa` | `ApiAfterPayload`:接口名、令牌信息、请求明文、响应明文 |
| `api.after` | 任意软件端接口处理完成(每个请求都会触发,谨慎使用) | 同上 |
| `pay.success` | 支付订单处理成功 | `PaySuccessPayload`:订单号、用户、金额、通道、订单类型 |

- 事件是**异步通知**:在宿主流程结束后、独立协程里调用,不能拦截或修改宿主结果。
- 带 `AppId` 的事件只派发给对该应用启用的插件。
- 判断接口是否成功:解析 `Response` 里的 `Status`。
- 请求明文可能包含密码等敏感字段,不要记录到日志。

### 5.8 定时任务

```go
err := host.RegisterCron("每日统计", "0 0 4 * * *", func(ctx context.Context) { ... })
```

表达式为 6 位(秒 分 时 日 月 周)。插件停用后任务不执行。多节点部署时每个节点都会执行。

## 6. 依赖规则

- **优先只用标准库。** 这是避免"版本不一致"最可靠的办法。
- 可以引用宿主没有用到的第三方库。
- 引用宿主也在用的库(见宿主 `go.mod`,如 gin、gorm)时,版本必须与宿主完全一致,且该库自身的依赖也要一致。不建议这样做。
- 不要引用宿主的 `server/...` 包。
- 修改 `kyplugin` SDK 会让所有旧插件失效;改动导出接口时递增 `SDKVersion`。

## 7. 安全规范

1. 来自软件端的一切数据(`ctx.Data`、`Online.Key`、`Online.Tab`、`AppVer`)都不可信,使用前校验。
2. SQL 一律占位符传参。
3. 涉及余额、积分、时间的动作必须 `RequireLogin: true`,并使用 `ctx.Online.Uid`,不要相信请求里传来的用户 Id。
4. 处理函数会被并发调用,插件内的共享变量自己加锁。
5. 处理函数里的 panic 会被宿主捕获并记日志;**自己启动的协程里的 panic 会让整个宿主崩溃**,必须自己 `recover`。
6. 不要调用 `os.Exit`,不要修改全局状态(如 `http.DefaultClient`、时区、环境变量)。
7. 错误信息返回给软件端前要脱敏。

## 8. 常见加载错误

| 提示 | 原因与处理 |
|---|---|
| 插件与宿主的编译环境不一致 | Go 版本、构建参数、SDK 源码或共用的第三方库版本不同。用宿主那台机器上的 `build.sh` 重新编译 |
| 同一插件已在本进程载入过 | 之前载入过相同 module 名的文件(包括安装失败的那次)。重启宿主后再装;更新版本用「升级」 |
| 宿主未启用插件支持 | 宿主是用 `CGO_ENABLED=0` 编译的。用 `build.sh host` 重新编译 |
| 插件SDK版本与宿主不一致 | 用宿主当前的 `kyplugin` 重新编译插件 |
| 插件 Init 失败 | 看提示里的具体错误。注意:失败的那份代码已留在进程里,修正后要重启宿主才能再次安装 |

## 9. 发布检查清单

- [ ] `Key`、module 名唯一,`Version` 已更新
- [ ] 用 `build.sh` 编译,且宿主也是用它编译的
- [ ] `Init` 可重复执行
- [ ] 软件端动作:校验了输入,需要登录的设置了 `RequireLogin`
- [ ] SQL 全部使用占位符,按应用隔离的数据带了 `app_id` 条件
- [ ] 自启协程有 `recover`
- [ ] 在测试环境走过:安装 → 对应用启用 → 调用 → 停用 → 升级并重启
