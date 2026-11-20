function 置用户云配置(JSON形参文本) {
    let 配置名 = "窗口宽度";
    let 配置值 = "360px";
    $用户在线信息.Uid = 57
    var 局_结果对象 = $api_置用户云配置($用户在线信息, 配置名, 配置值)
    if (局_结果对象.IsOk) {
        return "写入成功"
    }
    return 局_结果对象.Err
}