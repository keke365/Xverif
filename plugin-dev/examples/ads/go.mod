// module 名必须全局唯一(建议 kyplugin_<插件Key>): 两个插件 module 名相同会被视为同一个插件
module kyplugin_ads

go 1.25.0

require kyplugin v0.0.0

// 指向宿主仓库里的 SDK 目录,必须和宿主编译时用的是同一份源码
replace kyplugin => ../../../kyplugin
