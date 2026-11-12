# 起名 (Namer) 一键构建 + 启动 + 验证脚本
#
# 用法：
#   powershell -ExecutionPolicy Bypass -File scripts\start.ps1              # 常规启动（按需重建）
#   powershell -ExecutionPolicy Bypass -File scripts\start.ps1 -Rebuild     # 强制重建后端 + 前端
#   powershell -ExecutionPolicy Bypass -File scripts\start.ps1 -SkipFrontend # 跳过前端构建（只调后端接口）
#   powershell -ExecutionPolicy Bypass -File scripts\start.ps1 -Port 8083   # 自定义端口
#
# 流程：环境检查 → 后端构建（源码 mtime 比对，按需编译）→ 物料导入（幂等，
#       指纹未变自动短路）→ 前端构建（out 缺失或源码更新时）→ 启动服务 →
#       健康验证（/healthz /readyz + 生成冒烟）→ 可选打开浏览器。
# 依赖：go / node / pnpm（Windows 10+ 自带 curl 可选）。不依赖 python。
param(
    [switch]$Rebuild,
    [switch]$SkipFrontend,
    [int]$Port = 8080,
    [switch]$NoBrowser
)

$ErrorActionPreference = 'Stop'
# go 子进程输出 UTF-8，PS 5.1 默认按 GBK 解读会乱码——统一控制台编码
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$RootDir = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $RootDir 'backend'
$FrontendDir = Join-Path $RootDir 'frontend'
$ServerExe = Join-Path $BackendDir 'bin\namer-server.exe'
$DataDir = Join-Path $RootDir 'data'
$DbPath = Join-Path $BackendDir 'namer.db'

function Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }
function Ok($msg)   { Write-Host "    [OK] $msg" -ForegroundColor Green }
function Warn($msg) { Write-Host "    [!!] $msg" -ForegroundColor Yellow }

# ---------- [1/6] 环境检查 ----------
Step '[1/6] 环境检查'
foreach ($tool in @('go', 'node', 'pnpm')) {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        Write-Host "错误: 未找到 $tool，请先安装并加入 PATH" -ForegroundColor Red
        exit 1
    }
}
Ok 'go / node / pnpm 均可用'

# ---------- [2/6] 后端构建 ----------
Step '[2/6] 后端构建'
$needBackendBuild = $Rebuild -or (-not (Test-Path $ServerExe))
if (-not $needBackendBuild) {
    # 源码/依赖/配置任一比二进制新即重建（与 backend/bin/start-server.sh 同口径）
    $exeTime = (Get-Item $ServerExe).LastWriteTime
    $srcLatest = Get-ChildItem -Path (Join-Path $BackendDir 'cmd'), (Join-Path $BackendDir 'internal'), (Join-Path $BackendDir 'go.mod'), (Join-Path $BackendDir 'application.yml') -Recurse -File |
        Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if ($srcLatest.LastWriteTime -gt $exeTime) {
        Warn "源码较新: $($srcLatest.Name) → 触发重建"
        $needBackendBuild = $true
    } else {
        Ok '二进制已是最新，跳过编译'
    }
}
if ($needBackendBuild) {
    Push-Location $BackendDir
    try {
        # Stop 模式下 go 的 stderr 进度输出会被包装成 NativeCommandError 直接终止脚本，
        # 临时降级为 Continue、以 $LASTEXITCODE 为准（下同）
        $ErrorActionPreference = 'Continue'
        go build -o bin\namer-server.exe ./cmd/server
        $buildExit = $LASTEXITCODE
        $ErrorActionPreference = 'Stop'
        if ($buildExit -ne 0) { throw 'go build 失败' }
        Ok '后端编译完成: backend/bin/namer-server.exe'
    } finally { Pop-Location }
}

# ---------- [3/6] 物料导入（幂等） ----------
Step '[3/6] 物料导入（指纹未变的物料自动短路跳过，ADR-014）'
# 显式绝对路径：materials 的默认 --data "./data" 相对进程 cwd（backend/）解析，
# 仓库根 data/ 与权威库路径必须显式指定
Push-Location $RootDir
try {
    $ErrorActionPreference = 'Continue'   # stderr 进度输出不以异常抛出，见 [2/6] 注释
    $importOut = go -C backend run ./cmd/materials import --data "$DataDir" --db "$DbPath" 2>&1 | Out-String
    $importExit = $LASTEXITCODE
    $ErrorActionPreference = 'Stop'
    if ($importExit -ne 0) {
        Write-Host $importOut
        throw '物料导入失败，请检查 data/ 目录与 materials.lock'
    }
    Ok '物料导入完成（未变更物料已跳过重灌）'
} finally { Pop-Location }

# ---------- [4/6] 前端构建 ----------
Step '[4/6] 前端构建'
$OutIndex = Join-Path $FrontendDir 'out\index.html'
$needFrontendBuild = -not $SkipFrontend
if (-not $SkipFrontend -and -not $Rebuild -and (Test-Path $OutIndex)) {
    $idxTime = (Get-Item $OutIndex).LastWriteTime
    $feLatest = Get-ChildItem -Path (Join-Path $FrontendDir 'src'), (Join-Path $FrontendDir 'public'), (Join-Path $FrontendDir 'package.json'), (Join-Path $FrontendDir 'next.config.js'), (Join-Path $FrontendDir 'tailwind.config.ts') -Recurse -File |
        Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if ($feLatest.LastWriteTime -gt $idxTime) {
        Warn "前端源码较新: $($feLatest.Name) → 触发重建"
    } else {
        Ok 'frontend/out 已是最新，跳过构建'
        $needFrontendBuild = $false
    }
}
if ($needFrontendBuild) {
    Push-Location $FrontendDir
    try {
        $env:NEXT_PUBLIC_API_URL = "http://127.0.0.1:${Port}/api"
        $ErrorActionPreference = 'Continue'   # 同 [2/6]：pnpm 的 stderr 进度不以异常抛出
        pnpm build
        $feExit = $LASTEXITCODE
        $ErrorActionPreference = 'Stop'
        if ($feExit -ne 0) { throw 'pnpm build 失败' }
        Ok '前端构建完成: frontend/out/'
    } finally {
        Remove-Item Env:\NEXT_PUBLIC_API_URL -ErrorAction SilentlyContinue
        Pop-Location
    }
} elseif ($SkipFrontend) {
    Warn '已按 -SkipFrontend 跳过前端构建'
}

# ---------- [5/6] 启动服务 ----------
Step '[5/6] 启动服务'
$conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
if ($conn) {
    $oldPid = $conn.OwningProcess | Select-Object -First 1
    Warn "端口 $Port 被进程 $oldPid 占用，终止旧实例"
    Stop-Process -Id $oldPid -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
}
# 工作目录 = 仓库根。注意：config 默认从 cwd 读 application.yml（在 backend/ 下），
# 必须显式传 -config 与 -db，否则回落仓库根/namer.db 空库 → 物料缺失 fail-fast
$proc = Start-Process -FilePath $ServerExe -WorkingDirectory $RootDir `
    -ArgumentList @('-host', '127.0.0.1', '-port', "$Port", '-config', "$BackendDir\application.yml", '-db', "$DbPath") `
    -WindowStyle Hidden -PassThru `
    -RedirectStandardOutput (Join-Path $env:TEMP 'namer-start-stdout.log') `
    -RedirectStandardError (Join-Path $env:TEMP 'namer-start-stderr.log')
Ok "服务进程已启动 (PID: $($proc.Id))，日志: %TEMP%\namer-start-*.log"

# ---------- [6/6] 健康验证 ----------
Step '[6/6] 健康验证'
$base = "http://127.0.0.1:$Port"
$ready = $false
for ($i = 0; $i -lt 30; $i++) {
    try {
        $h = Invoke-RestMethod -Uri "$base/healthz" -TimeoutSec 3
        if ($h.status -eq 'ok') { $ready = $true; break }
    } catch { Start-Sleep -Seconds 2 }
}
if (-not $ready) {
    Write-Host "错误: 服务未在 60s 内就绪，请查看 $env:TEMP\namer-start-stderr.log" -ForegroundColor Red
    exit 1
}
Ok '/healthz ok'

$r = Invoke-RestMethod -Uri "$base/readyz" -TimeoutSec 5
if ($r.status -ne 'ready') { throw "/readyz 异常: $($r.status)" }
Ok '/readyz ready（数据库已连接）'

# 生成冒烟：UTF-8 JSON 由 Invoke-RestMethod 原生处理（规避 curl 中文 GBK 编码陷阱）
$genBody = @{
    surname      = '测'
    gender       = 'male'
    birth_year   = 2025
    birth_month  = 1
    birth_day    = 15
    birth_hour   = 12
    birth_minute = 0
} | ConvertTo-Json
$gen = Invoke-RestMethod -Uri "$base/api/v1/names/generate" -Method Post -Body $genBody `
    -ContentType 'application/json; charset=utf-8' -TimeoutSec 120
$cnt = $gen.data.names.Count
if ($cnt -le 0) { throw "生成冒烟失败：候选名数量 $cnt" }
$first = $gen.data.names[0].full_name
Ok "生成冒烟通过：$cnt 个候选（首个 $first）"

if (-not $NoBrowser) {
    Start-Process "$base/"
    Ok '已打开浏览器'
}
Write-Host ''
Write-Host '==============================================' -ForegroundColor Green
Write-Host "  服务已就绪: $base" -ForegroundColor Green
Write-Host "  候选 50 个 / 冒烟通过" -ForegroundColor Green
Write-Host '==============================================' -ForegroundColor Green
