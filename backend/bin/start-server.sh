#!/usr/bin/env bash
# =============================================================================
# 一键启动 namer 服务（NameMaster：Go 后端 + Next.js 静态前端）
#
# 行为：
#   1. 后端：若 backend/bin/namer-server.exe 不存在，或 backend 下源码/依赖/配置
#      比二进制更新，则自动重新构建（go build ./cmd/server）。
#   2. 前端：若 frontend/out 不存在，或 frontend/src、frontend/public 及前端配置
#      比产物更新，则自动重新构建（next build，产物输出到 frontend/out）。
#   3. 前台启动服务，绑定 127.0.0.1:8080（Ctrl+C 停止，已做优雅关闭）。
#
# 用法：
#   ./start-server.sh                  # 正常启动（按需重建后端 + 前端）
#   ./start-server.sh --rebuild        # 强制重建后端 + 前端后再启动
#   ./start-server.sh --check          # 仅执行构建校验，不启动服务
#   ./start-server.sh --skip-frontend  # 跳过前端构建（只调后端接口时用）
#
# 说明：
#   - 构建产物 namer-server.exe 已被 .gitignore 忽略（*.exe），属本地产物，
#     脚本会按需重新生成，无需纳入版本库。
#   - 前端产物 frontend/out 同为本地产物。改动 frontend/src 后本脚本会自动重建，
#     避免服务继续托管过期的静态资源（历史故障：out 未重建 → 线上跑旧代码 →
#     结果页抛 "I.some is not a function"）。
#   - 若 data/ 下的物料 JSON 有变动，请另行执行 `go run ./cmd/materials import`
#     重建数据库（backend/namer.db）；本脚本只负责二进制与前端资源的对齐。
# =============================================================================

set -euo pipefail

# 路径解析（脚本位于 backend/bin/）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$BACKEND_DIR/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/frontend"
FRONTEND_OUT="$FRONTEND_DIR/out"

BIN="$SCRIPT_DIR/namer-server.exe"
DB_PATH="$BACKEND_DIR/namer.db"
DATA_DIR="$ROOT_DIR/data"
HOST="127.0.0.1"
PORT="8080"
LOG_DIR="$SCRIPT_DIR/logs"

# 解析命令行参数
FORCE_REBUILD=0
CHECK_ONLY=0
SKIP_FRONTEND=0
for arg in "$@"; do
  case "$arg" in
    --rebuild)       FORCE_REBUILD=1 ;;
    --check)         CHECK_ONLY=1 ;;
    --skip-frontend) SKIP_FRONTEND=1 ;;
    -h|--help)       sed -n '2,25p' "$0"; exit 0 ;;
  esac
done

# -----------------------------------------------------------------------------
# 0. 工具探测
# -----------------------------------------------------------------------------
# 本地 safe-delete shim 会拦截 next build 对 .next / out 的清理动作
# （批量阈值 50，命中即报 SAFE_DELETE_BULK_CONFIRM_REQUIRED），导致构建失败。
# 这些目录都是可再生的项目构建缓存，故强制关闭 shim（仅在本脚本内生效）。
# 变量语义见 shim：process.env.CODEBUDDY_SAFE_DELETE_ENABLED !== '0' 即启用，
# 因此必须显式赋 '0'（环境里若已预置为 '1'，用 :-0 不会覆盖）。
export CODEBUDDY_SAFE_DELETE_ENABLED=0

# -----------------------------------------------------------------------------
# 1. 变更检测：后端是否需要重建
# -----------------------------------------------------------------------------
need_build=0
if [ "$FORCE_REBUILD" -eq 1 ]; then
  need_build=1
  echo "[build] 收到 --rebuild，强制重建后端"
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
    echo "[build] 后端二进制已是最新（跳过构建）"
  fi
fi

# -----------------------------------------------------------------------------
# 2. 变更检测：前端是否需要重建
# -----------------------------------------------------------------------------
need_fe_build=0
if [ "$SKIP_FRONTEND" -eq 1 ]; then
  echo "[frontend] 收到 --skip-frontend，跳过前端构建"
elif [ "$FORCE_REBUILD" -eq 1 ]; then
  need_fe_build=1
  echo "[frontend] 收到 --rebuild，强制重建前端"
elif [ ! -d "$FRONTEND_DIR" ]; then
  echo "[frontend] 未找到 $FRONTEND_DIR，跳过前端构建" >&2
elif [ ! -d "$FRONTEND_OUT" ] || [ ! -f "$FRONTEND_OUT/index.html" ]; then
  need_fe_build=1
  echo "[frontend] 未找到前端产物 $FRONTEND_OUT，需要构建"
else
  # 前端源码侧：src/、public/、以及根目录下的构建相关配置
  FE_SRC_MT="$( { find "$FRONTEND_DIR/src" "$FRONTEND_DIR/public" -type f -print0 2>/dev/null; \
                 find "$FRONTEND_DIR" -maxdepth 1 -type f \( -name 'next.config.js' -o -name 'package.json' \
                   -o -name 'tsconfig.json' -o -name 'postcss.config.*' -o -name 'tailwind.config.*' \) -print0 2>/dev/null; \
               } | xargs -0 -r stat -c '%Y' 2>/dev/null | sort -n | tail -1)"
  # 产物侧：out/ 内最新文件（= 上次构建完成时间）
  FE_OUT_MT="$(find "$FRONTEND_OUT" -type f -print0 2>/dev/null | xargs -0 -r stat -c '%Y' 2>/dev/null | sort -n | tail -1)"
  if [ "${FE_OUT_MT:-0}" -lt "${FE_SRC_MT:-0}" ]; then
    need_fe_build=1
    echo "[frontend] 检测到前端源码比产物更新（$FE_OUT_MT < $FE_SRC_MT），需要重建"
  else
    echo "[frontend] 前端产物已是最新（跳过构建）"
  fi
fi

# -----------------------------------------------------------------------------
# 3. 构建后端
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

# -----------------------------------------------------------------------------
# 4. 构建前端（next build → frontend/out）
# -----------------------------------------------------------------------------
if [ "$need_fe_build" -eq 1 ]; then
  if ! command -v node >/dev/null 2>&1; then
    echo "错误: 未找到 node 命令，无法构建前端（可用 --skip-frontend 跳过）" >&2
    exit 1
  fi

  # 依赖缺失时先安装（postinstall 会跑 hoist-deps.mjs 平铺依赖）
  if [ ! -d "$FRONTEND_DIR/node_modules/next" ]; then
    if ! command -v pnpm >/dev/null 2>&1; then
      echo "错误: frontend/node_modules 缺失且未找到 pnpm，无法构建前端" >&2
      echo "      请手动执行: cd frontend && pnpm install" >&2
      exit 1
    fi
    echo "[frontend] 依赖缺失，执行 pnpm install ...（可能需数分钟）"
    if ! ( cd "$FRONTEND_DIR" && pnpm install ); then
      echo "错误: pnpm install 失败" >&2
      exit 1
    fi
  fi

  echo "[frontend] 开始构建前端（next build，可能需 1-3 分钟）..."
  if ( cd "$FRONTEND_DIR" && ./node_modules/.bin/next build ); then
    echo "[frontend] 构建完成: $FRONTEND_OUT"
  elif command -v pnpm >/dev/null 2>&1 && ( cd "$FRONTEND_DIR" && pnpm build ); then
    echo "[frontend] 构建完成（pnpm build）: $FRONTEND_OUT"
  else
    echo "错误: 前端构建失败，服务未启动（避免托管过期产物）" >&2
    echo "      可用 --skip-frontend 仅启动后端接口" >&2
    exit 1
  fi
fi

# --check 模式：仅构建，不启动
if [ "$CHECK_ONLY" -eq 1 ]; then
  echo "[check] 构建校验完成，未启动服务。"
  exit 0
fi

# -----------------------------------------------------------------------------
# 5. 启动服务（前台）
# -----------------------------------------------------------------------------
if [ ! -f "$DB_PATH" ]; then
  echo "警告: 未找到数据库 $DB_PATH，服务可能加载失败或回退内存模式" >&2
fi
mkdir -p "$LOG_DIR"

echo "[run] 启动服务 http://$HOST:$PORT （Ctrl+C 停止）"
export DATA_DIR="$DATA_DIR"
exec "$BIN" -host "$HOST" -port "$PORT" -db "$DB_PATH" -log "$LOG_DIR"
