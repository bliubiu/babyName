# 离线粤语 TTS sidecar 启动脚本（Windows）
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
# 模型目录默认为本脚本同级 ..\model（即 backend\tts_sidecar\model，已随项目落地），
# 可用 MODEL_DIR 覆盖；服务通过 ARKTTS_MODEL_DIR 感知该路径。
$ErrorActionPreference = 'Stop'

$MIRROR = if ($env:MIRROR) { $env:MIRROR } else { 'modelscope' }
$Port = if ($env:PORT) { $env:PORT } else { '8024' }

$Here = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoDir = Join-Path $Here 'Audio8_TTS'
$RuntimeDir = Join-Path $RepoDir 'onnx_runtime_0_1b_int8'
if ($env:MODEL_DIR) { $ModelDir = $env:MODEL_DIR } else { $ModelDir = Join-Path $Here 'model' }

# --- 运行时代码（github.com 被墙时自动回退 gh-proxy.com 镜像，稀疏克隆只取 onnx_runtime_0_1b_int8）---
if (-not (Test-Path $RuntimeDir)) {
    if (Test-Path (Join-Path $RepoDir '.git')) {
        Push-Location $RepoDir
        try { git sparse-checkout set onnx_runtime_0_1b_int8 } catch { }
        finally { Pop-Location }
    }
}
if (-not (Test-Path $RuntimeDir)) {
    Write-Host '==> 获取 Audio8_TTS 运行时代码（首次）'
    $cloneOk = $false
    foreach ($base in @('https://github.com', 'https://gh-proxy.com/https://github.com')) {
        Write-Host "    clone: $base/Audio8-AI/Audio8_TTS"
        if (Test-Path $RepoDir) { Remove-Item -Recurse -Force $RepoDir }
        & git clone --depth 1 --filter=blob:none --sparse "$base/Audio8-AI/Audio8_TTS" $RepoDir
        if ($LASTEXITCODE -eq 0) {
            Push-Location $RepoDir
            try { & git sparse-checkout set onnx_runtime_0_1b_int8 } finally { Pop-Location }
            $cloneOk = $true
            break
        }
    }
    if (-not $cloneOk) {
        Write-Error '克隆失败：github.com 与 gh-proxy.com 均不可达，请检查网络或手动放置代码到 Audio8_TTS\onnx_runtime_0_1b_int8'
        exit 1
    }
}

Push-Location $RuntimeDir
try {
    # --- 模型下载（镜像可选；modelscope 入口是 console script，不是 python -m modelscope）---
    if (-not (Test-Path (Join-Path $ModelDir 'reference_codes.npy'))) {
        Write-Host "==> 下载 0.1B INT8 ONNX 模型（约 0.86GB，首次）[MIRROR=$MIRROR] -> $ModelDir"
        New-Item -ItemType Directory -Force -Path $ModelDir | Out-Null
        switch ($MIRROR) {
            'modelscope' {
                py -3 -m pip install -U modelscope
                $modelscopeCmd = Get-Command modelscope -ErrorAction SilentlyContinue
                if (-not $modelscopeCmd) { py -3 -m pip install --user -U modelscope }
                & modelscope download --model Audio8/audio8-TTS-0.1B-ONNX-INT8 --local_dir $ModelDir
            }
            'hf-mirror' {
                py -3 -m pip install -U huggingface_hub
                $env:HF_ENDPOINT = if ($env:HF_ENDPOINT) { $env:HF_ENDPOINT } else { 'https://hf-mirror.com' }
                & hf download Edge0/audio8-TTS-0.1B-ONNX-INT8 --local-dir $ModelDir
            }
            default {
                py -3 -m pip install -U huggingface_hub
                & hf download Audio8/audio8-TTS-0.1B-ONNX-INT8 --local-dir $ModelDir
            }
        }
    }

    # --- Python 解释器解析：ARKTTS_PYTHON > .venv > 系统 Python（已装依赖时快速路径）> setup.ps1 建 venv ---
    function Test-PyDeps($py) {
        & $py -c 'import onnxruntime, fastapi, uvicorn, soundfile, tokenizers, scipy, pydantic, multipart' *> $null
        return $LASTEXITCODE -eq 0
    }

    $PyBin = $null
    if ($env:ARKTTS_PYTHON) {
        $PyBin = $env:ARKTTS_PYTHON
    } elseif (Test-Path '.venv\Scripts\python.exe') {
        $PyBin = (Resolve-Path '.venv\Scripts\python.exe').Path
    } elseif ((Get-Command python -ErrorAction SilentlyContinue) -and (Test-PyDeps 'python')) {
        $PyBin = 'python'
    } elseif ((Get-Command py -ErrorAction SilentlyContinue) -and (Test-PyDeps 'py -3')) {
        $PyBin = 'py'
    } else {
        Write-Host '==> 安装运行时依赖（setup.ps1，首次约 1-2 分钟）'
        & .\setup.ps1
        $PyBin = (Resolve-Path '.venv\Scripts\python.exe').Path
    }
    Write-Host "==> Python: $PyBin"

    # --- 注册默认音色（reference_text 取自 runtime_manifest.json，与 reference_codes 对齐；缺失或 FORCE_REGISTER=1 时重建）---
    if (-not (Test-Path 'voices\default\codes.npy') -or $env:FORCE_REGISTER -eq '1') {
        Write-Host '==> 注册默认音色'
        $pyArgs = @($PyBin -split ' ') + @('scripts/register_default_voice.py', '--model-dir', $ModelDir, '--voices-dir', (Join-Path $RuntimeDir 'voices'), '--overwrite')
        & $pyArgs[0] $pyArgs[1..($pyArgs.Length - 1)]
    }

    # --- 启动服务（前台；环境变量契约与官方 run_server.ps1 一致）---
    $env:ARKTTS_MODEL_DIR = $ModelDir
    $env:ARKTTS_VOICES_DIR = Join-Path $RuntimeDir 'voices'
    $env:ARKTTS_REGISTRATION_DIR = Join-Path $ModelDir 'registration'
    $env:ARKTTS_PRECISION = if ($env:ARKTTS_PRECISION) { $env:ARKTTS_PRECISION } else { 'int8' }
    $env:ARKTTS_CODEC_PRECISION = if ($env:ARKTTS_CODEC_PRECISION) { $env:ARKTTS_CODEC_PRECISION } else { 'fp16' }
    $env:ARKTTS_THREADS = if ($env:ARKTTS_THREADS) { $env:ARKTTS_THREADS } else { '5' }
    $env:PYTHONIOENCODING = if ($env:PYTHONIOENCODING) { $env:PYTHONIOENCODING } else { 'utf-8' }
    $env:NO_PROXY = if ($env:NO_PROXY) { $env:NO_PROXY } else { '127.0.0.1,localhost' }

    Write-Host "==> 启动离线粤语 TTS 服务（http://127.0.0.1:$Port）"
    if ($PyBin -eq 'py') {
        & py -3 -m uvicorn arktts_runtime.service:app --app-dir $RuntimeDir --host 127.0.0.1 --port $Port
    } else {
        & $PyBin -m uvicorn arktts_runtime.service:app --app-dir $RuntimeDir --host 127.0.0.1 --port $Port
    }
} finally {
    Pop-Location
}
