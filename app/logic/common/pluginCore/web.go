package pluginCore

import _ "embed"

// 插件管理页与前端辅助脚本。放在本包而不是 app/web: 那里的非 .go 文件被 .gitignore 排除

//go:embed web/index.html
var W管理页_首页 []byte

//go:embed web/sdk.js
var W管理页_脚本 []byte
