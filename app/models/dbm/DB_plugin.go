package dbm

// DB_plugin 已安装插件表(.so 原生插件)
// 列名不用 key: 它是 mysql 关键字
type DB_plugin struct {
	Key         string `json:"key" gorm:"column:plugin_key;primarykey;size:32;comment:插件唯一标识"`
	Name        string `json:"name" gorm:"column:name;size:100;comment:插件名称"`
	Version     string `json:"version" gorm:"column:version;size:32;comment:当前文件版本"`
	Author      string `json:"author" gorm:"column:author;size:100;comment:作者"`
	Description string `json:"description" gorm:"column:description;size:1000;comment:说明"`
	Status      int    `json:"status" gorm:"column:status;default:1;comment:1启用 2停用"`
	File        string `json:"file" gorm:"column:file;size:255;comment:so文件名,位于运行目录/plugins下"`
	Sha256      string `json:"sha256" gorm:"column:sha256;size:64;comment:so文件sha256"`
	LoadError   string `json:"loadError" gorm:"column:load_error;size:2000;comment:最近一次加载失败原因,成功为空"`
	CreateTime  int64  `json:"createTime" gorm:"column:create_time;comment:安装时间戳"`
	UpdateTime  int64  `json:"updateTime" gorm:"column:update_time;comment:更新时间戳"`
}

func (DB_plugin) TableName() string {
	return "db_plugin"
}

// DB_pluginApp 插件在各应用下的启用状态与配置
type DB_pluginApp struct {
	Key        string `json:"key" gorm:"column:plugin_key;primarykey;size:32;comment:插件唯一标识"`
	AppId      int    `json:"appId" gorm:"column:app_id;primarykey;autoIncrement:false;comment:应用AppId"`
	Status     int    `json:"status" gorm:"column:status;default:2;comment:1启用 2停用"`
	Settings   string `json:"settings" gorm:"column:settings;type:text;comment:配置json,键值均为字符串"`
	UpdateTime int64  `json:"updateTime" gorm:"column:update_time;comment:更新时间戳"`
}

func (DB_pluginApp) TableName() string {
	return "db_plugin_app"
}
