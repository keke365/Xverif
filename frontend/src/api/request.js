import axios from 'axios' // 引入axios
import {ElMessage, ElMessageBox} from 'element-plus'
import {store} from '@/store/store.ts'
import router from '@/router/index'


// 后端重构后把字段名统一成了小驼峰(appId、list)，而本前端的页面仍按旧名字(Appid、List)读写。
// 请求方向不用处理：后端解析 JSON 时字段名不区分大小写。
// 返回方向在这里统一兼容：把返回数据里的每个对象包一层，按旧名字读写时，落到“忽略大小写后同名”的那个真实字段上。
// 真实字段名不变，所以把对象原样提交回后端时不会多出重复字段。
// 已知限制(写新代码时请直接用后端的真实字段名，就不会碰到)：
//   1. {...对象}、Object.assign、JSON 深拷贝得到的是普通对象，只带真实字段名，没有这层兼容；
//      把返回对象 Object.assign 到一个用旧名字做默认值的对象上，会同时出现新旧两套键。
//   2. structuredClone 不能克隆这种对象(会抛 DataCloneError)，需要时先 JSON 深拷贝。
//   3. 同一个对象不要混用两种写法：Vue 按"访问时用的名字"追踪依赖，用 A 写法读、用 B 写法改，界面不会刷新；
//      同理，watch(..., {deep: true}) 按真实字段名建立依赖，用旧名字修改不会触发它。
//   4. 查找不存在的键时也会忽略大小写：对象里没有 ABC 但有 abc 时，读 ABC 得到 abc 的值。
//   5. 后端同时返回 Status 和 status 这种"只差大小写"的两个键时，各读各的，不做合并(见 软件用户api.js 的处理)。
const 不参与兼容的键 = new Set(['then', 'toJSON', 'length', 'constructor'])

const 查找真实键 = (目标, 键) => {
    if (typeof 键 !== 'string' || 键 in 目标 || 键.startsWith('__') || 不参与兼容的键.has(键)) {
        return 键
    }
    const 小写键 = 键.toLowerCase()
    for (const 真实键 of Object.keys(目标)) {
        if (真实键.toLowerCase() === 小写键) {
            return 真实键
        }
    }
    return 键
}

const 兼容处理器 = {
    get: (目标, 键) => 目标[查找真实键(目标, 键)],
    set: (目标, 键, 值) => {
        目标[查找真实键(目标, 键)] = 值
        return true
    },
    has: (目标, 键) => 查找真实键(目标, 键) in 目标,
    // 让 hasOwnProperty / Object.hasOwn 与上面的 in 判断保持一致
    getOwnPropertyDescriptor: (目标, 键) => {
        const 真实键 = 查找真实键(目标, 键)
        // 被 Object.freeze 等锁定的对象不允许报告它本身没有的键，否则引擎会抛 TypeError
        if (真实键 !== 键 && !Object.isExtensible(目标)) {
            return undefined
        }
        return Reflect.getOwnPropertyDescriptor(目标, 真实键)
    },
    deleteProperty: (目标, 键) => delete 目标[查找真实键(目标, 键)],
}

// 数组本身不需要兼容(下标和方法不涉及字段名)，只处理它的元素
const 兼容旧字段名 = (值) => {
    if (Array.isArray(值)) {
        for (let i = 0; i < 值.length; i++) {
            值[i] = 兼容旧字段名(值[i])
        }
        return 值
    }
    if (值 !== null && typeof 值 === 'object') {
        for (const 键 of Object.keys(值)) {
            值[键] = 兼容旧字段名(值[键])
        }
        return new Proxy(值, 兼容处理器)
    }
    return 值
}

const service = axios.create({

    //baseURL: axios.defaults.baseURL,
    // 开发模式下页面由 vite 提供，接口在后端端口(.env.development 的 VITE_SERVER_PORT)；构建后页面与接口同源
    baseURL: import.meta.env.DEV
        ? window.location.protocol + "//" + window.location.hostname + ":" + import.meta.env.VITE_SERVER_PORT
        : window.location.origin,
    timeout: 99999
})

// http request 拦截请求 写入token
service.interceptors.request.use(
    config => {
        config.url+="?time="+new Date().getTime().toString();
        config.headers = {
            'Content-Type': 'application/json',
            'Token': localStorage.getItem("AdminToken"),
            ...config.headers
        }
        return config
    },
    error => {
        ElMessage.error(error)
        return error
    }
)


// http response 拦截响应
service.interceptors.response.use(
    response => {
        //console.info(response)
        // 只处理 JSON 接口的返回；下载等非 JSON 响应(Blob、文本)原样放行
        if (Object.prototype.toString.call(response.data) === '[object Object]') {
            response.data.data = 兼容旧字段名(response.data.data)
        }
        // 响应成功 返回data内容
        if (response.data.code === 10000) {
            return response.data
        } else {
            //判断失败原因

            // 101登录状态失效 清除token 本地数据 跳转登陆页
            if (response.data.code === 202 && response.data.data.KuaiYan === true) {

                console.info("快验个人中心未登录")

                router.push('/个人中心')
                if (response.data.msg!==""){
                    ElMessage.success(response.data.msg || decodeURI(response.headers.msg))
                }
            } else if (response.data.code === 202) {

                console.info("登录token被注销了")
                console.info(response)
                store.commit("setToken", "")
                localStorage.clear()
                router.replace('Login')

            } else {
                ElMessage.error(response.data.msg || decodeURI(response.headers.msg))
            }


            return response.data.msg ? response.data : response
        }
    },
    error => {
        // if (!error.config.donNotShowLoading) {
        //   closeLoading()
        // }

        if (!error.response) {
            ElMessageBox.confirm(`
        <p>检测到请求错误</p>
        <p>${error}</p>
        `, '请求报错', {
                dangerouslyUseHTMLString: true,
                distinguishCancelAndClose: true,
                confirmButtonText: '稍后重试',
                cancelButtonText: '取消'
            })
            return
        }

        switch (error.response.status) {
            case 500:
                ElMessageBox.confirm(`
        <p>检测到接口错误${error}</p>
        <p>错误码<span style="color:red"> 500 </span>：此类错误内容常见于后台panic，请先查看后台日志，如果影响您正常使用可强制登出清理缓存</p>
        `, '接口报错', {
                    dangerouslyUseHTMLString: true,
                    distinguishCancelAndClose: true,
                    confirmButtonText: '清理缓存',
                    cancelButtonText: '取消'
                })
                    .then(() => {
                        // localStorage.clear()
                        // router.push( 'Login')
                    })
                break
            case 404:
                ElMessageBox.confirm(`
          <p>检测到接口错误${error}</p>
          <p>错误码<span style="color:red"> 404 </span>：此类错误多为接口未注册（或未重启）或者请求路径（方法）与api路径（方法）不符--如果为自动化代码请检查是否存在空格</p>
          `, '接口报错', {
                    dangerouslyUseHTMLString: true,
                    distinguishCancelAndClose: true,
                    confirmButtonText: '我知道了',
                    cancelButtonText: '取消'
                })
                break
        }

        return error
    }
)
export default service
