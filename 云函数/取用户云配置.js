function 取用户云配置(JSON形参文本) {
    let 配置名 = "窗口宽度";
    $用户在线信息.Uid = 57
    var 局_结果对象 = $api_取用户云配置($用户在线信息, 配置名)
    if (局_结果对象.IsOk) {
        return 局_结果对象.Data
    }
    return 局_结果对象.Err
}