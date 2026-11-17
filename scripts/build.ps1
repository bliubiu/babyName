# 起名 - 一键桌面端构建脚本（Wails v3）
# 产物：
#   backend/bin/namer-desktop.exe              单文件桌面 exe
#   backend/bin/namer-amd64-installer.exe     NSIS 安装器（user 作用域 + 种子库随装）
#
# 真正的构建逻辑在 backend/Taskfile.yml（wails3 build / wails3 package）：
# 本脚本只负责 CLI 定位、版本护栏、种子库校验与前置检查，避免两套构建逻辑漂移。
# 前端产物由 pnpm build 末尾的 scripts/sync-dist.mjs 同步进 Go embed 目录。
#
# 种子库策略（ADR-010「A + 单库」）：
#   NSIS 安装器把 data/namer.db 原样随装给新用户，故打包时它必须是纯净种子库
#   （单文件 / journal_mode=delete / 用户表为空 / 卦象对齐金标准）。开发机跑过
#   app 后该库会带本机收藏与历史，并可能因 WAL 模式留下 -wal/-shm 残留——直接
#   打包既泄露本机数据又产出不符合分发态的安装包。故构建期临时以 git 入库的
#   纯净版本替换，--verify-only 校验通过后构建，最后无论成败都恢复开发机原库。
#   ※ 构建前请先关闭正在运行的 namer-desktop.exe，避免恢复后写回内存态数据。

param(
    [string]$Wails3Path = "",   # 未加入 PATH 时显式指定 wails3.exe 路径
    [switch]$SkipInstaller      # 只出裸 exe，跳过 NSIS 安装器构建
)

$RootDir = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $RootDir "backend"
$SeedDb = Join-Path $RootDir "data\namer.db"

# 恢复开发机数据库：主库 + 可能的 -wal/-shm sidecar 一起还原。
# 定义在顶层，供校验失败分支与 finally 共用，避免恢复逻辑分叉。
function Restore-DevDb {
    param([string]$BackupDir, [string]$Target)
    $backupMain = Join-Path $BackupDir "namer.db"
    if (-not (Test-Path $backupMain)) { return }
    Copy-Item $backupMain $Target -Force
    foreach ($sfx in @("-wal", "-shm")) {
        $src = Join-Path $BackupDir "namer.db$sfx"
        $dst = "$Target$sfx"
        if (Test-Path $src) { Copy-Item $src $dst -Force }
        elseif (Test-Path $dst) { Remove-Item $dst -Force }
    }
}

# 1. 定位 wails3 CLI：参数 > PATH > %USERPROFILE%\go\bin\wails3.exe
if ($Wails3Path) {
    $Wails3Exe = $Wails3Path
}
elseif (Get-Command wails3 -ErrorAction SilentlyContinue) {
    $Wails3Exe = "wails3"
}
else {
    $fallback = Join-Path $env:USERPROFILE "go\bin\wails3.exe"
    if (Test-Path $fallback) { $Wails3Exe = $fallback }
    else {
        Write-Host "[错误] 未找到 wails3 CLI，请安装：go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0" -ForegroundColor Red
        Write-Host "       或用 -Wails3Path <路径> 显式指定。" -ForegroundColor Red
        exit 1
    }
}

# 2. 版本护栏：Go module / CLI / @wailsio/runtime 必须精确 pin 同一版本
#    （v3 GA 后是 nightly + 每周次版本，三者错位会导致绑定 ID 漂移，docs/48 §2）
$goMod = Get-Content (Join-Path $BackendDir "go.mod") -Raw
if ($goMod -notmatch 'github\.com/wailsapp/wails/v3 v3\.0\.0') {
    Write-Host "[错误] backend/go.mod 未锁定 wails/v3 v3.0.0，终止构建。" -ForegroundColor Red
    exit 1
}
$pkgJson = Get-Content (Join-Path $RootDir "frontend\package.json") -Raw
if ($pkgJson -notmatch '"@wailsio/runtime": "3\.0\.0"') {
    Write-Host "[错误] frontend/package.json 未锁定 @wailsio/runtime 3.0.0，终止构建。" -ForegroundColor Red
    exit 1
}

# 3. makensis 只认 PATH；缺失时仅跳过安装器，不影响 exe 构建
$HasMakensis = $false
if (-not $SkipInstaller) {
    if (Get-Command makensis -ErrorAction SilentlyContinue) { $HasMakensis = $true }
    else {
        Write-Host "[提示] 未找到 makensis（NSIS），跳过安装器构建。" -ForegroundColor Yellow
        Write-Host "       安装后重跑本脚本即可出安装器：winget install NSIS.NSIS" -ForegroundColor Yellow
    }
}

Write-Host "=== 起名 桌面端构建（Wails v3.0.0） ===" -ForegroundColor Cyan
Write-Host "CLI: $Wails3Exe"
Write-Host ""

# 4. 种子库准备与校验（与 CI quality.yml 的 build_seed --verify-only 同口径）
$SeedBackup = Join-Path $env:TEMP ("namer-seed-backup-" + (Get-Date -Format "yyyyMMdd-HHmmss"))
$seedSwapped = $false

Write-Host "[1/3] 准备并校验种子库（ADR-010 单库）..." -ForegroundColor Yellow

# 4.1 备份开发机当前运行库（可能含本机收藏/历史），失败则中止——没有备份就绝不替换
if (Test-Path $SeedDb) {
    New-Item -ItemType Directory -Path $SeedBackup -Force | Out-Null
    Copy-Item $SeedDb (Join-Path $SeedBackup "namer.db") -Force
    foreach ($sfx in @("-wal", "-shm")) {
        if (Test-Path "$SeedDb$sfx") { Copy-Item "$SeedDb$sfx" (Join-Path $SeedBackup "namer.db$sfx") -Force }
    }
    if (-not (Test-Path (Join-Path $SeedBackup "namer.db"))) {
        Write-Host "[错误] 备份种子库失败，终止构建（未对原库做任何修改）。" -ForegroundColor Red
        exit 1
    }
    Write-Host "  开发机库已备份: $SeedBackup" -ForegroundColor Gray
}

# 4.2 用 git 入库的纯净版本替换（保证打包不含本机数据）
git -C $RootDir cat-file -e "HEAD:data/namer.db" 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "[错误] git HEAD 中缺少 data/namer.db，无法取得纯净种子库。" -ForegroundColor Red
    Restore-DevDb -BackupDir $SeedBackup -Target $SeedDb
    exit 1
}
git -C $RootDir checkout HEAD -- data/namer.db
if ($LASTEXITCODE -ne 0) {
    Write-Host "[错误] 恢复纯净种子库失败。" -ForegroundColor Red
    Restore-DevDb -BackupDir $SeedBackup -Target $SeedDb
    exit 1
}
$seedSwapped = $true
foreach ($sfx in @("-wal", "-shm")) {
    if (Test-Path "$SeedDb$sfx") { Remove-Item "$SeedDb$sfx" -Force }
}

# 4.3 校验纯净库；失败即中止并恢复开发机库
Push-Location $BackendDir
& go run ./tools/build_seed --verify-only --out $SeedDb
$verifyExit = $LASTEXITCODE
Pop-Location
if ($verifyExit -ne 0) {
    Write-Host "[错误] 种子库校验失败，终止构建（详见上方报告）。" -ForegroundColor Red
    Restore-DevDb -BackupDir $SeedBackup -Target $SeedDb
    Write-Host "  开发机库已恢复。" -ForegroundColor Gray
    exit 1
}
Write-Host "  [完成] 种子库校验通过" -ForegroundColor Green

Push-Location $BackendDir
try {
    # wails3 build = Taskfile 的 build 任务：pnpm build(sync-dist) + go build -tags production
    Write-Host "[2/3] 构建桌面 exe ..." -ForegroundColor Yellow
    & $Wails3Exe build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [错误] wails3 build 失败" -ForegroundColor Red
        exit 1
    }
    Write-Host "  [完成] backend\bin\namer-desktop.exe" -ForegroundColor Green

    if ($HasMakensis) {
        # wails3 package = Taskfile 的 package 任务：生成 WebView2 bootstrapper + makensis
        # 传入 WAILS3 变量，让子任务用同一个 CLI 可执行文件（CLI 可能不在 PATH）。
        Write-Host "[3/3] 构建 NSIS 安装器（user 作用域 + 种子库随装）..." -ForegroundColor Yellow
        & $Wails3Exe package "WAILS3=$Wails3Exe"
        if ($LASTEXITCODE -ne 0) {
            Write-Host "  [错误] wails3 package 失败" -ForegroundColor Red
            exit 1
        }
        Write-Host "  [完成] backend\bin\namer-amd64-installer.exe" -ForegroundColor Green
    }
    else {
        Write-Host "[3/3] 跳过 NSIS 安装器（-SkipInstaller 或未检测到 makensis）" -ForegroundColor Gray
    }
}
finally {
    Pop-Location
    # 无论构建成败都恢复开发机运行库，绝不把本机数据留在打包态、也绝不丢失它
    if ($seedSwapped) {
        Restore-DevDb -BackupDir $SeedBackup -Target $SeedDb
        Write-Host "[提示] 开发机数据库已恢复（含本机收藏/历史）。" -ForegroundColor Gray
    }
}

Write-Host ""
Write-Host "=== 构建完成 ===" -ForegroundColor Cyan
Write-Host "产物: $(Join-Path $BackendDir 'bin\namer-desktop.exe')"
if ($HasMakensis) {
    Write-Host "安装器: $(Join-Path $BackendDir 'bin\namer-amd64-installer.exe')"
}
Write-Host "启动方式: 双击 exe，或在 backend 目录运行 .\bin\namer-desktop.exe"
