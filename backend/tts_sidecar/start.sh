#!/usr/bin/env bash
# 离线粤语 TTS sidecar 启动脚本（Linux / macOS）
# 拉起 Audio8 0.1B INT8 ONNX Runtime 官方服务（默认监听 127.0.0.1:8024）。
# Go 后端 /api/v1/tts 会代理到它；Ctrl+C 停止本脚本即停止该服务。
#
# 依赖：Git、Python 3.10+、可访问 github.com 与 huggingface.co（首次下载约 0.6GB 模型）。
set -euo pipefail

REPO_DIR="$(dirname "$0")/Audio8_TTS"
RUNTIME_DIR="$REPO_DIR/onnx_runtime_0_1b_int8"

if [ ! -d "$RUNTIME_DIR" ]; then
  echo '==> 克隆 Audio8_TTS 仓库（首次）'
  git clone https://github.com/Audio8-AI/Audio8_TTS.git "$REPO_DIR"
fi

cd "$RUNTIME_DIR"

python3 -m pip install -U "huggingface_hub[cli]"

if [ ! -f model/reference_codes.npy ]; then
  echo '==> 下载 0.1B INT8 ONNX 模型（约 0.6GB，首次）'
  hf download Audio8/audio8-TTS-0.1B-ONNX-INT8 --local-dir model
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
