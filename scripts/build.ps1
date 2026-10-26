# 起名 - 一键构建脚本（SSR 全栈：无前端构建步骤）
# 产物：dist/namer.exe + dist/application.yml + dist/data/

param(
    [string]$Platform = "windows", # windows | linux | darwin
    [string]$Arch = "amd64",       # amd64 | arm64
    [string]$Port = "8080",
    [string]$OutDir = "./dist"
)

$RootDir = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $RootDir "backend"
$DataDir = Join-Path $RootDir "data"
$DistDir = Join-Path $RootDir $OutDir

Write-Host "=== 起名 构建脚本 ===" -ForegroundColor Cyan
Write-Host "平台: $Platform/$Arch"
Write-Host "输出目录: $DistDir"
Write-Host ""

New-Item -ItemType Directory -Force -Path $DistDir | Out-Null

# 1. 编译后端（SSR 模板与 Tailwind 已通过 go:embed 编入二进制）
Write-Host "[1/3] 编译后端二进制..." -ForegroundColor Yellow
Set-Location $BackendDir

$env:GOOS = $Platform
$env:GOARCH = $Arch

$binaryName = "namer"
if ($Platform -eq "windows") {
    $binaryName += ".exe"
}

Write-Host "  -> 下载 Go 依赖..." -ForegroundColor Gray
go mod tidy
if ($LASTEXITCODE -ne 0) {
    Write-Host "  [警告] go mod tidy 有问题，继续编译..." -ForegroundColor Yellow
}

Write-Host "  -> 编译 $binaryName ..." -ForegroundColor Gray
go build -o (Join-Path $DistDir $binaryName) -ldflags="-s -w" ./cmd/server/
if ($LASTEXITCODE -ne 0) {
    Write-Host "  [错误] 后端编译失败" -ForegroundColor Red
    exit 1
}
Write-Host "  [完成] 二进制: $DistDir\$binaryName" -ForegroundColor Green

# 2. 复制运行时配置与字库（与二进制同级）
Write-Host "[2/3] 复制配置与运行时数据..." -ForegroundColor Yellow
Copy-Item -Force (Join-Path $BackendDir "application.yml") (Join-Path $DistDir "application.yml")

if (Test-Path $DataDir) {
    $DistDataDir = Join-Path $DistDir "data"
    Remove-Item -Recurse -Force $DistDataDir -ErrorAction SilentlyContinue
    Copy-Item -Recurse -Force $DataDir $DistDataDir
    Write-Host "  [完成] 字库目录: $DistDataDir" -ForegroundColor Green
} else {
    Write-Host "  [错误] 未找到字库目录: $DataDir（服务会拒绝启动）" -ForegroundColor Red
    exit 1
}

Set-Location $RootDir

# 3. 生成启动脚本
Write-Host "[3/3] 生成启动脚本..." -ForegroundColor Yellow

$startScript = @"
@echo off
REM 起名 - SSR 全栈单文件启动（页面由后端模板渲染）
echo 正在启动起名...
"%~dp0$binaryName" -mode api -port $Port
"@
$startScript | Out-File -FilePath (Join-Path $DistDir "start.bat") -Encoding ASCII

$startPsScript = @"
# 起名 - SSR 全栈单文件启动
Write-Host "正在启动起名..." -ForegroundColor Cyan
& "`$PSScriptRoot\$binaryName" -mode api -port $Port
"@
$startPsScript | Out-File -FilePath (Join-Path $DistDir "start.ps1") -Encoding UTF8

Write-Host ""
Write-Host "=== 构建完成 ===" -ForegroundColor Cyan
Write-Host "输出目录: $DistDir"
Write-Host "二进制文件: $binaryName"
Write-Host "启动方式: .\$binaryName -mode api -port $Port"
Write-Host "访问地址: http://localhost:$Port"
