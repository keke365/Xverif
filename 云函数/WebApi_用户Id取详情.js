function WebApi_用户Id取详情(JSON形参文本) {
    JSON形参文本 = JSON形参文本.replace(/'/g, '"')
    var 局_形参对象 = JSON.parse(JSON形参文本);
    $用户在线信息.Uid = 局_形参对象.Uid
    局_结果 = $api_用户Id取详情($用户在线信息)
    return 局_结果
}