# Xverif 开发入口。依赖: Go(版本不低于 go.mod 里的要求，不会自动下载工具链)、gcc、Node.js、bash、GNU make、MySQL
#
#   make run    同时启动后端和管理端前端开发服务器(热更新)，Ctrl+C 一起停止
#               管理端: http://localhost:18080/Admin/   后端: http://127.0.0.1:18888
#   make build  构建前端并嵌入后端，产出单个可执行文件 ./server.bin
#
# 首次运行：先在 MySQL 里建好一个空库(初始化不会自动建库)，打开管理端后按页面提示填数据库信息，
# 初始化完成后的默认管理员账号和密码都是 admin；数据库配置保存在 ./config.json。
#
# 后端把可执行文件所在目录(即本目录)当作运行目录：config.json、plugins/ 等都生成在这里。
# 后端端口取自 config.json 的 port(默认 18888)，make run 会把它传给前端，两边不需要手工对齐。
# 注意：管理后台里的"重启系统"是让后端进程自己退出，在 make run 下等于前后端一起停止，需要重新 make run。
# 从脚本或 IDE 里停止 make run 时请发 SIGTERM；只给 make 进程单独发 SIGINT 不会停(终端里的 Ctrl+C 不受影响)。
#
# 已知问题(都不是 Makefile 的问题，写在这里是为了不踩坑)：
#   - 管理端前端是上游 2025-12 的版本，后端更新；两边的差异在 frontend/src/api/request.js 和各 api 文件里做了兼容，
#     写新页面时请直接用后端现在的字段名，细节见 request.js 开头的说明。
#   - 代理端(/Agent)和用户中心(/user/...)的前端源码不在本仓库，构建时用占位页代替(见下方 embed-stubs)。
#   - 后端启动时会扫描 /var/tmp/gopids(宝塔面板的 Go 项目目录)：里面恰好只有一个 .pid 文件时会把它改写成自己的 pid。
#   - 数据库初始化之前访问 /Agent/ 会在后端日志里打出一段 panic 堆栈(已被中间件接住，不影响使用)。
#   - 后端的"制卡统计""任务池分析"图表按 UTC 自然日分桶，东八区每天 8 点以后的数据会记到下一天。
#   - 公共函数"测试执行"的返回值会被后端转成字符串显示。
#   - 右上角的管理员用户名和未读消息角标显示不出来；用户账号列表的升序/降序与界面相反(均为上游前端原有问题)。
#   - go test ./app 里的分层约束测试有 4 条失败，上游代码原样如此。

SHELL       := /bin/bash
.SHELLFLAGS := -ec
.ONESHELL:
.NOTPARALLEL:

FRONTEND    := frontend
# 依赖安装完成的标记：比 package.json / package-lock.json 旧就重新 npm install(安装规则末尾会 touch 它)
NODE_STAMP  := $(FRONTEND)/node_modules/.package-lock.json
# 后端 go:embed 的管理端构建产物(vite 的 outDir 指向这里)，缺失时后端无法编译
ADMIN_INDEX := app/web/VueAdmin/index.html
# config.json 还不存在(后端从未启动过)时使用的后端端口，即后端自己的默认值
DEFAULT_BACKEND_PORT := 18888

.PHONY: run build backend frontend-build embed-stubs

run: backend $(NODE_STAMP)
	@listening() { (exec 3<>/dev/tcp/127.0.0.1/$$1) 2>/dev/null; }
	# 取 config.json 顶层的 port(键名不区分大小写，与后端一致)；文件不存在或没有这一项时用默认值
	port=$$(node -e 'const c = JSON.parse(require("fs").readFileSync("config.json", "utf8")); const k = Object.keys(c).find((k) => k.toLowerCase() === "port"); const p = parseInt(c[k], 10); if (p > 0) console.log(p)' 2>/dev/null) || true
	port=$${port:-$(DEFAULT_BACKEND_PORT)}
	if listening $$port; then
		echo "端口 $$port 已被占用(可能是上一次没退出的后端)，请先停掉占用它的进程" >&2
		exit 1
	fi

	# 任何方式退出都把两个服务一起停掉；Ctrl+C 是正常的停止方式，不让 make 打印报错行
	trap 'kill $$backend $$frontend 2>/dev/null; wait' EXIT
	trap 'exit 0' INT
	trap 'exit 143' TERM

	./server.bin & backend=$$!
	# 等后端就绪再启动前端：浏览器自动打开时接口要能连上，vite 也要读后端生成的 config.json(管理入口)
	for ((i = 0; i < 60; i++)); do
		listening $$port && break
		kill -0 $$backend 2>/dev/null || { echo "后端启动失败，见上方输出" >&2; exit 1; }
		sleep 0.5
	done
	listening $$port || echo "提示: 后端 30 秒内未监听 $$port 端口，先启动前端；若页面连不上后端，查看上方后端输出" >&2

	# 直接执行 vite(等同 npm run dev)，少一层进程，退出时才能可靠地停掉它。
	# --strictPort: 18080 被占用时直接报错，而不是悄悄换个端口(多半是上一次的 vite 没退出)
	# --clearScreen false: 不清屏，保留上面后端打印的启动信息
	# VITE_SERVER_PORT: 告诉前端后端在哪个端口(环境变量优先于 .env.development 里的同名值)
	(cd $(FRONTEND) && VITE_SERVER_PORT=$$port exec node_modules/.bin/vite --strictPort --clearScreen false) & frontend=$$!

	wait -n || true
	if kill -0 $$backend 2>/dev/null; then
		echo "前端开发服务器已退出，停止后端" >&2
	else
		echo "后端已退出，停止前端" >&2
	fi
	exit 1

build: frontend-build backend

# 宿主必须用 plugin-dev/build.sh 编译：构建参数要与 .so 插件一致，否则插件无法加载
backend: embed-stubs $(ADMIN_INDEX)
	bash plugin-dev/build.sh host

# 产物缺失时也走这条规则，所以首次 make run 会先构建一次前端
frontend-build $(ADMIN_INDEX): | $(NODE_STAMP)
	npm --prefix $(FRONTEND) run build

$(NODE_STAMP): $(FRONTEND)/package.json $(FRONTEND)/package-lock.json
	npm --prefix $(FRONTEND) install
	touch $@

# 代理端(VueAgent)和用户中心(webUser)的前端不在本仓库，上游也不提交它们的构建产物。
# 缺失时生成占位文件，否则 go:embed 找不到文件、后端无法编译；已有真实产物时不会覆盖。
embed-stubs:
	@for dir in app/web/VueAgent/assets app/web/webUser/assets app/web/webUser/static; do
		mkdir -p $$dir
		[ -n "$$(ls -A $$dir)" ] || echo '占位文件，由 Makefile 生成' > $$dir/placeholder.txt
	done
	for page in app/web/VueAgent/index.html app/web/webUser/index.html; do
		[ -e $$page ] || echo '<!doctype html><meta charset="utf-8"><p>该前端未构建（占位页，由 Makefile 生成）。</p>' > $$page
	done
