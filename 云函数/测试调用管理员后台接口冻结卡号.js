function 测试调用管理员后台接口冻结卡号(参数) {
    局_url = "http://127.0.0.1:18888/Admin/AppUser/SetStatus"
    局_post = '{"AppId":10001,"Id":[69],"Status":2}'
    局_token = "WD3NMTTWNG40DERXA6WRZTK3BZZLTKMJ"
    协议头 = "Token: " + 局_token
    返回对象 = $api_网页访问_POST(局_url, 局_post,协议头, "", 15, "")
    return 返回对象
}