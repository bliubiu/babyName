#!/usr/bin/env bash
# =============================================================================
# verify-quality.sh — 起名链路质量与解析说明「一键验证」
#
# 覆盖四道防线（详见 docs/22、docs/28、docs/33）：
#   ① 单元层（不需服务）：fate 评分正确性 / 评分快照防漂移 / 八维 score_detail 透传
#   ② 门禁审计（不需服务）：quality gate 覆盖分析
#   ②+ 专项维度巡检（不需服务）：dim_audit 四维度已知答案+不变量
#      （八字/五行/生肖/诗词出典准确性，直接吃真实 domain 逻辑）
#   ③ 端到端（需服务）：e2e_check 八大段 18 项（耗时/用字质量/参数生效/
#                        双路径同分/可复现/并发/测名/探索）
#   ④ 解释准确性：喜用神双算法对拍、诗词来源筛选、PDF 真实数据 等已内含于上述测试
#
# 退出码：任一阶段失败即以 1 退出，便于接入 CI。
#
# 用法：
#   ./scripts/verify-quality.sh                 # 全量（自动构建并拉起服务）
#   SERVER_URL=http://127.0.0.1:8080 ./scripts/verify-quality.sh   # 复用已起服务，跳过构建/启停
#   SKIP_SERVER=1 ./scripts/verify-quality.sh   # 只跑不需要服务的阶段（单元+门禁）
#
# 可选环境变量（覆盖 e2e 巡检参数）：
#   E2E_REPEAT / E2E_CONCURRENCY / E2E_ROUNDS
# =============================================================================

set -uo pipefail

# ---------------------------------------------------------------------------
# 路径解析
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
BIN="$BACKEND_DIR/bin/namer-server.exe"
DB_PATH="$BACKEND_DIR/namer.db"
DATA_DIR="$ROOT_DIR/data"
LOG_DIR="$BACKEND_DIR/bin/logs"
HOST="127.0.0.1"
PORT="8080"
URL="http://$HOST:$PORT"
SERVER_LOG="$LOG_DIR/verify-server.log"

# e2e 巡检参数（可被环境变量覆盖）
E2E_REPEAT="${E2E_REPEAT:-6}"
E2E_CONCURRENCY="${E2E_CONCURRENCY:-16}"
E2E_ROUNDS="${E2E_ROUNDS:-2}"

FAILURES=0
SERVER_PID=""
REUSE_SERVER=0

# ---------------------------------------------------------------------------
# 工具函数
# ---------------------------------------------------------------------------
log()  { printf '\033[1;34m[verify]\033[0m %s\n' "$*"; }
pass() { printf '\033[1;32m[PASS]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[FAIL]\033[0m %s\n' "$*"; FAILURES=$((FAILURES+1)); }

# 运行一个命令并把结果计入 FAILURES；失败时打印其退出码
run_phase() {
  local name="$1"; shift
  log "▶ $name"
  if "$@"; then
    pass "$name"
  else
    fail "$name (退出码 $?)"
  fi
}

# 关闭后台服务（无论成功失败都执行）
cleanup() {
  if [ -n "$SERVER_PID" ] && [ "$REUSE_SERVER" -eq 0 ]; then
    log "关闭验证服务 (pid=$SERVER_PID)…"
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# 探测就绪：优先 /readyz（校验数据已加载），回退 /healthz
probe_ready() {
  for ep in /readyz /healthz; do
    code=$(curl --noproxy '*' -s -o /dev/null -w '%{http_code}' --max-time 2 "$URL$ep" 2>/dev/null)
    if [ "$code" = "200" ]; then
      return 0
    fi
  done
  return 1
}

# ---------------------------------------------------------------------------
# 0. 环境检查
# ---------------------------------------------------------------------------
log "工作目录: $BACKEND_DIR"
if ! command -v go >/dev/null 2>&1; then
  fail "未找到 go 命令，请先安装 Go 并加入 PATH"
  exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
  log "提示: 未找到 curl，就绪探测将依赖 e2e_check 自身预热"
fi

# ---------------------------------------------------------------------------
# 1. 单元层（不需服务）
# ---------------------------------------------------------------------------
run_phase "单元: fate 评分/喜用神/来源加分" \
  go -C "$BACKEND_DIR" test ./internal/domain/fate/ -count=1

run_phase "单元: 评分快照防漂移(golden) + 八维依据透传(score_detail)" \
  go -C "$BACKEND_DIR" test ./internal/application/services -run 'TestGoldenNames|TestGenerate_EmitsScoreDetail|TestApplyFateScoreDetail' -count=1

# ---------------------------------------------------------------------------
# 2. 门禁表覆盖审计（不需服务）
# ---------------------------------------------------------------------------
run_phase "门禁审计: gate_audit -static-only" \
  go -C "$BACKEND_DIR" run ./tools/gate_audit -static-only

# ---------------------------------------------------------------------------
# 2+. 专项维度巡检：八字/五行/生肖/诗词出典准确性（不需服务）
#     dim_audit 直接调用真实 domain 逻辑，用已知答案 fixtures + 结构不变量校验
# ---------------------------------------------------------------------------
run_phase "专项维度巡检: dim_audit（八字/五行/生肖/诗词）" \
  go -C "$BACKEND_DIR" run ./tools/dim_audit -data "$DATA_DIR"

# ---------------------------------------------------------------------------
# 3. 端到端（需服务）
# ---------------------------------------------------------------------------
if [ "${SKIP_SERVER:-0}" = "1" ]; then
  log "SKIP_SERVER=1：跳过端到端巡检（如需含 E2E，去掉该变量并确保服务可启动）"
else
  # 复用已起服务 / 自动构建并拉起
  if [ -n "${SERVER_URL:-}" ]; then
    URL="$SERVER_URL"
    REUSE_SERVER=1
    log "复用外部服务: $URL"
  elif probe_ready; then
    REUSE_SERVER=1
    log "检测到本地已有服务在 $URL，复用之（不启停）"
  else
    # 按需构建后端
    need_build=0
    if [ ! -f "$BIN" ]; then
      need_build=1
    else
      bin_mt=$(stat -c '%Y' "$BIN" 2>/dev/null || echo 0)
      src_mt=$(find "$BACKEND_DIR" \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) -print0 2>/dev/null \
                | xargs -0 -r stat -c '%Y' 2>/dev/null | sort -n | tail -1)
      cfg_mt=$(find "$BACKEND_DIR" -name '*.yml' -print0 2>/dev/null \
                | xargs -0 -r stat -c '%Y' 2>/dev/null | sort -n | tail -1)
      newest=$(printf '%s\n%s\n' "${src_mt:-0}" "${cfg_mt:-0}" | sort -n | tail -1)
      [ "${bin_mt:-0}" -lt "${newest:-0}" ] && need_build=1
    fi
    if [ "$need_build" -eq 1 ]; then
      log "构建后端 → $BIN …"
      if ! go -C "$BACKEND_DIR" build -o "$BIN" ./cmd/server; then
        fail "后端构建失败，无法启动服务"
        exit 1
      fi
    fi

    mkdir -p "$LOG_DIR"
    log "启动验证服务 → $URL （日志: $SERVER_LOG）"
    DATA_DIR="$DATA_DIR" "$BIN" -host "$HOST" -port "$PORT" -db "$DB_PATH" -log "$LOG_DIR" \
      >>"$SERVER_LOG" 2>&1 &
    SERVER_PID=$!

    # 等待就绪（最多 60s）
    ready=0
    for i in $(seq 1 60); do
      if probe_ready; then ready=1; break; fi
      # 若进程已退出，立即报错
      if ! kill -0 "$SERVER_PID" 2>/dev/null; then break; fi
      sleep 1
    done
    if [ "$ready" -ne 1 ]; then
      fail "服务在 60s 内未就绪（详见 $SERVER_LOG）"
      exit 1
    fi
    log "服务就绪"
  fi

  # e2e_check 的 -host 只需 host:port（工具会自动补 http://），故剥掉协议头
  e2e_host="$HOST:$PORT"
  if [ "$REUSE_SERVER" -eq 1 ] && [ -n "${SERVER_URL:-}" ]; then
    e2e_host="$(printf '%s' "$SERVER_URL" | sed -E 's#^https?://##; s#/.*##')"
  fi

  run_phase "E2E 巡检: e2e_check -strict (repeat=$E2E_REPEAT conc=$E2E_CONCURRENCY rounds=$E2E_ROUNDS)" \
    go -C "$BACKEND_DIR" run ./tools/e2e_check \
      -host "$e2e_host" -repeat "$E2E_REPEAT" -concurrency "$E2E_CONCURRENCY" -rounds "$E2E_ROUNDS" -strict
fi

# ---------------------------------------------------------------------------
# 4. 汇总
# ---------------------------------------------------------------------------
echo
if [ "$FAILURES" -eq 0 ]; then
  printf '\033[1;32m==== 全部质量防线通过 ✅ ====\033[0m\n'
  exit 0
else
  printf '\033[1;31m==== 有 %d 项未通过 ❌ ====\033[0m\n' "$FAILURES"
  exit 1
fi
