function 执行SQL功能测试(JSON形参文本) {
	return 0
    var 局_结果对象 = $api_执行SQL功能("UPDATE db_public_js SET Type=Type+1 WHERE  Id=11")
    if (局_结果对象.isOk) {
        let 影响行数 = Number(局_结果对象.Err)
        return 影响行数
    }
    return 局_结果对象.Err
}