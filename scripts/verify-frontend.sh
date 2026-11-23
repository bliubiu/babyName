#!/usr/bin/env bash
# =============================================================================
# verify-frontend.sh — 前端三件套质量门禁（docs/45 阶段 0 / P1-29）
#
# 覆盖：
#   ① TypeScript 类型检查：tsc --noEmit
#   ② ESLint 静态检查：warning 阈值（不要求一次清零，但不允许新增）
#   ③ Vitest 单元测试：vitest run
#
# 与 Go 门禁（scripts/verify-quality.sh）刻意解耦：本仓库 Go 侧门禁不需要 Node，
# 前端也不需要 Go；两者各自独立失败、独立修复。
#   - CI：独立 job 用 `--require` 调用，缺 node/依赖一律失败；
#   - pre-push：不带 `--require`，本地未安装前端依赖时只提示并跳过，
#     避免纯后端改动被前端环境误伤（hook 本就可被 SKIP_VERIFY 绕过，
#     所以 CI 侧必须独立存在，不能只靠 hook）。
#
# 用法：
#   bash scripts/verify-frontend.sh                 # 本地 / CI 通用
#   bash scripts/verify-frontend.sh --require      # CI：缺 node/依赖即失败
#   SKIP_FRONTEND=1 git push                       # 临时跳过（pre-push 侧）
#   ESLINT_MAX_WARNINGS=30 bash scripts/verify-frontend.sh   # 收紧阈值
#
# 退出码：任一阶段失败即以 1 退出，便于接入 CI 与 hook。
# =============================================================================

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
FE_DIR="$ROOT_DIR/frontend"

# 2026-11-09 实测基线：eslint src --ext .ts,.tsx → 0 error / 34 warning。
# 阈值只许下调：每清理一条 warning 就同步把这里减 1，防止 lint 债务回流。
ESLINT_MAX_WARNINGS="${ESLINT_MAX_WARNINGS:-34}"

REQUIRED=0
for arg in "$@"; do
  case "$arg" in
    --require) REQUIRED=1 ;;
    *) printf '[前端门禁] 未知参数: %s\n' "$arg" >&2; exit 2 ;;
  esac
done

if [ "${SKIP_FRONTEND:-0}" = "1" ]; then
  echo "[前端门禁] SKIP_FRONTEND=1：跳过前端三件套"
  exit 0
fi

FAILURES=0
log()  { printf '\033[1;34m[前端门禁]\033[0m %s\n' "$*"; }
pass() { printf '\033[1;32m[PASS]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[FAIL]\033[0m %s\n' "$*"; FAILURES=$((FAILURES+1)); }

run_phase() {
  local name="$1"; shift
  log "▶ $name"
  local rc=0
  "$@" || rc=$?
  if [ "$rc" -eq 0 ]; then
    pass "$name"
  else
    fail "$name (退出码 $rc)"
  fi
}

# ---------------------------------------------------------------------------
# 0. 环境检查
#    本地缺依赖 = 跳过（提示如何安装）；CI 用了 --require = 直接失败。
# ---------------------------------------------------------------------------
skip_or_fail() {
  local msg="$1"
  if [ "$REQUIRED" -eq 1 ]; then
    fail "$msg（CI 使用 --require，不允许降级）"
    return 1
  fi
  log "$msg —— 本地跳过；如需本地跑前端门禁，先在 frontend/ 安装依赖"
  return 0
}

if ! command -v node >/dev/null 2>&1; then
  skip_or_fail "未找到 node 命令" || exit 1
  exit 0
fi

if [ ! -d "$FE_DIR/node_modules" ]; then
  skip_or_fail "frontend/node_modules 不存在" || exit 1
  exit 0
fi

for entry in \
  node_modules/typescript/lib/tsc.js \
  node_modules/eslint/bin/eslint.js \
  node_modules/vitest/vitest.mjs; do
  if [ ! -f "$FE_DIR/$entry" ]; then
    skip_or_fail "缺少前端门禁入口 $entry" || exit 1
    exit 0
  fi
done

cd "$FE_DIR" || { fail "无法进入 $FE_DIR"; exit 1; }

log "工作目录: $FE_DIR"
log "ESLint warning 阈值: ${ESLINT_MAX_WARNINGS}（当前基线 34，只许下调）"

# ---------------------------------------------------------------------------
# 1~3. 三件套
#      直调 node_modules 内的 CLI 入口，不依赖全局 pnpm / npx，
#      避免本机 pnpm 缺失或联网拉取版本造成的环境差异。
# ---------------------------------------------------------------------------
run_phase "类型检查: tsc --noEmit" \
  node node_modules/typescript/lib/tsc.js --noEmit

run_phase "静态检查: eslint src --ext .ts,.tsx --max-warnings ${ESLINT_MAX_WARNINGS}" \
  node node_modules/eslint/bin/eslint.js src --ext .ts,.tsx --max-warnings "$ESLINT_MAX_WARNINGS"

run_phase "单元测试: vitest run" \
  node node_modules/vitest/vitest.mjs run --reporter=dot

# ---------------------------------------------------------------------------
# 4. 汇总
# ---------------------------------------------------------------------------
echo
if [ "$FAILURES" -eq 0 ]; then
  printf '\033[1;32m==== 前端三件套全部通过 ✅ ====\033[0m\n'
  exit 0
else
  printf '\033[1;31m==== 前端门禁有 %d 项未通过 ❌ ====\033[0m\n' "$FAILURES"
  exit 1
fi
