// 示例插件: 广告投放
// 演示了插件的全部能力: 自建表、按应用配置、软件端动作、管理端动作、管理端页面、事件、定时任务
package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"time"

	"kyplugin"
)

//go:embed static
var 集_静态文件 embed.FS

type 广告插件 struct {
	host kyplugin.Host
	表    string
}

// KyPlugin 插件入口,名称和签名固定
func KyPlugin() kyplugin.Plugin { return &广告插件{} }

func (j *广告插件) Info() kyplugin.Info {
	return kyplugin.Info{
		Key:         "ads",
		Name:        "广告投放",
		Version:     "1.0.0",
		Author:      "example",
		Description: "按应用投放广告,统计展示与点击",
		SDKVersion:  kyplugin.SDKVersion,
		Settings: []kyplugin.SettingField{
			{Key: "max", Label: "单次返回数量", Type: "number", Default: "5"},
			{Key: "vip_free", Label: "会员免广告", Type: "switch", Default: "0", Tip: "开启后,未到期的已登录用户不返回广告(仅计时应用有意义)"},
			{Key: "notice", Label: "广告位说明", Type: "text", Default: "", Tip: "随广告列表一起返回给软件端"},
		},
		Menus: []kyplugin.Menu{{Title: "广告管理", Path: "index.html"}},
	}
}

func (j *广告插件) Init(host kyplugin.Host) error {
	j.host = host
	j.表 = host.TableName("item") // plg_ads_item

	// 建表必须可重复执行: 每次宿主启动都会调用 Init
	_, 局_错误 := host.DB().Exec("CREATE TABLE IF NOT EXISTS `" + j.表 + "` (" +
		"`id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY," +
		"`app_id` INT NOT NULL DEFAULT 0 COMMENT '投放应用,0为全部已启用应用'," +
		"`title` VARCHAR(200) NOT NULL DEFAULT ''," +
		"`image` VARCHAR(500) NOT NULL DEFAULT ''," +
		"`url` VARCHAR(500) NOT NULL DEFAULT ''," +
		"`sort` INT NOT NULL DEFAULT 0," +
		"`status` TINYINT NOT NULL DEFAULT 1 COMMENT '1投放 2暂停'," +
		"`show_count` BIGINT NOT NULL DEFAULT 0," +
		"`click_count` BIGINT NOT NULL DEFAULT 0," +
		"`create_time` BIGINT NOT NULL DEFAULT 0," +
		"KEY `idx_app_status` (`app_id`,`status`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")
	if 局_错误 != nil {
		return 局_错误
	}

	host.RegisterClientAction("getAds", kyplugin.ClientOption{}, j.取广告)
	host.RegisterClientAction("click", kyplugin.ClientOption{}, j.点击上报)

	host.RegisterAdminAction("list", j.管理_列表)
	host.RegisterAdminAction("save", j.管理_保存)
	host.RegisterAdminAction("delete", j.管理_删除)

	host.RegisterEvent(kyplugin.EventApiAfterOf("UserLogin"), j.事件_登录后)

	局_子目录, 局_错误 := fs.Sub(集_静态文件, "static")
	if 局_错误 != nil {
		return 局_错误
	}
	host.RegisterStatic(局_子目录)

	return host.RegisterCron("每日统计", "0 0 4 * * *", j.定时_每日统计)
}

type 广告 struct {
	Id         int    `json:"id"`
	AppId      int    `json:"appId"`
	Title      string `json:"title"`
	Image      string `json:"image"`
	Url        string `json:"url"`
	Sort       int    `json:"sort"`
	Status     int    `json:"status"`
	ShowCount  int64  `json:"showCount"`
	ClickCount int64  `json:"clickCount"`
	CreateTime int64  `json:"createTime"`
}

// 取广告 软件端: {"Api":"PluginCall","Plugin":"ads","Action":"getAds"}
func (j *广告插件) 取广告(ctx *kyplugin.ClientContext) (any, error) {
	局_配置 := j.host.Setting(ctx.AppId)
	type 项 struct {
		Id    int    `json:"id"`
		Title string `json:"title"`
		Image string `json:"image"`
		Url   string `json:"url"`
	}
	局_列表 := []项{}

	if 局_配置.Bool("vip_free") && ctx.LoggedIn {
		if 局_用户, 局_错误 := j.host.GetAppUser(ctx.AppId, ctx.Online.Uid); 局_错误 == nil && 局_用户.VipTime > time.Now().Unix() {
			return map[string]any{"notice": "", "list": 局_列表}, nil
		}
	}

	局_数量 := 局_配置.Int("max")
	if 局_数量 <= 0 || 局_数量 > 50 {
		局_数量 = 5
	}
	// 始终用占位符传参,不要拼接来自请求的内容
	局_行, 局_错误 := j.host.DB().QueryContext(ctx.Ctx,
		"SELECT `id`,`title`,`image`,`url` FROM `"+j.表+"` WHERE `status`=1 AND `app_id` IN (0,?) ORDER BY `sort` DESC,`id` DESC LIMIT ?",
		ctx.AppId, 局_数量)
	if 局_错误 != nil {
		j.host.Log("查询广告失败: %v", 局_错误)
		return nil, errors.New("获取广告失败") // 不要把数据库错误原文返回给软件端
	}
	defer 局_行.Close()
	for 局_行.Next() {
		var 局_项 项
		if 局_错误 = 局_行.Scan(&局_项.Id, &局_项.Title, &局_项.Image, &局_项.Url); 局_错误 != nil {
			return nil, errors.New("获取广告失败")
		}
		局_列表 = append(局_列表, 局_项)
	}
	for _, 局_项 := range 局_列表 {
		_, _ = j.host.DB().ExecContext(ctx.Ctx, "UPDATE `"+j.表+"` SET `show_count`=`show_count`+1 WHERE `id`=?", 局_项.Id)
	}
	return map[string]any{"notice": 局_配置.Get("notice"), "list": 局_列表}, nil
}

// 点击上报 软件端: {"Api":"PluginCall","Plugin":"ads","Action":"click","Data":{"id":1}}
func (j *广告插件) 点击上报(ctx *kyplugin.ClientContext) (any, error) {
	var 局_请求 struct {
		Id int `json:"id"`
	}
	if 局_错误 := ctx.Bind(&局_请求); 局_错误 != nil {
		return nil, 局_错误
	}
	if 局_请求.Id <= 0 {
		return nil, errors.New("id错误")
	}
	// 带上 app_id 条件: 软件端只能上报投放给自己应用的广告
	_, 局_错误 := j.host.DB().ExecContext(ctx.Ctx,
		"UPDATE `"+j.表+"` SET `click_count`=`click_count`+1 WHERE `id`=? AND `app_id` IN (0,?)", 局_请求.Id, ctx.AppId)
	if 局_错误 != nil {
		return nil, errors.New("上报失败")
	}
	return nil, nil
}

func (j *广告插件) 管理_列表(ctx *kyplugin.AdminContext) (any, error) {
	局_行, 局_错误 := j.host.DB().QueryContext(ctx.Ctx,
		"SELECT `id`,`app_id`,`title`,`image`,`url`,`sort`,`status`,`show_count`,`click_count`,`create_time` FROM `"+j.表+"` ORDER BY `sort` DESC,`id` DESC LIMIT 1000")
	if 局_错误 != nil {
		return nil, 局_错误
	}
	defer 局_行.Close()
	局_列表 := []广告{}
	for 局_行.Next() {
		var 局_项 广告
		if 局_错误 = 局_行.Scan(&局_项.Id, &局_项.AppId, &局_项.Title, &局_项.Image, &局_项.Url, &局_项.Sort, &局_项.Status, &局_项.ShowCount, &局_项.ClickCount, &局_项.CreateTime); 局_错误 != nil {
			return nil, 局_错误
		}
		局_列表 = append(局_列表, 局_项)
	}
	return map[string]any{"list": 局_列表}, 局_行.Err()
}

func (j *广告插件) 管理_保存(ctx *kyplugin.AdminContext) (any, error) {
	var 局_请求 广告
	if 局_错误 := ctx.Bind(&局_请求); 局_错误 != nil {
		return nil, 局_错误
	}
	if 局_请求.Title == "" {
		return nil, errors.New("标题不能为空")
	}
	if 局_请求.Status != 2 {
		局_请求.Status = 1
	}
	var 局_错误 error
	if 局_请求.Id > 0 {
		_, 局_错误 = j.host.DB().ExecContext(ctx.Ctx,
			"UPDATE `"+j.表+"` SET `app_id`=?,`title`=?,`image`=?,`url`=?,`sort`=?,`status`=? WHERE `id`=?",
			局_请求.AppId, 局_请求.Title, 局_请求.Image, 局_请求.Url, 局_请求.Sort, 局_请求.Status, 局_请求.Id)
	} else {
		_, 局_错误 = j.host.DB().ExecContext(ctx.Ctx,
			"INSERT INTO `"+j.表+"` (`app_id`,`title`,`image`,`url`,`sort`,`status`,`create_time`) VALUES (?,?,?,?,?,?,?)",
			局_请求.AppId, 局_请求.Title, 局_请求.Image, 局_请求.Url, 局_请求.Sort, 局_请求.Status, time.Now().Unix())
	}
	if 局_错误 == nil {
		j.host.Log("管理员[%s]保存了广告[%s]", ctx.AdminUser, 局_请求.Title)
	}
	return nil, 局_错误
}

func (j *广告插件) 管理_删除(ctx *kyplugin.AdminContext) (any, error) {
	var 局_请求 struct {
		Id int `json:"id"`
	}
	if 局_错误 := ctx.Bind(&局_请求); 局_错误 != nil {
		return nil, 局_错误
	}
	_, 局_错误 := j.host.DB().ExecContext(ctx.Ctx, "DELETE FROM `"+j.表+"` WHERE `id`=?", 局_请求.Id)
	return nil, 局_错误
}

// 事件_登录后 事件是异步通知,不能改变登录结果
func (j *广告插件) 事件_登录后(ctx context.Context, e kyplugin.Event) {
	var 局_数据 kyplugin.ApiAfterPayload
	if e.Bind(&局_数据) != nil {
		return
	}
	j.host.Log("应用%d 有用户请求登录,令牌Id=%d", e.AppId, 局_数据.Online.TokenId)
}

func (j *广告插件) 定时_每日统计(ctx context.Context) {
	var 局_展示, 局_点击 int64
	_ = j.host.DB().QueryRowContext(ctx, "SELECT COALESCE(SUM(`show_count`),0),COALESCE(SUM(`click_count`),0) FROM `"+j.表+"`").Scan(&局_展示, &局_点击)
	j.host.Log("累计展示 %d 次,点击 %d 次", 局_展示, 局_点击)
}
