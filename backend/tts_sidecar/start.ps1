# 离线粤语 TTS sidecar 启动脚本（Windows）
# 拉起 Audio8 0.1B INT8 ONNX Runtime 官方服务（默认监听 127.0.0.1:8024）。
# Go 后端 /api/v1/tts 会代理到它；Ctrl+C 停止本脚本即停止该服务。
#
# 依赖：Git、Python 3.10+、可访问 github.com 与 huggingface.co（首次下载约 0.6GB 模型）。
$ErrorActionPreference = 'Stop'

$RepoDir = Join-Path $PSScriptRoot 'Audio8_TTS'
$RuntimeDir = Join-Path $RepoDir 'onnx_runtime_0_1b_int8'

if (-not (Test-Path $RuntimeDir)) {
    Write-Host '==> 克隆 Audio8_TTS 仓库（首次）'
    git clone https://github.com/Audio8-AI/Audio8_TTS.git $RepoDir
}

Push-Location $RuntimeDir
try {
    Write-Host '==> 安装 huggingface_hub CLI'
    py -3 -m pip install -U huggingface_hub

    if (-not (Test-Path 'model\reference_codes.npy')) {
        Write-Host '==> 下载 0.1B INT8 ONNX 模型（约 0.6GB，首次）'
        hf download Audio8/audio8-TTS-0.1B-ONNX-INT8 --local-dir model
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
