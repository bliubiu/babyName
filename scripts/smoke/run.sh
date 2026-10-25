#!/usr/bin/env bash
# =============================================================================
# scripts/smoke/run.sh — SSR 全页面冒烟套件运行器
#
# 按序运行 4 个冒烟脚本（对服务发 SSR 表单/页面请求，UTF-8 urllib，规避 GBK 坑）：
#   smoke_e2e.py     首页/生成/结果/换一批/收藏/历史/测名/黄历/统计 + 错误路径（45 项）
#   smoke_ssr.py     服务层 FullName 回填回归（候选名带姓、渲染不失败）
#   smoke_ssr_ext.py 筛选/对比/JSON查看器/暗色/报告/历史回看/统计查询/黄历万年历/Flash（77 项）
#   smoke_delete.py  取消收藏 / 删除历史链路
#
# 服务来源（优先级）：
#   1) 环境变量 SMOKE_BASE / SERVER_URL 指定 → 复用，不启停（供 verify-quality.sh 调用）
#   2) 127.0.0.1:8080 已就绪 → 复用
#   3) 否则按需构建并临时拉起服务，跑完经 trap 自动关闭
#   SKIP_SERVER=1 且无可用地址 → 直接报错退出（不静默自启，避免 CI 假过）
#
# 任一脚本退出码非 0（有 FAIL/异常）→ 本脚本 exit 1，便于接入 CI。
#
# 用法：
#   bash scripts/smoke/run.sh                                 # 自动管理服务
#   SMOKE_BASE=http://127.0.0.1:8080 bash scripts/smoke/run.sh   # 复用已起服务
# =============================================================================
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
BACKEND="$ROOT/backend"
BIN="$BACKEND/bin/namer-server.exe"
DB="$BACKEND/namer.db"
DATA="$ROOT/data"
LOG_DIR="$BACKEND/bin/logs"
HOST="127.0.0.1"
PORT="8080"

# python 探测：逐个候选实跑 `import sys`，选第一个真正可用的解释器。
# 兼顾 CI(ubuntu 的 python3) 与 Windows（python3 常为 Microsoft Store 占位符，
# 实跑即以退出码 49 失败，故必须实跑验证而非仅 command -v）。
pick_python() {
  local cand
  for cand in "${PYTHON:-}" python3 python py; do
    [ -z "$cand" ] && continue
    if command -v "$cand" >/dev/null 2>&1 && "$cand" -c 'import sys' >/dev/null 2>&1; then
      printf '%s\n' "$cand"
      return 0
    fi
  done
  return 1
}
if [ -z "${PYTHON:-}" ]; then
  if ! PYTHON="$(pick_python)"; then
    echo "[smoke] 未找到可用的 python 解释器（尝试过 python3/python/py）" >&2
    exit 1
  fi
fi

SCRIPTS=(smoke_e2e.py smoke_ssr.py smoke_ssr_ext.py smoke_delete.py)

BASE_URL="${SMOKE_BASE:-${SERVER_URL:-}}"
REUSE=0
SERVER_PID=""

cleanup() {
  if [ -n "$SERVER_PID" ]; then
    echo "[smoke] 关闭临时服务 (pid=$SERVER_PID)…"
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

probe() {
  local base="$1" code
  code=$(curl --noproxy '*' -s -o /dev/null -w '%{http_code}' --max-time 2 "$base/readyz" 2>/dev/null)
  [ "$code" = "200" ] && return 0
  code=$(curl --noproxy '*' -s -o /dev/null -w '%{http_code}' --max-time 2 "$base/healthz" 2>/dev/null)
  [ "$code" = "200" ] && return 0
  return 1
}

# ---- 确定服务来源 ----
if [ -n "$BASE_URL" ]; then
  REUSE=1
elif [ "${SKIP_SERVER:-0}" = "1" ]; then
  if probe "http://$HOST:$PORT"; then
    BASE_URL="http://$HOST:$PORT"; REUSE=1
  else
    echo "[smoke] SKIP_SERVER=1 但 $HOST:$PORT 未就绪，且未提供 SMOKE_BASE/SERVER_URL" >&2
    exit 1
  fi
elif probe "http://$HOST:$PORT"; then
  BASE_URL="http://$HOST:$PORT"; REUSE=1
else
  # 按需构建（namer-server.exe 这一文件名在 Windows/Linux 下均由 go build -o 原样产出）
  if [ ! -f "$BIN" ]; then
    echo "[smoke] 构建后端 → $BIN …"
    if ! go -C "$BACKEND" build -o "$BIN" ./cmd/server; then
      echo "[smoke] 后端构建失败" >&2; exit 1
    fi
  fi
  mkdir -p "$LOG_DIR"
  echo "[smoke] 启动临时服务 → http://$HOST:$PORT （日志: $LOG_DIR/smoke-server.log）"
  DATA_DIR="$DATA" "$BIN" -host "$HOST" -port "$PORT" -db "$DB" -log "$LOG_DIR" \
    >"$LOG_DIR/smoke-server.log" 2>&1 &
  SERVER_PID=$!
  BASE_URL="http://$HOST:$PORT"
  ready=0
  for _ in $(seq 1 60); do
    probe "$BASE_URL" && { ready=1; break; }
    kill -0 "$SERVER_PID" 2>/dev/null || break
    sleep 1
  done
  if [ "$ready" -ne 1 ]; then
    echo "[smoke] 服务 60s 未就绪（详见 $LOG_DIR/smoke-server.log）" >&2
    exit 1
  fi
fi

echo "=================================================================="
echo "[smoke] 目标: $BASE_URL  python=$PYTHON  reuse=$REUSE"
echo "=================================================================="

FAILS=0
for s in "${SCRIPTS[@]}"; do
  echo ""
  echo "---- $s ----"
  SMOKE_BASE="$BASE_URL" "$PYTHON" "$HERE/$s"
  rc=$?
  if [ "$rc" -ne 0 ]; then
    echo "[smoke] ✗ $s 未通过（退出码 $rc）"
    FAILS=$((FAILS+1))
  fi
done

echo ""
echo "=================================================================="
if [ "$FAILS" -eq 0 ]; then
  echo "[smoke] ✅ 冒烟套件全部通过（${#SCRIPTS[@]} 个脚本）"
  exit 0
else
  echo "[smoke] ❌ $FAILS / ${#SCRIPTS[@]} 个脚本未通过"
  exit 1
fi
