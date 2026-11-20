#!/usr/bin/env bash
# 宿主与插件的统一构建脚本。两者必须用同一个 Go、同样的参数编译,否则插件无法加载。
#
#   plugin-dev/build.sh host                 编译宿主        -> ./server.bin
#   plugin-dev/build.sh plugin <插件目录>     编译插件        -> <插件目录>/<目录名>.so
set -euo pipefail

# 变量名只能用英文: bash 不支持中文变量名
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root_dir="$(cd "$script_dir/.." && pwd)"
# 这三项宿主与插件必须完全一致,不要单独给某一方加 -race / -gcflags / -tags
export CGO_ENABLED=1
export GOFLAGS="-trimpath"
export GOTOOLCHAIN=local

command -v gcc >/dev/null || { echo "缺少 gcc: 插件机制依赖 cgo" >&2; exit 1; }
echo "Go: $(go version)"

case "${1:-}" in
host)
  cd "$root_dir"
  go build -o server.bin .
  echo "宿主已生成: $root_dir/server.bin"
  ;;
plugin)
  plugin_dir="$(cd "${2:?用法: build.sh plugin <插件目录>}" && pwd)"
  output="$plugin_dir/$(basename "$plugin_dir").so"
  cd "$plugin_dir"
  go build -buildmode=plugin -o "$output" .
  echo "插件已生成: $output"
  ;;
*)
  sed -n '2,5p' "${BASH_SOURCE[0]}"; exit 1 ;;
esac
