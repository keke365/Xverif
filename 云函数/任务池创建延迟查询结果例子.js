function 任务池创建查询例子(形参) {
    let 任务类型ID = 1
    let 结果 = $api_任务池_任务创建($用户在线信息, 任务类型ID, JSON形参文本)
    if (结果.IsOk) {
        let 局_任务对象 = 结果.Data
        let 任务结果
        for (let i = 0; i < 3; i++) {
            $程序_延时(5000);
            任务结果 = $api_任务池_任务查询(局_任务对象.TaskUuid)
            if (任务结果.Data.Status !== 1 && 任务结果.Data.Status !== 2) {
                break
            }
        }
        if (任务结果.Data.Status === 3) {
            return { Code: 1, Msg: "ok", recognition: 任务结果.Data.ReturnData }
        }
    }
    return { Code: -1, Msg: "失败" }
}