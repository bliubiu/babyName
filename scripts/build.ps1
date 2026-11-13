# 起名 - 一键桌面端构建脚本（Wails）
# 产物：单文件桌面 exe（前端 dist 由 wails build 自动同步进 embed 目录）

param(
    [string]$OutDir = "backend\build\bin"
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

Write-Host "=== 起名 桌面端构建（Wails） ===" -ForegroundColor Cyan
Write-Host ""

# 1. 前端依赖 + 构建
Write-Host "[1/2] 构建前端（Vite）..." -ForegroundColor Yellow
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
Write-Host "[2/2] Wails 打包..." -ForegroundColor Yellow
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

Write-Host ""
Write-Host "=== 构建完成 ===" -ForegroundColor Cyan
$exePath = Join-Path $BackendDir "build\bin\namer-desktop.exe"
Write-Host "产物: $exePath"
Write-Host "启动方式: 双击 exe，或后端目录运行 .\build\bin\namer-desktop.exe"
