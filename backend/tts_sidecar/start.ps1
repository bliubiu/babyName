# 离线粤语 TTS sidecar 启动脚本（Windows）
# 拉起 Audio8 0.1B INT8 ONNX Runtime 官方服务（默认监听 127.0.0.1:8024）。
# Go 后端 /api/v1/tts 会代理到它；Ctrl+C 停止本脚本即停止该服务。
#
# 依赖：Git、Python 3.10+、网络可访问 github.com 与模型镜像。
# 模型下载支持镜像（环境变量 MIRROR，默认 modelscope）：
#   modelscope  -> 魔搭社区（国内推荐，直连 www.modelscope.cn，无需登录公开模型）
#   hf-mirror   -> HF 镜像（HF_ENDPOINT=https://hf-mirror.com，仓库在镜像侧为 Edge0/...）
#   hf          -> 直连 huggingface.co（Audio8/...）
# 也可整体设置 HF_ENDPOINT 自行指定 HF 镜像。首次下载约 0.86GB。
$ErrorActionPreference = 'Stop'

$MIRROR = if ($env:MIRROR) { $env:MIRROR } else { 'modelscope' }

$RepoDir = Join-Path $PSScriptRoot 'Audio8_TTS'
$RuntimeDir = Join-Path $RepoDir 'onnx_runtime_0_1b_int8'

if (-not (Test-Path $RuntimeDir)) {
    Write-Host '==> 克隆 Audio8_TTS 仓库（首次）'
    git clone https://github.com/Audio8-AI/Audio8_TTS.git $RepoDir
}

Push-Location $RuntimeDir
try {
    if (-not (Test-Path 'model\reference_codes.npy')) {
        Write-Host "==> 下载 0.1B INT8 ONNX 模型（约 0.86GB，首次）[MIRROR=$MIRROR]"
        switch ($MIRROR) {
            'modelscope' {
                py -3 -m pip install -U modelscope
                py -3 -m modelscope download --model Audio8/audio8-TTS-0.1B-ONNX-INT8 --local_dir model
            }
            'hf-mirror' {
                py -3 -m pip install -U huggingface_hub
                $env:HF_ENDPOINT = if ($env:HF_ENDPOINT) { $env:HF_ENDPOINT } else { 'https://hf-mirror.com' }
                hf download Edge0/audio8-TTS-0.1B-ONNX-INT8 --local-dir model
            }
            default {
                py -3 -m pip install -U huggingface_hub
                hf download Audio8/audio8-TTS-0.1B-ONNX-INT8 --local-dir model
            }
        }
    }

    if (-not (Test-Path '.venv')) {
        Write-Host '==> 安装运行时依赖（setup.ps1）'
        .\setup.ps1
    }

    if (-not (Test-Path 'model\registration\reference_codes.npy')) {
        Write-Host '==> 注册默认音色'
        .\.venv\Scripts\python.exe scripts/register_default_voice.py
    }

    Write-Host '==> 启动离线粤语 TTS 服务（127.0.0.1:8024）'
    .\start_server.ps1
} finally {
    Pop-Location
}
