import service from '@/api/request'

import {取url根入口路径} from "@/utils/utils";
const url=取url根入口路径() +"/publicJs/"
// GetList
//{ "Page": 0,"Size": 10 }
// @Success 200 {string} json "{"success":true,"data":{},"msg":"获取成功"}"
export const GetList = (data) => {
  return service({
    url: url+'getList',
    method: 'post',
    // 新后端按 AppId 过滤，而页面的搜索条件里写死了 AppId: 1 且没有选择控件；
    // 去掉它才能像旧版一样列出全部函数(任务池 Hook、ApiHook 等不属于 AppId 1)
    data: {...data, AppId: undefined}
  })
}

//  Delete
// {
//   "data": [
//   {
//     "AppId": 1,
//     "Name": "系统名称",
//   },
//   {
//     "AppId": 1,
//     "Name": "系统类型",
//   }
// ]
// }
export const DeleteInfo= (data) => {
  return service({
    url: url+'delete',
    method: 'post',
    data: data
  })
}

//  GetInfo
//*{
// 	"AppId": 1,
// 	"Name": "系统名称"
// }*/
export const GetInfo = (data) => {
  return service({
    url:  url+'getInfo',
    method: 'post',
    data: data
  })
}

//  SaveInfo
/*{
	"AppId": 1,
	"Name": "系统名称",
	"Value":"飞鸟快验应用管理系统",
}*/
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "获取成功"}"
export const  SaveInfo = (data) => {
  return service({
    url:  url+'saveInfo',
    method: 'post',
    data: data
  })
}

//  New
/*{
	"AppId": 1,
	"Name": "系统名称",
	"Value":"飞鸟快验应用管理系统",
	"KeyType":11
}*/
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "添加成功"}"
export const  New = (data) => {
  return service({
    url: url+'new',
    method: 'post',
    data: data
  })
}

//测试执行js {"Id":1,"Param":"{}"}
export const  TestRunJs = (data) => {
  return service({
    url: url+'testRunJs',
    method: 'post',
    data: data
  })
}
