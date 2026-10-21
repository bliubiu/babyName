#!/usr/bin/env bash
# =============================================================================
# 一键启动 namer 后端服务（NameMaster）
#
# 行为：
#   1. 若 backend/bin/namer-server.exe 不存在，或 backend 下源码/依赖/配置比
#      二进制更新，则自动重新构建（go build ./cmd/server）。
#   2. 前台启动服务，绑定 127.0.0.1:8080（Ctrl+C 停止，已做优雅关闭）。
#
# 用法：
#   ./start-server.sh            # 正常启动
#   ./start-server.sh --rebuild  # 强制重建后再启动
#   ./start-server.sh --check    # 仅检查/构建，不启动服务
#
# 说明：
#   - 构建产物 namer-server.exe 已被 .gitignore 忽略（*.exe），属本地产物，
#     脚本会按需重新生成，无需纳入版本库。
#   - 若 data/ 下的物料 JSON 有变动，请另行执行 `go run ./cmd/materials import`
#     重建数据库（backend/namer.db）；本脚本只负责二进制与服务的对齐。
# =============================================================================

set -euo pipefail

# 路径解析（脚本位于 backend/bin/）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$BACKEND_DIR/.." && pwd)"

BIN="$SCRIPT_DIR/namer-server.exe"
DB_PATH="$BACKEND_DIR/namer.db"
DATA_DIR="$ROOT_DIR/data"
HOST="127.0.0.1"
PORT="8080"
LOG_DIR="$SCRIPT_DIR/logs"

# 解析命令行参数
FORCE_REBUILD=0
CHECK_ONLY=0
for arg in "$@"; do
  case "$arg" in
    --rebuild) FORCE_REBUILD=1 ;;
    --check)   CHECK_ONLY=1 ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
  esac
done

# -----------------------------------------------------------------------------
# 1. 变更检测：是否需要重建
# -----------------------------------------------------------------------------
need_build=0
if [ "$FORCE_REBUILD" -eq 1 ]; then
  need_build=1
  echo "[build] 收到 --rebuild，强制重建"
elif [ ! -f "$BIN" ]; then
  need_build=1
  echo "[build] 二进制不存在，需要构建"
else
  BIN_MT="$(stat -c '%Y' "$BIN" 2>/dev/null || echo 0)"
  # 源码 / 依赖 / 配置的最新修改时间（用 xargs 批处理 stat，避免 -exec 派生过多进程）
  SRC_MT="$(find "$BACKEND_DIR" \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) -print0 2>/dev/null | xargs -0 -r stat -c '%Y' 2>/dev/null | sort -n | tail -1)"
  CFG_MT="$(find "$BACKEND_DIR" -name '*.yml' -print0 2>/dev/null | xargs -0 -r stat -c '%Y' 2>/dev/null | sort -n | tail -1)"
  NEWEST="$(printf '%s\n%s\n' "${SRC_MT:-0}" "${CFG_MT:-0}" | sort -n | tail -1)"
  if [ "${BIN_MT:-0}" -lt "${NEWEST:-0}" ]; then
    need_build=1
    echo "[build] 检测到源码/配置比二进制更新（$BIN_MT < $NEWEST），需要重建"
  else
    echo "[build] 二进制已是最新（跳过构建）"
  fi
fi

# -----------------------------------------------------------------------------
# 2. 构建
# -----------------------------------------------------------------------------
if [ "$need_build" -eq 1 ]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "错误: 未找到 go 命令，请先安装 Go 并将其加入 PATH" >&2
    exit 1
  fi
  echo "[build] 开始构建 namer-server ...（首次可能需数十秒）"
  if ! ( cd "$BACKEND_DIR" && go build -o "$BIN" ./cmd/server ); then
    echo "错误: 构建失败，请检查 Go 环境与源码" >&2
    exit 1
  fi
  echo "[build] 构建完成: $BIN"
fi

# --check 模式：仅构建，不启动
if [ "$CHECK_ONLY" -eq 1 ]; then
  echo "[check] 构建校验完成，未启动服务。"
  exit 0
fi

# -----------------------------------------------------------------------------
# 3. 启动服务（前台）
# -----------------------------------------------------------------------------
if [ ! -f "$DB_PATH" ]; then
  echo "警告: 未找到数据库 $DB_PATH，服务可能加载失败或回退内存模式" >&2
fi
mkdir -p "$LOG_DIR"

echo "[run] 启动服务 http://$HOST:$PORT （Ctrl+C 停止）"
export DATA_DIR="$DATA_DIR"
exec "$BIN" -host "$HOST" -port "$PORT" -db "$DB_PATH" -log "$LOG_DIR"
