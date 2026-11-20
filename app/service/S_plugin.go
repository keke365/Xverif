package service

import (
	"gorm.io/gorm"

	"server/app/models/dbm"
)

// S_plugin 插件表与插件应用配置表的数据操作
type S_plugin struct {
	db *gorm.DB
}

func NewPlugin(db *gorm.DB) *S_plugin {
	return &S_plugin{db: db}
}

func (j *S_plugin) List() (列表 []dbm.DB_plugin, err error) {
	err = j.db.Model(dbm.DB_plugin{}).Order("create_time").Find(&列表).Error
	return
}

func (j *S_plugin) Info(key string) (info dbm.DB_plugin, err error) {
	err = j.db.Model(dbm.DB_plugin{}).Where("plugin_key = ?", key).First(&info).Error
	return
}

func (j *S_plugin) Create(info *dbm.DB_plugin) error {
	return j.db.Model(dbm.DB_plugin{}).Create(info).Error
}

func (j *S_plugin) Update(key string, 数据 map[string]interface{}) error {
	return j.db.Model(dbm.DB_plugin{}).Where("plugin_key = ?", key).Updates(数据).Error
}

// Delete 删除插件记录及其全部应用配置
func (j *S_plugin) Delete(key string) error {
	if err := j.db.Where("plugin_key = ?", key).Delete(&dbm.DB_pluginApp{}).Error; err != nil {
		return err
	}
	return j.db.Where("plugin_key = ?", key).Delete(&dbm.DB_plugin{}).Error
}

// AppList 取应用配置, key 为空取全部插件的
func (j *S_plugin) AppList(key string) (列表 []dbm.DB_pluginApp, err error) {
	tx := j.db.Model(dbm.DB_pluginApp{})
	if key != "" {
		tx = tx.Where("plugin_key = ?", key)
	}
	err = tx.Find(&列表).Error
	return
}

// AppSave 新增或覆盖一条应用配置(联合主键 plugin_key+app_id)
func (j *S_plugin) AppSave(info *dbm.DB_pluginApp) error {
	return j.db.Save(info).Error
}
