# 启动后端服务
Write-Host "启动后端服务..." -ForegroundColor Green
Start-Process -FilePath "go" -ArgumentList "run","cmd/server/main.go" -WorkingDirectory "g:\19-Training\learngo\name\backend" -WindowStyle Normal

# 等待服务启动
Write-Host "等待服务启动..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

# 检查服务状态
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/healthz" -UseBasicParsing -TimeoutSec 5
    Write-Host "后端服务启动成功！" -ForegroundColor Green
    Write-Host "健康检查: $($response.Content)" -ForegroundColor Green
} catch {
    Write-Host "后端服务启动失败或仍在启动中..." -ForegroundColor Red
}

# 检查前端静态文件
if (Test-Path "g:\19-Training\learngo\name\frontend\out\index.html") {
    Write-Host "前端静态文件已存在" -ForegroundColor Green
} else {
    Write-Host "前端静态文件不存在，需要构建..." -ForegroundColor Yellow
    Set-Location "g:\19-Training\learngo\name\frontend"
    pnpm build
}

Write-Host "`n服务地址: http://localhost:8080" -ForegroundColor Cyan
