@echo off
rem 起名 (Namer) 一键构建 + 启动 + 验证（薄壳入口，双击即可）
rem 逻辑本体见同目录 start.ps1（不依赖 python）
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0start.ps1" %*
pause
