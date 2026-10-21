@echo off
REM ============================================================================
REM 一键启动 namer 后端服务（Windows 双击入口）
REM 委托 Git Bash 运行同目录的 start-server.sh（自动按需重建并启动）。
REM 若 PATH 中无 bash，则尝试常见 Git 安装路径。
REM ============================================================================
setlocal
set "SCRIPT_DIR=%~dp0"

where bash >nul 2>nul
if %ERRORLEVEL%==0 (
  bash "%SCRIPT_DIR%start-server.sh" %*
  goto :eof
)

set "GIT_BASH=C:\Program Files\Git\bin\bash.exe"
if exist "%GIT_BASH%" (
  "%GIT_BASH%" "%SCRIPT_DIR%start-server.sh" %*
  goto :eof
)

set "GIT_BASH2=C:\Program Files\Git\usr\bin\bash.exe"
if exist "%GIT_BASH2%" (
  "%GIT_BASH2%" "%SCRIPT_DIR%start-server.sh" %*
  goto :eof
)

echo 错误: 未找到 bash。请安装 Git for Windows 并将 bash 加入 PATH，
echo       或直接在 Git Bash 中运行: bash "%SCRIPT_DIR%start-server.sh"
exit /b 1
