# 起名 - 一键构建脚本
# 构建单文件可执行二进制（前端静态文件内嵌在二进制中）

param(
    [string]$Mode = "all",        # all | backend
    [string]$Platform = "windows", # windows | linux | darwin
    [string]$Arch = "amd64",      # amd64 | arm64
    [string]$Port = "8080",
    [string]$OutDir = "./dist"
)

$RootDir = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $RootDir "backend"
$FrontendDir = Join-Path $RootDir "frontend"
$DistDir = Join-Path $RootDir $OutDir

Write-Host "=== 起名 构建脚本 ===" -ForegroundColor Cyan
Write-Host "模式: $Mode"
Write-Host "平台: $Platform/$Arch"
Write-Host "输出目录: $DistDir"
Write-Host ""

# 确保输出目录存在
New-Item -ItemType Directory -Force -Path $DistDir | Out-Null

if ($Mode -eq "all" -or $Mode -eq "frontend") {
    # 构建前端
    Write-Host "[1/4] 构建前端静态文件..." -ForegroundColor Yellow
    Set-Location $FrontendDir
    
    # 检查 node_modules
    if (-not (Test-Path "node_modules")) {
        Write-Host "  -> 安装 pnpm 依赖..." -ForegroundColor Gray
        pnpm install
        if ($LASTEXITCODE -ne 0) {
            Write-Host "  [错误] pnpm install 失败" -ForegroundColor Red
            exit 1
        }
    }
    
    # 执行构建
    pnpm build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [错误] 前端构建失败" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "  [完成] 前端构建成功: $FrontendDir\out" -ForegroundColor Green
    
    # 复制前端文件到 Go embed 目录（编译进二进制）
    Write-Host "  -> 复制前端文件到 embed 目录..." -ForegroundColor Gray
    $EmbedDir = Join-Path $BackendDir "internal\infrastructure\server\frontend"
    Remove-Item -Recurse -Force "$EmbedDir\*" -ErrorAction SilentlyContinue
    Copy-Item -Recurse -Force "out\*" $EmbedDir
    
    # 复制静态文件到 dist（兼容旧部署方式）
    if (Test-Path "out") {
        Copy-Item -Recurse -Force "out\*" (Join-Path $DistDir "frontend")
    }
    
    Set-Location $RootDir
}

if ($Mode -eq "all" -or $Mode -eq "backend") {
    # 编译后端
    Write-Host "[2/5] 编译后端二进制..." -ForegroundColor Yellow
    Set-Location $BackendDir
    
    # 设置编译参数
    $env:GOOS = $Platform
    $env:GOARCH = $Arch
    
    $binaryName = "namer"
    if ($Platform -eq "windows") {
        $binaryName += ".exe"
    }
    
    # 下载依赖
    Write-Host "  -> 下载 Go 依赖..." -ForegroundColor Gray
    go mod tidy
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [警告] go mod tidy 有问题，继续编译..." -ForegroundColor Yellow
    }
    
    # 编译
    Write-Host "  -> 编译 $binaryName ..." -ForegroundColor Gray
    go build -o (Join-Path $DistDir $binaryName) -ldflags="-s -w" ./cmd/server/
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [错误] 后端编译失败" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "  [完成] 后端编译成功: $DistDir\$binaryName" -ForegroundColor Green
    Set-Location $RootDir
}

# 复制运行时数据目录（与二进制同级，main.go 从 cwd/data 加载）
# 包含 namer.json（8105字）、name_frequency.json（重名率）、naming_quality.json（门禁字表）等
if ($Mode -eq "all" -or $Mode -eq "backend") {
    Write-Host "[3/5] 复制运行时数据目录..." -ForegroundColor Yellow
    $DataDir = Join-Path $BackendDir "data"
    $DistDataDir = Join-Path $DistDir "data"
    if (Test-Path $DataDir) {
        Remove-Item -Recurse -Force $DistDataDir -ErrorAction SilentlyContinue
        Copy-Item -Recurse -Force $DataDir $DistDataDir
        Write-Host "  [完成] 数据目录已复制到: $DistDataDir" -ForegroundColor Green
    } else {
        Write-Host "  [警告] 未找到数据目录: $DataDir" -ForegroundColor Red
    }
}

# 生成启动脚本
Write-Host "[4/5] 生成启动脚本..." -ForegroundColor Yellow

if ($Mode -eq "all") {
    $startScript = @"
@echo off
REM 起名 - 单文件启动
REM 使用内置前端静态文件
echo 正在启动起名...
"%%~dp0namer.exe" -mode all -port %Port%
"@
    $startScript | Out-File -FilePath (Join-Path $DistDir "start.bat") -Encoding ASCII
    
    # 同时生成 PowerShell 版本
    $startPsScript = @"
# 起名 - 单文件启动
Write-Host "正在启动起名..." -ForegroundColor Cyan
.\namer.exe -mode all -port $Port
"@
    $startPsScript | Out-File -FilePath (Join-Path $DistDir "start.ps1") -Encoding UTF8
}

Write-Host ""
Write-Host "=== 构建完成 ===" -ForegroundColor Cyan
Write-Host "输出目录: $DistDir"
Write-Host "二进制文件: namer.exe"
Write-Host "启动方式: .\namer.exe -mode all -port $Port"
Write-Host "访问地址: http://localhost:$Port"
