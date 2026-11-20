import service from '@/api/request'

import {取url根入口路径} from "@/utils/utils";
const url=取url根入口路径() +"/plugin/"

// GetList 已安装插件列表
// 返回 data: { sdkVersion, list: [{ key, name, version, author, description, status(1启用 2停用), sha256, loadError,
//   createTime, updateTime, loaded, loadedVersion, pendingUpgrade,
//   settings: [{ Key, Label, Type, Default, Options, Tip }], menus: [{ Title, Path }] }] }
// settings / menus 里的字段名是大写开头(后端 kyplugin SDK 的结构体没有 json 标签)
export const GetList = () => {
  return service({
    url: url+'getList',
    method: 'post',
    data: {}
  })
}

// Upload 上传 .so 文件。key 为空是安装新插件,否则是升级已安装的插件(重启系统后生效)
// onProgress 可选,上传过程中多次回调 ({ loaded, total })
export const Upload = (file, key, onProgress) => {
  const 表单 = new FormData()
  表单.append('file', file)
  if (key) {
    表单.append('key', key)
  }
  return service({
    url: url+'upload',
    method: 'post',
    data: 表单,
    // 覆盖 request.js 默认的 application/json,否则 axios 会把表单转成 JSON; 浏览器会自动补上 boundary
    headers: {'Content-Type': 'multipart/form-data'},
    timeout: 0, // 插件文件最大 200MB,不限制上传时间
    onUploadProgress: onProgress
  })
}

// SetStatus 全局启用/停用  { key, status: 1启用 2停用 }
export const SetStatus = (data) => {
  return service({
    url: url+'setStatus',
    method: 'post',
    data: data
  })
}

// Uninstall 卸载  { key }
export const Uninstall = (data) => {
  return service({
    url: url+'uninstall',
    method: 'post',
    data: data
  })
}

// GetAppList 插件在各应用下的启用状态与配置  { key }
// 返回 data: { list: [{ appId, appName, status(1启用 2停用), settings: { 配置键: 字符串值 } }] }
export const GetAppList = (data) => {
  return service({
    url: url+'getAppList',
    method: 'post',
    data: data
  })
}

// SaveApp 保存插件在某应用下的启用状态与配置  { key, appId, status, settings: { 配置键: 字符串值 } }
export const SaveApp = (data) => {
  return service({
    url: url+'saveApp',
    method: 'post',
    data: data
  })
}

// 插件自带管理页面的地址。必须与管理端同源: 页面里的 sdk.js 从 localStorage 读管理员令牌
// (开发模式下由 vite.config.ts 里的代理把这个路径转给后端)
export const 取插件页面地址 = (key, path) => {
  return window.location.origin + url + 'static/' + key + '/' + String(path || '').replace(/^\/+/, '')
}
