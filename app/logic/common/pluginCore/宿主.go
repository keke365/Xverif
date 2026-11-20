package pluginCore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	"kyplugin"

	"server/app/global"
	"server/app/logic/common/appUser"
	"server/app/logic/common/cron"
	"server/app/logic/common/log"
	"server/app/logic/common/user"
	"server/app/logic/webSocket"
	"server/app/service"
)

// 插件宿主 kyplugin.Host 的实现,每个插件一个
type 插件宿主 struct {
	key string
	实例  *插件实例
	可注册 atomic.Bool //仅 Init 期间为 true
}

func (j *插件宿主) 检查可注册(方法 string) {
	if !j.可注册.Load() {
		panic("kyplugin: " + 方法 + " 只能在 Init 内调用")
	}
}

func (j *插件宿主) RegisterClientAction(name string, opt kyplugin.ClientOption, fn kyplugin.ClientHandler) {
	j.检查可注册("RegisterClientAction")
	if name == "" || fn == nil {
		panic("kyplugin: RegisterClientAction 的 name 和 fn 不能为空")
	}
	j.实例.客户端动作[name] = 客户端动作{选项: opt, 处理: fn}
}

func (j *插件宿主) RegisterAdminAction(name string, fn kyplugin.AdminHandler) {
	j.检查可注册("RegisterAdminAction")
	if name == "" || fn == nil {
		panic("kyplugin: RegisterAdminAction 的 name 和 fn 不能为空")
	}
	j.实例.管理端动作[name] = fn
}

func (j *插件宿主) RegisterEvent(eventName string, fn kyplugin.EventHandler) {
	j.检查可注册("RegisterEvent")
	if eventName == "" || fn == nil {
		panic("kyplugin: RegisterEvent 的 event 和 fn 不能为空")
	}
	j.实例.事件[eventName] = append(j.实例.事件[eventName], fn)
}

func (j *插件宿主) RegisterCron(name, spec string, fn func(ctx context.Context)) error {
	j.检查可注册("RegisterCron")
	if name == "" || fn == nil {
		return errors.New("定时任务 name 和 fn 不能为空")
	}
	局_key, 局_实例 := j.key, j.实例
	return cron.Q全_定时任务.T添加本机任务("插件:"+局_key+":"+name, spec, func() {
		//插件被卸载/停用/Init失败后,任务仍留在调度器里,这里判断后空跑
		if L_插件.取可用实例(局_key) != 局_实例 {
			return
		}
		defer func() {
			if 局_异常 := recover(); 局_异常 != nil {
				global.GVA_LOG.Printf("[插件:%s] 定时任务[%s]发生异常: %v\n%s", 局_key, name, 局_异常, debug.Stack())
			}
		}()
		fn(context.Background())
	})
}

func (j *插件宿主) RegisterStatic(files fs.FS) {
	j.检查可注册("RegisterStatic")
	j.实例.静态 = files
}

func (j *插件宿主) DB() *sql.DB {
	局_连接池, 局_错误 := global.Get局db().DB()
	if 局_错误 != nil {
		return nil
	}
	return 局_连接池
}

func (j *插件宿主) TableName(name string) string {
	return "plg_" + j.key + "_" + name
}

func (j *插件宿主) Setting(appId int) kyplugin.Settings {
	return L_插件.合并配置(j.key, L_插件.取状态()[j.key].应用[appId].配置)
}

func (j *插件宿主) AppEnabled(appId int) bool {
	return L_插件.应用已启用(j.key, appId)
}

func (j *插件宿主) DataDir() string {
	局_目录 := filepath.Join(L_插件.Q取插件目录(), 常量_插件数据目录, j.key)
	_ = os.MkdirAll(局_目录, 0o755)
	return 局_目录
}

func (j *插件宿主) GetUser(uid int) (kyplugin.User, error) {
	局_用户, 局_存在 := service.NewUser(&gin.Context{}, global.Get局db()).Id取详情(uid)
	if !局_存在 || 局_用户.Id == 0 {
		return kyplugin.User{}, errors.New("用户不存在")
	}
	return kyplugin.User{
		Id: 局_用户.Id, User: 局_用户.User, Phone: 局_用户.Phone, Email: 局_用户.Email, Qq: 局_用户.Qq,
		Status: 局_用户.Status, Rmb: 局_用户.Rmb, RegisterTime: 局_用户.RegisterTime,
	}, nil
}

func (j *插件宿主) GetAppUser(appId, uid int) (kyplugin.AppUser, error) {
	if appId <= 10000 {
		return kyplugin.AppUser{}, errors.New("AppId错误")
	}
	局_用户, 局_存在 := service.NewAppUser(&gin.Context{}, global.Get局db(), appId).Uid取详情(appId, uid)
	if !局_存在 || 局_用户.Id == 0 {
		return kyplugin.AppUser{}, errors.New("软件用户不存在")
	}
	return kyplugin.AppUser{
		Id: 局_用户.Id, Uid: 局_用户.Uid, Status: 局_用户.Status, Key: 局_用户.Key, VipTime: 局_用户.VipTime,
		VipNumber: 局_用户.VipNumber, Note: 局_用户.Note, MaxOnline: 局_用户.MaxOnline,
		UserClassId: 局_用户.UserClassId, RegisterTime: 局_用户.RegisterTime, AgentUid: 局_用户.AgentUid,
	}, nil
}

func (j *插件宿主) AddBalance(uid int, amount float64, note string) (float64, error) {
	局_用户, 局_错误 := j.GetUser(uid)
	if 局_错误 != nil {
		return 0, 局_错误
	}
	局_新余额, 局_错误 := user.L_user.Id余额增减(&gin.Context{}, uid, math.Abs(amount), amount >= 0)
	if 局_错误 != nil {
		return 0, 局_错误
	}
	go log.L_log.Log_写余额日志(局_用户.User, "", j.日志备注(note)+"|新余额≈"+strconv.FormatFloat(局_新余额, 'f', 2, 64), amount)
	return 局_新余额, nil
}

func (j *插件宿主) AddVipNumber(appId, uid int, amount float64, note string) error {
	局_Id, 局_错误 := j.取软件用户Id(appId, uid)
	if 局_错误 != nil {
		return 局_错误
	}
	局_上下文 := &gin.Context{}
	if 局_错误 = appUser.L_appUser.Id积分增减(局_上下文, appId, 局_Id, math.Abs(amount), amount >= 0); 局_错误 != nil {
		return 局_错误
	}
	go log.L_log.Log_写积分点数时间日志(j.取登录名(appId, uid), "", j.日志备注(note), amount, appId, 1)
	return nil
}

func (j *插件宿主) AddVipTime(appId, uid int, amount int64, note string) error {
	局_Id, 局_错误 := j.取软件用户Id(appId, uid)
	if 局_错误 != nil {
		return 局_错误
	}
	局_绝对值 := amount
	if 局_绝对值 < 0 {
		局_绝对值 = -局_绝对值
	}
	局_上下文 := &gin.Context{}
	if 局_错误 = appUser.L_appUser.Id点数增减(局_上下文, appId, 局_Id, 局_绝对值, amount >= 0); 局_错误 != nil {
		return 局_错误
	}
	局_日志类型 := 3
	if service.NewAppInfo(局_上下文, global.Get局db()).App是否为计点(appId) {
		局_日志类型 = 2
	}
	go log.L_log.Log_写积分点数时间日志(j.取登录名(appId, uid), "", j.日志备注(note), float64(amount), appId, 局_日志类型)
	return nil
}

func (j *插件宿主) WsSend(tokenId int, message []byte) error {
	return webSocket.L_webSocket.SendMessage(tokenId, message)
}

func (j *插件宿主) Log(format string, args ...any) {
	global.GVA_LOG.Println("[插件:" + j.key + "] " + fmt.Sprintf(format, args...))
}

func (j *插件宿主) 日志备注(note string) string {
	return "插件[" + j.key + "]" + note
}

func (j *插件宿主) 取软件用户Id(appId, uid int) (int, error) {
	if appId <= 10000 || uid <= 0 {
		return 0, errors.New("AppId或Uid错误")
	}
	局_Id := service.NewAppUser(&gin.Context{}, global.Get局db(), appId).Uid取Id(appId, uid)
	if 局_Id == 0 {
		return 0, errors.New("软件用户不存在")
	}
	return 局_Id, nil
}

// 取登录名 写日志用: 卡号模式应用取卡号,账号模式取账号
func (j *插件宿主) 取登录名(appId, uid int) string {
	局_上下文 := &gin.Context{}
	db := global.Get局db()
	if service.NewAppInfo(局_上下文, db).App是否为卡号(appId) {
		局_卡号, _ := service.NewKa(局_上下文, db).Id取详情(uid)
		return 局_卡号.Name
	}
	局_用户, _ := service.NewUser(局_上下文, db).Id取详情(uid)
	return 局_用户.User
}
