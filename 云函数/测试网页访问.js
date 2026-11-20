function 测试网页访问(JSON形参文本) {
    局_url = "https://www.baidu.com/sugrec?&prod=pc_his&from=pc_web"
    返回对象 = $api_网页访问_POST(局_url, "api=123",协议头,"", 15, "")
    return 返回对象.Body
}