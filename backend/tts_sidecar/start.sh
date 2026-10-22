#!/usr/bin/env bash
# 离线粤语 TTS sidecar 启动脚本（Linux / macOS / Git Bash）
# 拉起 Audio8 0.1B INT8 ONNX Runtime 官方服务（默认监听 127.0.0.1:8024）。
# Go 后端 /api/v1/tts 会代理到它；Ctrl+C 停止本脚本即停止该服务。
#
# 依赖：Git、Python 3.10+、网络可访问 github.com 与模型镜像。
# 模型下载支持镜像（环境变量 MIRROR，默认 modelscope）：
#   modelscope  -> 魔搭社区（国内推荐，直连 www.modelscope.cn，无需登录公开模型）
#   hf-mirror   -> HF 镜像（HF_ENDPOINT=https://hf-mirror.com，仓库在镜像侧为 Edge0/...）
#   hf          -> 直连 huggingface.co（Audio8/...）
# 也可整体设置 HF_ENDPOINT 自行指定 HF 镜像。首次下载约 0.86GB。
set -euo pipefail

MIRROR="${MIRROR:-modelscope}"

REPO_DIR="$(dirname "$0")/Audio8_TTS"
RUNTIME_DIR="$REPO_DIR/onnx_runtime_0_1b_int8"

if [ ! -d "$RUNTIME_DIR" ]; then
  echo '==> 克隆 Audio8_TTS 仓库（首次）'
  git clone https://github.com/Audio8-AI/Audio8_TTS.git "$REPO_DIR"
fi

cd "$RUNTIME_DIR"

# --- 模型下载（镜像可选）---
if [ ! -f model/reference_codes.npy ]; then
  echo "==> 下载 0.1B INT8 ONNX 模型（约 0.86GB，首次）[MIRROR=$MIRROR]"
  case "$MIRROR" in
    modelscope)
      python3 -m pip install -U modelscope
      python3 -m modelscope download --model Audio8/audio8-TTS-0.1B-ONNX-INT8 --local_dir model
      ;;
    hf-mirror)
      python3 -m pip install -U "huggingface_hub[cli]"
      export HF_ENDPOINT="${HF_ENDPOINT:-https://hf-mirror.com}"
      hf download Edge0/audio8-TTS-0.1B-ONNX-INT8 --local-dir model
      ;;
    *)
      python3 -m pip install -U "huggingface_hub[cli]"
      hf download Audio8/audio8-TTS-0.1B-ONNX-INT8 --local-dir model
      ;;
  esac
fi

if [ ! -d .venv ]; then
  echo '==> 安装运行时依赖（setup.sh）'
  bash setup.sh
fi

if [ ! -f model/registration/reference_codes.npy ]; then
  echo '==> 注册默认音色'
  ./.venv/bin/python scripts/register_default_voice.py
fi

echo '==> 启动离线粤语 TTS 服务（127.0.0.1:8024）'
bash start_server.sh
