@echo off
REM ============================================================================
REM One-click launcher for the namer backend server (Windows double-click entry).
REM
REM Why cd first: we change into the script directory and then call bash with a
REM RELATIVE script name. Passing a Windows backslash path (e.g.
REM C:\Users\...\start-server.sh) straight to bash turns the backslashes into
REM escape characters, so bash cannot find the script and the console flashes
REM and closes instantly. A relative name has no backslashes, so it works.
REM
REM The script auto-rebuilds BOTH the Go backend binary and the Next.js
REM frontend (frontend/out) when they are missing or out of date, then starts
REM the server in the foreground.
REM
REM Extra args are forwarded to the shell script, e.g.:
REM   start-server.bat --rebuild        force rebuild of backend + frontend
REM   start-server.bat --skip-frontend  skip the frontend build
REM   start-server.bat --check          build only, do not start the server
REM ============================================================================
setlocal EnableExtensions

REM Use UTF-8 codepage so the server's Chinese logs render correctly.
chcp 65001 >nul 2>nul

REM Change to the script directory (trailing dot avoids the "\"+quote quirk).
cd /d "%~dp0."

REM 1) derive bash from git location first (most reliable for exec'ing the .exe)
for /f "delims=" %%G in ('where git 2^>nul') do (
  if exist "%%~dpG..\bin\bash.exe" (
    "%%~dpG..\bin\bash.exe" start-server.sh %*
    goto :eof
  )
  if exist "%%~dpG..\usr\bin\bash.exe" (
    "%%~dpG..\usr\bin\bash.exe" start-server.sh %*
    goto :eof
  )
)

REM 2) bash on PATH
where bash >nul 2>nul
if %ERRORLEVEL%==0 (
  bash start-server.sh %*
  goto :eof
)

REM 3) common install locations
for %%P in (
  "C:\Program Files\Git\bin\bash.exe"
  "C:\Program Files\Git\usr\bin\bash.exe"
  "C:\Program Files (x86)\Git\bin\bash.exe"
) do (
  if exist %%P (
    %%P start-server.sh %*
    goto :eof
  )
)

echo.
echo ERROR: bash not found. Install Git for Windows and add bash to PATH,
echo        or run in Git Bash: bash "%~dp0start-server.sh"
echo.
pause
exit /b 1
