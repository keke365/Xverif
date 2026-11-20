import service from '@/api/request'

import {取url根入口路径} from "@/utils/utils";
const url=取url根入口路径() +"/appUser/"
// 分页获取AppUser信息列表
//{ "Page": 0,"Size": 10 }
// @Success 200 {string} json "{"success":true,"data":{},"msg":"获取成功"}"
export const GetAppUserList = async (data) => {
  const res = await service({
    url: url+'getList',
    method: 'post',
    data: data
  })
  // 后端这个接口的每一行同时带 "Status"(内嵌结构体的字段，恒为 0)和 "status"(真实状态 1正常 2冻结)。
  // 页面读的是 Status；状态开关拿到 0 会被组件自动纠正成"冻结"并触发保存，把用户真的冻结掉。
  // 所以这里把真实状态写回 Status。
  res?.data?.list?.forEach((row) => {
    row.Status = row.status
  })
  return res
}

//  Del批量删除AppUser
//{ "id": [ 5 ]}
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "注销成功"}"
export const Del批量删除AppUser = (data) => {
  return service({
    url: url+'delete',
    method: 'post',
    data: data
  })
}

//  GetAppUser详细信息
//{"Id": 1}
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "获取成功"}"
export const GetAppUser详细信息 = (data) => {
  return service({
    url:  url+'getInfo',
    method: 'post',
    data: data
  })
}

//  SaveAppUser详细信息
//...
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "获取成功"}"
export const  SaveAppUser信息 = (data) => {
  return service({
    url:  url+'saveInfo',
    method: 'post',
    data: data
  })
}

//  NewAppUser信息
//...
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "添加成功"}"
export const  NewAppUser信息 = (data) => {
  return service({
    url: url+'new',
    method: 'post',
    data: data
  })
}

//  SetStatus
//...
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "修改成"}"
export const  SetStatus = (data) => {
  return service({
    url: url+'setStatus',
    method: 'post',
    data: data
  })
}

//  Set批量维护_勾选用户增减时间点数
//...
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "修改成"}"
export const  Set批量维护增减时间点数 = (data) => {
  return service({
    url: url+'setBatchAddVipTime',
    method: 'post',
    data: data
  })
}

//  Set批量维护_勾选用户增减积分
//...
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "修改成"}"
export const  Set批量维护增减积分 = (data) => {
  return service({
    url: url+'setBatchAddVipNumber',
    method: 'post',
    data: data
  })
}
//  Set批量维护_勾选用户增减积分
//...
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "修改成"}"
export const  Set批量维护置用户云配置 = (data) => {
  return service({
    url: url+'setBatchSetUserConfig',
    method: 'post',
    data: data
  })
}
//  Set批量维护_勾选用户增减时间点数
//...
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "修改成"}"
export const  Set批量维护修改用户类型 = (data) => {
  return service({
    url: url+'setBatchUserClass',
    method: 'post',
    data: data
  })
}
//  Set批量维护_勾选用户增减时间点数
//...
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "修改成"}"
export const  Set批量维护修改用户备注 = (data) => {
  return service({
    url: url+'batchSetAppUserNote',
    method: 'post',
    data: data
  })
}

//  Set批量维护_全部用户增减时间点数
//...
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "修改成"}"
export const  Set批量维护_全部用户增减时间点数 = (data) => {
  return service({
    url: url+'setBatchAllUserVipTime',
    method: 'post',
    data: data
  })
}

//  Del批量维护_删除
export const Del批量维护_删除 = (data) => {
  return service({
    url: url+'deleteBatch',
    method: 'post',
    data: data
  })
}

//  Del批量维护_删除
export const Del批量维护_设置用户绑定信息 = (data) => {
  return service({
    url: url+'batchSetAppUserKey',
    method: 'post',
    data: data
  })
}

//  Del批量维护_删除
export const Del批量维护_导入软件用户 = (data) => {
  return service({
    url: url+'batchAddUser',
    method: 'post',
    data: data
  })
}
