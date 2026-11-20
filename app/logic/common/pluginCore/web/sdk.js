// 飞鸟快验 插件管理端页面辅助脚本
// 插件页面引入: <script src="../../sdk.js"></script>   (页面地址为 /{管理入口}/plugin/static/<key>/xxx.html)
// 之后使用 kyplugin.call('动作名', {参数}) 调用本插件用 RegisterAdminAction 注册的动作,返回 Promise<返回值>
(function () {
  var 匹配 = location.pathname.match(/^(.*\/plugin)\/(?:static\/([^\/]+)\/|ui)/);
  var 根路径 = 匹配 ? 匹配[1] : '';
  var 插件key = (匹配 && 匹配[2]) || '';

  function 请求(地址, 选项) {
    选项.method = 'POST';
    选项.headers = Object.assign({ Token: localStorage.getItem('AdminToken') || '' }, 选项.headers || {});
    return fetch(地址, 选项).then(function (响应) {
      if (!响应.ok) throw new Error('请求失败: HTTP ' + 响应.status);
      return 响应.json();
    }).then(function (结果) {
      if (结果.code === 10000) return 结果;
      var 错误 = new Error(结果.code === 202 ? '登录已失效,请先登录管理后台' : (结果.msg || '请求失败'));
      错误.code = 结果.code;
      throw 错误;
    });
  }

  window.kyplugin = {
    key: 插件key,
    // 调用本插件的管理端动作,返回动作的返回值
    call: function (动作, 数据) {
      return 请求(根路径 + '/call/' + 插件key + '/' + 动作, {
        headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(数据 || {})
      }).then(function (结果) { return 结果.data; });
    },
    // 调用宿主的插件管理接口(getList / getAppList 等),返回完整响应 {code,data,msg}
    host: function (接口, 数据) {
      return 请求(根路径 + '/' + 接口, {
        headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(数据 || {})
      });
    },
    // 上传表单(FormData),返回完整响应
    upload: function (表单) {
      return 请求(根路径 + '/upload', { body: 表单 });
    }
  };
})();
