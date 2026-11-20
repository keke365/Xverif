
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import  path from 'path'
import  fs from 'fs'
//element plus 使用的按需导入
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

// 引入 vite-plugin-cdn-import 插件
import { Plugin as importToCDN } from 'vite-plugin-cdn-import'


// 前端把页面地址的路径当作接口前缀(src/utils/utils.ts 的 取url根入口路径)，
// 所以开发服务器必须把页面挂在后端的管理入口下，挂在 / 下所有接口都会 404。
// 入口名取自后端的 config.json(后端首次启动时生成)，读不到时用后端的默认值 Admin。
// 只在 vite 启动时读取一次：改了后端的管理入口后要重启开发服务器。
function 读取管理入口(): string {
    try {
        const 后端配置 = JSON.parse(fs.readFileSync(path.resolve(__dirname, '../config.json'), 'utf8'))
        return String(后端配置['管理入口'] || '').replace(/^\/+|\/+$/g, '') || 'Admin'
    } catch {
        return 'Admin'
    }
}

// https://vitejs.dev/config/
export default defineConfig(({ command, mode }) => ({

    // 开发: /Admin/；构建: 相对路径，产物可挂在任意管理入口下
    base: command === 'serve' ? `/${读取管理入口()}/` : './',
    resolve:{
        alias:{
            '@': path.resolve(__dirname, 'src'),
            '@/*': path.resolve(__dirname, 'src/*'),
            'com': path.resolve(__dirname, 'src/components')  // tsconfig.json 里也必须配置路径别名
        }
    },
    build: {
        // 前端位于 后端/frontend，产物直接输出到后端 go:embed 的目录
        outDir: path.resolve(__dirname, '../app/web/VueAdmin'),
        emptyOutDir: false, // 目录里有 VueAdmin.go，不能清空；旧产物由 build.js 清理
        minify: "terser",
        terserOptions: {
            compress: {
                drop_console: true,
                drop_debugger: true
            }
        }
    },
    css: {
        //css预处理
        preprocessorOptions: {
            scss: {
/*                引入全局预定义变量 需要设置 lang="scss" */
                additionalData: `@import "./src/styles/variables.scss";`,
            }
        }
    },
    plugins: [
        vue(),
        AutoImport({
            resolvers: [ElementPlusResolver()],
        }),
        Components({
            resolvers: [ElementPlusResolver()],
        }),
        // 配置 CDN  打包时会自动替换
        importToCDN({
            modules: [
/*                {     //vue 正确应该引入 3.3.8  但是网站暂时没有,这个引入会失败,所以放弃
                    name: 'vue',
                    var: 'Vue',
                    path: `https://cdn.bootcdn.net/ajax/libs/vue/3.3.4/vue.runtime.global.prod.min.js`,
                },*/
                // {
                //     name: "echarts",
                //     var: "echarts",
                //     path: "https://cdn.bootcdn.net/ajax/libs/echarts/5.4.2/echarts.min.js", //预防投毒,避免被攻击改为本地化
                // },
/*                {
                    name: "element-plus", // 依赖名称,
                    var: "ElementPlus",  // 全局变量名
                    path: "https://cdn.bootcdn.net/ajax/libs/element-plus/2.3.1/index.full.min.js", // CDN 链接
                    css: "https://cdn.bootcdn.net/ajax/libs/element-plus/2.3.1/index.min.css", // 依赖有css就填，没有就去掉这个
                },*/
            ],
        }),
    ],
    server: {
        hmr: { overlay: false }, // 禁用或配置 HMR 连接 设置 server.hmr.overlay 为 false 可以禁用服务器错误遮罩层
        // 服务配置
        port: 18080, // 类型： number 指定服务器端口;
        open: true, // 类型： boolean | string在服务器启动时自动在浏览器中打开应用程序；
        cors: true, // 类型： boolean | CorsOptions 为开发服务器配置 CORS。默认启用并允许任何源
        //proxy look for https://vitejs.cn/config/#server-proxy
        proxy: {
            '/dev': {
                target: 'http://127.0.0.1:18888',
                changeOrigin: true,
                rewrite: path => path.replace(/^\/dev/, '')
            },
            // 插件自带的管理页面(plugin/static/...)、它引用的 sdk.js 和它调用的接口都转给后端，
            // 让插件页面与管理端同源：sdk.js 从 localStorage 读管理员令牌，直接打开后端端口上的页面读不到。
            // 后端端口与 src/api/request.js 用的是同一个 VITE_SERVER_PORT(make run 传入，否则取 .env.development)
            [`/${读取管理入口()}/plugin/`]: {
                target: `http://127.0.0.1:${loadEnv(mode, __dirname).VITE_SERVER_PORT || 18888}`,
            }
        }
    }
}))
