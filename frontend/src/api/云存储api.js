import service from '@/api/request'
import {取url根入口路径} from "@/utils/utils";
const url=取url根入口路径() +"/cloudStorage/"

//{ "Page": 0,"Size": 10 }
// @Success 200 {string} json "{"success":true,"data":{},"msg":"获取成功"}"
export const GetCloudStorageList = async (data) => {
  // 旧后端一次返回整个目录；新后端改成了分页(每页最多 100 条)，而页面按"拿到的是全部"来用。
  // 这里把各页取完再合并，返回形状与旧版一致(List 为全部条目，Count 为总数)。
  const 每页数量 = 100
  let res = await service({url: url+'getList', method: 'post', data: {...data, Size: 每页数量, Page: 1, Order: 1}})
  if (res.code !== 10000 || !Array.isArray(res.data?.List)) {
    return res
  }
  const 全部 = res.data.List
  for (let 页码 = 2; 全部.length < res.data.Count; 页码++) {
    const 下一页 = await service({url: url+'getList', method: 'post', data: {...data, Size: 每页数量, Page: 页码, Order: 1}})
    if (下一页.code !== 10000 || !下一页.data?.List?.length) {
      break
    }
    全部.push(...下一页.data.List)
  }
  res.data.Count = 全部.length
  return res
}


//获取上传凭证
export const GetUpToken = (data) => {
  return service({
    url: url+'getUpToken',
    method: 'post',
    data: data
  })
}



//  Del批量删除
//{ "path": [ "" ]}
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "注销成功"}"
export const Del批量删除 = (data) => {
  return service({
    url: url+'delete',
    method: 'post',
    data: data
  })
}


//  文件移动
//{ "path": [ "" ]}
// @Success 200 {string} json "{"code": 0, "data": {},"msg": "注销成功"}"
export const 文件移动 = (data) => {
  return service({
    url: url+'moveTo',
    method: 'post',
    data: data
  })
}
export const 下载 = (data) => {
  return service({
    url: url+'download',
    method: 'post',
    data: data
  })
}

export const 获取外链 = (data) => {
  return service({
    url: url+'getDownloadUrl',
    method: 'post',
    data: data
  })
}

export const 取基础信息 = (data) => {
  return service({
    url: url+'getBaseInfo',
    method: 'post',
    data: data
  })
}
