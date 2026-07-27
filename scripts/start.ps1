# 起名项目启动脚本
Write-Host "=== 起名项目启动 ===" -ForegroundColor Cyan

# 1. 检查并终止已占用8080端口的进程
$existingProcess = Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue
if ($existingProcess) {
    $pid = $existingProcess.OwningProcess | Select-Object -First 1
    Write-Host "发现8080端口被进程 $pid 占用，正在终止..." -ForegroundColor Yellow
    Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
}

# 2. 启动后端服务
Write-Host "`n[1/2] 启动后端服务..." -ForegroundColor Green
$backendJob = Start-Job -ScriptBlock {
    Set-Location "g:\19-Training\learngo\name\backend"
    go run cmd/server/main.go 2>&1
}
Write-Host "后端服务正在启动 (PID: $($backendJob.Id))..."

# 等待后端启动
Start-Sleep -Seconds 8

# 检查后端是否启动成功
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/healthz" -UseBasicParsing -TimeoutSec 5
    if ($response.StatusCode -eq 200) {
        Write-Host "后端服务启动成功! http://localhost:8080" -ForegroundColor Green
        Write-Host "健康检查: $($response.Content)" -ForegroundColor Green
    }
} catch {
    Write-Host "后端服务可能还在启动中，请稍候检查..." -ForegroundColor Yellow
}

# 3. 安装前端依赖并构建
Write-Host "`n[2/2] 准备前端..." -ForegroundColor Green
Set-Location "g:\19-Training\learngo\name\frontend"

# 检查 node_modules 是否存在
if (-not (Test-Path "node_modules")) {
    Write-Host "安装前端依赖..." -ForegroundColor Yellow
    pnpm install 2>&1
}

# 构建前端（因为后端 mode=all 需要静态文件）
Write-Host "构建前端静态文件..." -ForegroundColor Yellow
pnpm build 2>&1

Write-Host "`n=== 启动完成 ===" -ForegroundColor Cyan
Write-Host "后端API: http://localhost:8080/api/v1/" -ForegroundColor White
Write-Host "健康检查: http://localhost:8080/healthz" -ForegroundColor White
Write-Host "前端页面: http://localhost:8080/" -ForegroundColor White
