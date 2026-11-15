# 起名 - 一键桌面端构建脚本（Wails）
# 产物：单文件桌面 exe（前端 dist 由 wails build 自动同步进 embed 目录）
#       + 可选 NSIS 安装器（exe + 物料随装，user 作用域安装到 %LocalAppData%\Programs）

param(
    [string]$OutDir = "backend\build\bin",
    [switch]$SkipInstaller  # 跳过 NSIS 安装器构建（未装 makensis 或只需裸 exe 时）
)

$RootDir = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $RootDir "backend"
$FrontendDir = Join-Path $RootDir "frontend"

$WailsExe = "wails"
# 常见安装位置兜底（wails CLI 不在 PATH 时）
if (-not (Get-Command $WailsExe -ErrorAction SilentlyContinue)) {
    $fallback = Join-Path $env:USERPROFILE "go\bin\wails.exe"
    if (Test-Path $fallback) { $WailsExe = $fallback }
    else {
        Write-Host "[错误] 未找到 wails CLI，请先安装：go install github.com/wailsapp/wails/v2/cmd/wails@latest" -ForegroundColor Red
        exit 1
    }
}

# makensis 只认 PATH（wails 同款探测逻辑）；缺失时仅跳过安装器，不影响 exe 构建
$HasMakensis = $false
if (-not $SkipInstaller) {
    if (Get-Command makensis -ErrorAction SilentlyContinue) {
        $HasMakensis = $true
    }
    else {
        Write-Host "[提示] 未找到 makensis（NSIS），跳过安装器构建。" -ForegroundColor Yellow
        Write-Host "       安装后重跑本脚本即可出安装器：winget install NSIS.NSIS" -ForegroundColor Yellow
        Write-Host "       （wails 只在 PATH 中查找 makensis，安装后需重开终端）" -ForegroundColor Yellow
    }
}

Write-Host "=== 起名 桌面端构建（Wails） ===" -ForegroundColor Cyan
Write-Host ""

# 1. 前端依赖 + 构建
Write-Host "[1/3] 构建前端（Vite）..." -ForegroundColor Yellow
Push-Location $FrontendDir
try {
    if (-not (Test-Path "node_modules")) {
        Write-Host "  -> 安装 pnpm 依赖..." -ForegroundColor Gray
        pnpm install
        if ($LASTEXITCODE -ne 0) {
            Write-Host "  [错误] pnpm install 失败" -ForegroundColor Red
            exit 1
        }
    }
    pnpm build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [错误] 前端构建失败" -ForegroundColor Red
        exit 1
    }
}
finally {
    Pop-Location
}
Write-Host "  [完成] 前端产物: frontend\dist" -ForegroundColor Green

# 2. wails build（自动同步 frontend/dist → backend/frontend/dist 并编译进 exe）
Write-Host "[2/3] Wails 打包..." -ForegroundColor Yellow
Push-Location $BackendDir
try {
    & $WailsExe build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [错误] wails build 失败" -ForegroundColor Red
        exit 1
    }
}
finally {
    Pop-Location
}
Write-Host "  [完成] 桌面 exe: backend\build\bin\namer-desktop.exe" -ForegroundColor Green

# 3. NSIS 安装器（wails build -nsis；物料随装与卸载保库逻辑见 build/windows/installer/project.nsi）
if ($HasMakensis) {
    Write-Host "[3/3] 构建 NSIS 安装器（user 作用域 + 物料随装）..." -ForegroundColor Yellow
    Push-Location $BackendDir
    try {
        # user 作用域：装到 %LocalAppData%\Programs\NameMaster，免管理员且目录可写
        # ——便携数据语义（db/logs 收敛 exe 同级 data/）依赖安装目录可写
        & $WailsExe build -nsis -installscope user
        if ($LASTEXITCODE -ne 0) {
            Write-Host "  [错误] NSIS 安装器构建失败" -ForegroundColor Red
            exit 1
        }
    }
    finally {
        Pop-Location
    }
    Write-Host "  [完成] 安装器: backend\build\bin\namer-amd64-installer.exe" -ForegroundColor Green
}
else {
    Write-Host "[3/3] 跳过 NSIS 安装器（-SkipInstaller 或未检测到 makensis）" -ForegroundColor Gray
}

Write-Host ""
Write-Host "=== 构建完成 ===" -ForegroundColor Cyan
$exePath = Join-Path $BackendDir "build\bin\namer-desktop.exe"
Write-Host "产物: $exePath"
if ($HasMakensis) {
    Write-Host "安装器: $(Join-Path $BackendDir 'build\bin\namer-amd64-installer.exe')"
}
Write-Host "启动方式: 双击 exe，或后端目录运行 .\build\bin\namer-desktop.exe"
