#!/usr/bin/env bash
# 离线粤语 TTS sidecar 启动脚本（Linux / macOS / Git Bash）
# 拉起 Audio8 0.1B INT8 ONNX Runtime 官方服务（默认监听 127.0.0.1:8024）。
# Go 后端 /api/v1/tts 会代理到它；Ctrl+C 停止本脚本即停止该服务。
#
# 依赖：Git、Python 3.10+、网络可访问 github.com（或 gh-proxy 镜像）与模型镜像。
# 模型下载支持镜像（环境变量 MIRROR，默认 modelscope）：
#   modelscope  -> 魔搭社区（国内推荐，直连 www.modelscope.cn，无需登录公开模型）
#   hf-mirror   -> HF 镜像（HF_ENDPOINT=https://hf-mirror.com，仓库在镜像侧为 Edge0/...）
#   hf          -> 直连 huggingface.co（Audio8/...）
# 也可整体设置 HF_ENDPOINT 自行指定 HF 镜像。首次下载约 0.86GB。
#
# 模型目录默认为本脚本同级 ../model（即 backend/tts_sidecar/model，已随项目落地），
# 可用 MODEL_DIR 覆盖；服务通过 ARKTTS_MODEL_DIR 感知该路径。
set -euo pipefail

MIRROR="${MIRROR:-modelscope}"
MODEL_DIR="${MODEL_DIR:-$(cd "$(dirname "$0")" && pwd)/model}"
PORT="${PORT:-8024}"

HERE="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$HERE/Audio8_TTS"
RUNTIME_DIR="$REPO_DIR/onnx_runtime_0_1b_int8"

# --- 运行时代码（github.com 被墙时自动回退 gh-proxy.com 镜像，稀疏克隆只取 onnx_runtime_0_1b_int8）---
if [ ! -d "$RUNTIME_DIR" ]; then
  # 已有仓库但没做 sparse-checkout 的情况（克隆中断）
  if [ -d "$REPO_DIR/.git" ]; then
    (cd "$REPO_DIR" && git sparse-checkout set onnx_runtime_0_1b_int8) || true
  fi
fi
if [ ! -d "$RUNTIME_DIR" ]; then
  echo '==> 获取 Audio8_TTS 运行时代码（首次）'
  clone_ok=0
  for base in "https://github.com" "https://gh-proxy.com/https://github.com"; do
    echo "    clone: $base/Audio8-AI/Audio8_TTS"
    rm -rf "$REPO_DIR"
    if git clone --depth 1 --filter=blob:none --sparse "$base/Audio8-AI/Audio8_TTS" "$REPO_DIR"; then
      (cd "$REPO_DIR" && git sparse-checkout set onnx_runtime_0_1b_int8)
      clone_ok=1
      break
    fi
  done
  if [ "$clone_ok" != 1 ]; then
    echo '克隆失败：github.com 与 gh-proxy.com 均不可达，请检查网络或手动放置代码到 Audio8_TTS/onnx_runtime_0_1b_int8' >&2
    exit 1
  fi
fi

cd "$RUNTIME_DIR"

# --- 模型下载（镜像可选；modelscope 入口是 console script，不是 python -m modelscope）---
if [ ! -f "$MODEL_DIR/reference_codes.npy" ]; then
  echo "==> 下载 0.1B INT8 ONNX 模型（约 0.86GB，首次）[MIRROR=$MIRROR] -> $MODEL_DIR"
  mkdir -p "$MODEL_DIR"
  case "$MIRROR" in
    modelscope)
      python3 -m pip install -U modelscope
      command -v modelscope >/dev/null 2>&1 || python3 -m pip install --user -U modelscope
      modelscope download --model Audio8/audio8-TTS-0.1B-ONNX-INT8 --local_dir "$MODEL_DIR"
      ;;
    hf-mirror)
      python3 -m pip install -U "huggingface_hub[cli]"
      export HF_ENDPOINT="${HF_ENDPOINT:-https://hf-mirror.com}"
      hf download Edge0/audio8-TTS-0.1B-ONNX-INT8 --local-dir "$MODEL_DIR"
      ;;
    *)
      python3 -m pip install -U "huggingface_hub[cli]"
      hf download Audio8/audio8-TTS-0.1B-ONNX-INT8 --local-dir "$MODEL_DIR"
      ;;
  esac
fi

# --- Python 解释器解析：ARKTTS_PYTHON > .venv > 系统 Python（已装依赖时快速路径）> setup.sh 建 venv ---
has_deps() { "$1" -c 'import onnxruntime, fastapi, uvicorn, soundfile, tokenizers, scipy, pydantic, multipart' >/dev/null 2>&1; }

PY_BIN=""
if [ -n "${ARKTTS_PYTHON:-}" ]; then
  PY_BIN="$ARKTTS_PYTHON"
elif [ -x "$RUNTIME_DIR/.venv/bin/python" ]; then
  PY_BIN="$RUNTIME_DIR/.venv/bin/python"
elif [ -x "$RUNTIME_DIR/.venv/Scripts/python.exe" ]; then
  PY_BIN="$RUNTIME_DIR/.venv/Scripts/python.exe"
elif command -v python3 >/dev/null 2>&1 && has_deps python3; then
  PY_BIN=python3
elif command -v python >/dev/null 2>&1 && has_deps python; then
  PY_BIN=python
else
  echo '==> 安装运行时依赖（setup.sh，首次约 1-2 分钟）'
  bash setup.sh
  if [ -x "$RUNTIME_DIR/.venv/bin/python" ]; then
    PY_BIN="$RUNTIME_DIR/.venv/bin/python"
  else
    PY_BIN="$RUNTIME_DIR/.venv/Scripts/python.exe"
  fi
fi
echo "==> Python: $PY_BIN"

# --- 注册默认音色（reference_text 取自 runtime_manifest.json，与 reference_codes 对齐；缺失或 FORCE_REGISTER=1 时重建）---
if [ ! -f "$RUNTIME_DIR/voices/default/codes.npy" ] || [ "${FORCE_REGISTER:-0}" = 1 ]; then
  echo '==> 注册默认音色'
  "$PY_BIN" scripts/register_default_voice.py --model-dir "$MODEL_DIR" --voices-dir "$RUNTIME_DIR/voices" --overwrite
fi

# --- 启动服务（前台；环境变量契约与官方 run_server.sh 一致）---
export ARKTTS_MODEL_DIR="$MODEL_DIR"
export ARKTTS_VOICES_DIR="$RUNTIME_DIR/voices"
export ARKTTS_REGISTRATION_DIR="$MODEL_DIR/registration"
export ARKTTS_PRECISION="${ARKTTS_PRECISION:-int8}"
export ARKTTS_CODEC_PRECISION="${ARKTTS_CODEC_PRECISION:-fp16}"
export ARKTTS_THREADS="${ARKTTS_THREADS:-5}"
export PYTHONIOENCODING="${PYTHONIOENCODING:-utf-8}"
export NO_PROXY="${NO_PROXY:-127.0.0.1,localhost}"
export no_proxy="${no_proxy:-127.0.0.1,localhost}"

echo "==> 启动离线粤语 TTS 服务（http://127.0.0.1:$PORT）"
exec "$PY_BIN" -m uvicorn arktts_runtime.service:app --app-dir "$RUNTIME_DIR" --host 127.0.0.1 --port "$PORT"
