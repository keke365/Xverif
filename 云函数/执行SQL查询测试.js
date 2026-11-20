function 执行SQL查询测试(JSON形参文本) {
	return {}
    var 局_结果对象 = $api_执行SQL查询(" SELECT * FROM db_public_js")
    if (局_结果对象.isOk) {
        return 局_结果对象.Data
    }
    return 局_结果对象.Data
}