@echo off
echo === 启动起名项目服务 ===
echo.

echo [1/2] 启动后端服务...
cd /d g:\19-Training\learngo\name\backend
start "NameServer" go run cmd/server/main.go

echo 等待服务启动...
timeout /t 5 /nobreak >nul

echo [2/2] 检查服务状态...
curl -s http://localhost:8080/healthz
if %errorlevel% equ 0 (
    echo.
    echo 服务启动成功！
    echo 访问地址: http://localhost:8080
) else (
    echo.
    echo 服务可能还在启动中，请稍后访问 http://localhost:8080
)

echo.
pause
