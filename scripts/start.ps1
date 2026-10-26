# 起名项目启动脚本（SSR 全栈：Go 服务端渲染，无前端构建）
param(
    [string]$Port = "8080"
)

$RootDir = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $RootDir "backend"

Write-Host "=== 起名项目启动 ===" -ForegroundColor Cyan

# 1. 释放端口
$existing = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue
if ($existing) {
    $targetPid = $existing.OwningProcess | Select-Object -First 1
    Write-Host "端口 $Port 被进程 $targetPid 占用，正在终止..." -ForegroundColor Yellow
    Stop-Process -Id $targetPid -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
}

# 2. 启动服务（SSR 页面与 API 同源）
Write-Host "启动后端（SSR + API）..." -ForegroundColor Green
$job = Start-Job -ScriptBlock {
    param($dir, $port)
    Set-Location $dir
    go run ./cmd/server -port $port -mode api 2>&1
} -ArgumentList $BackendDir, $Port

Write-Host "进程已提交 (Job PID: $($job.Id))，等待就绪..."

$ready = $false
for ($i = 1; $i -le 30; $i++) {
    Start-Sleep -Seconds 1
    try {
        $response = Invoke-WebRequest -Uri "http://localhost:$Port/healthz" -UseBasicParsing -TimeoutSec 3
        if ($response.StatusCode -eq 200) {
            $ready = $true
            Write-Host "服务已就绪: http://localhost:$Port" -ForegroundColor Green
            Write-Host "健康检查: $($response.Content)" -ForegroundColor Green
            break
        }
    } catch {
        # 继续等待
    }
}

if (-not $ready) {
    Write-Host "服务未在 30 秒内就绪，查看日志：" -ForegroundColor Red
    Receive-Job -Job $job -Keep
    exit 1
}

Write-Host ""
Write-Host "首页:   http://localhost:$Port/" -ForegroundColor White
Write-Host "API:    http://localhost:$Port/api/v1/" -ForegroundColor White
Write-Host "探活:   http://localhost:$Port/healthz" -ForegroundColor White
Write-Host "就绪:   http://localhost:$Port/readyz" -ForegroundColor White
Write-Host ""
Write-Host "停止服务: Stop-Job -Id $($job.Id); Remove-Job -Id $($job.Id)" -ForegroundColor Gray
