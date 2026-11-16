# 起名 - 一键桌面端构建脚本（Wails v3）
# 产物：
#   backend/bin/namer-desktop.exe              单文件桌面 exe
#   backend/bin/namer-amd64-installer.exe     NSIS 安装器（user 作用域 + 物料随装）
#
# 真正的构建逻辑在 backend/Taskfile.yml（wails3 build / wails3 package）：
# 本脚本只负责 CLI 定位、版本护栏与前置检查，避免两套构建逻辑漂移。
# 前端产物由 pnpm build 末尾的 scripts/sync-dist.mjs 同步进 Go embed 目录。

param(
    [string]$Wails3Path = "",   # 未加入 PATH 时显式指定 wails3.exe 路径
    [switch]$SkipInstaller      # 只出裸 exe，跳过 NSIS 安装器构建
)

$RootDir = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $RootDir "backend"

# 1. 定位 wails3 CLI：参数 > PATH > %USERPROFILE%\go\bin\wails3.exe
if ($Wails3Path) {
    $Wails3Exe = $Wails3Path
}
elseif (Get-Command wails3 -ErrorAction SilentlyContinue) {
    $Wails3Exe = "wails3"
}
else {
    $fallback = Join-Path $env:USERPROFILE "go\bin\wails3.exe"
    if (Test-Path $fallback) { $Wails3Exe = $fallback }
    else {
        Write-Host "[错误] 未找到 wails3 CLI，请安装：go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0" -ForegroundColor Red
        Write-Host "       或用 -Wails3Path <路径> 显式指定。" -ForegroundColor Red
        exit 1
    }
}

# 2. 版本护栏：Go module / CLI / @wailsio/runtime 必须精确 pin 同一版本
#    （v3 GA 后是 nightly + 每周次版本，三者错位会导致绑定 ID 漂移，docs/48 §2）
$goMod = Get-Content (Join-Path $BackendDir "go.mod") -Raw
if ($goMod -notmatch 'github\.com/wailsapp/wails/v3 v3\.0\.0') {
    Write-Host "[错误] backend/go.mod 未锁定 wails/v3 v3.0.0，终止构建。" -ForegroundColor Red
    exit 1
}
$pkgJson = Get-Content (Join-Path $RootDir "frontend\package.json") -Raw
if ($pkgJson -notmatch '"@wailsio/runtime": "3\.0\.0"') {
    Write-Host "[错误] frontend/package.json 未锁定 @wailsio/runtime 3.0.0，终止构建。" -ForegroundColor Red
    exit 1
}

# 3. makensis 只认 PATH；缺失时仅跳过安装器，不影响 exe 构建
$HasMakensis = $false
if (-not $SkipInstaller) {
    if (Get-Command makensis -ErrorAction SilentlyContinue) { $HasMakensis = $true }
    else {
        Write-Host "[提示] 未找到 makensis（NSIS），跳过安装器构建。" -ForegroundColor Yellow
        Write-Host "       安装后重跑本脚本即可出安装器：winget install NSIS.NSIS" -ForegroundColor Yellow
    }
}

Write-Host "=== 起名 桌面端构建（Wails v3.0.0） ===" -ForegroundColor Cyan
Write-Host "CLI: $Wails3Exe"
Write-Host ""

Push-Location $BackendDir
try {
    # wails3 build = Taskfile 的 build 任务：pnpm build(sync-dist) + go build -tags production
    Write-Host "[1/2] 构建桌面 exe ..." -ForegroundColor Yellow
    & $Wails3Exe build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [错误] wails3 build 失败" -ForegroundColor Red
        exit 1
    }
    Write-Host "  [完成] backend\bin\namer-desktop.exe" -ForegroundColor Green

    if ($HasMakensis) {
        # wails3 package = Taskfile 的 package 任务：生成 WebView2 bootstrapper + makensis
        # 传入 WAILS3 变量，让子任务用同一个 CLI 可执行文件（CLI 可能不在 PATH）。
        Write-Host "[2/2] 构建 NSIS 安装器（user 作用域 + 物料随装）..." -ForegroundColor Yellow
        & $Wails3Exe package "WAILS3=$Wails3Exe"
        if ($LASTEXITCODE -ne 0) {
            Write-Host "  [错误] wails3 package 失败" -ForegroundColor Red
            exit 1
        }
        Write-Host "  [完成] backend\bin\namer-amd64-installer.exe" -ForegroundColor Green
    }
    else {
        Write-Host "[2/2] 跳过 NSIS 安装器（-SkipInstaller 或未检测到 makensis）" -ForegroundColor Gray
    }
}
finally {
    Pop-Location
}

Write-Host ""
Write-Host "=== 构建完成 ===" -ForegroundColor Cyan
Write-Host "产物: $(Join-Path $BackendDir 'bin\namer-desktop.exe')"
if ($HasMakensis) {
    Write-Host "安装器: $(Join-Path $BackendDir 'bin\namer-amd64-installer.exe')"
}
Write-Host "启动方式: 双击 exe，或在 backend 目录运行 .\bin\namer-desktop.exe"
