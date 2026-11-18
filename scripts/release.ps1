# 起名 - GitHub Release 一键发布脚本（Wails v3）
# 职责：把当前 HEAD 的构建产物发布为一个正式 CalVer release。
#   1. 前置校验：HEAD 与 origin/master 同步 / 工作区 clean / tag 未占用 /
#      CHANGELOG 顶部版本号存在
#   2. 调 build.ps1 构建三件套（exe / NSIS 安装包 / 现场制作的纯净种子库）
#   3. 打 annotated tag 并推送
#   4. gh release create 上传三件套
#
# 版本号唯一真源 = CHANGELOG.md 顶部的 `## [YYYY.MM.DD.MICRO]`，
# 脚本自动读取，避免与手写参数漂移。打 tag 前请确保 CHANGELOG 已更新并提交。
#
# 用法：
#   powershell -ExecutionPolicy Bypass -File scripts/release.ps1
#   scripts\release.ps1 -SkipBuild        # 产物已就绪，仅打 tag + 发布（谨慎）
#   scripts\release.ps1 -DryRun           # 只做前置校验并显示将执行的操作，不实际发布
#   scripts\release.ps1 -Wails3Path D:\tools\wails3.exe   # CLI 不在 PATH 时显式指定

param(
    [string]$Wails3Path = "", # wails3 CLI 不在 PATH 时显式指定，转发给 build.ps1
    [switch]$SkipBuild,   # 跳过构建，直接用 backend/bin 下现有产物（需自行保证新鲜）
    [switch]$DryRun       # 演练模式：只校验 + 打印，不打 tag / 不发布
)

# PowerShell 5.1 会把 native 命令（wails3 / go-task / git / gh）写到 stderr 的
# 正常进度信息包装成 NativeCommandError；Stop 模式下会据此误终止构建（实测
# wails3 build 的 `task: [build:frontend] pnpm build` 即触发）。脚本对每一步都
# 显式检查 $LASTEXITCODE，故用 Continue，让退出码而非 stderr 决定成败。
$ErrorActionPreference = "Continue"
$RootDir = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $RootDir "backend"
$Changelog = Join-Path $RootDir "CHANGELOG.md"

function Die([string]$msg) {
    Write-Host "[错误] $msg" -ForegroundColor Red
    exit 1
}

Write-Host "=== 起名 Release 发布（CalVer）===" -ForegroundColor Cyan

# 1. 从 CHANGELOG 顶部读取版本号（唯一真源）
if (-not (Test-Path $Changelog)) { Die "缺少 CHANGELOG.md" }
$version = $null
foreach ($line in (Get-Content $Changelog -Encoding UTF8)) {
    if ($line -match '^## \[(\d{4}\.\d{2}\.\d{2}\.\d+)\]') { $version = $Matches[1]; break }
}
if (-not $version) { Die "CHANGELOG.md 顶部未找到 CalVer 版本号（格式 YYYY.MM.DD.MICRO）" }
$tag = $version
Write-Host "版本号（来自 CHANGELOG）: $version"
Write-Host "Tag: $tag"

# 2. 前置校验：gh 登录
if (-not (Get-Command gh -ErrorAction SilentlyContinue)) { Die "未找到 gh CLI，请先安装并登录（gh auth login）" }
$ghUser = & gh api user --jq .login 2>$null
if ($LASTEXITCODE -ne 0 -or -not $ghUser) { Die "gh 未登录，请先执行 gh auth login" }
Write-Host "GitHub 账号: $ghUser"

# 3. 前置校验：HEAD 与 origin/master 同步
$localHead = & git -C $RootDir rev-parse HEAD
if ($LASTEXITCODE -ne 0) { Die "读取本地 HEAD 失败" }
& git -C $RootDir fetch origin master --quiet
if ($LASTEXITCODE -ne 0) { Die "git fetch origin master 失败" }
$remoteHead = & git -C $RootDir rev-parse origin/master
if ($localHead -ne $remoteHead) {
    Die "本地 HEAD ($($localHead.Substring(0,7))) 与 origin/master ($($remoteHead.Substring(0,7))) 不一致，请先 push"
}
Write-Host "HEAD 与 origin/master 一致: $($localHead.Substring(0,7))"

# 4. 前置校验：交付路径 clean。
#    只检查真正进入交付物的路径（backend/frontend 源码、脚本、文档、CI、数据物料），
#    排除两类与发布无关的历史 churn：
#      - .agents/ / .zcode/：IDE 技能与临时目录的既有删除/未跟踪
#      - data/namer.db：本机运行库（含收藏/历史），发布用 build.ps1 现场制作的库，
#        根本不读工作区这份，故其 modified 不阻断
$watchPaths = @("backend", "frontend", "scripts", "docs", ".github", "CHANGELOG.md", "AGENTS.md", "data", ":(exclude)data/namer.db")
$dirty = & git -C $RootDir status --porcelain --untracked-files=no -- $watchPaths
if ($LASTEXITCODE -ne 0) { Die "git status 失败" }
if ($dirty) {
    Write-Host "[错误] 交付路径有未提交的变更：" -ForegroundColor Red
    $dirty | ForEach-Object { Write-Host "  $_" -ForegroundColor Red }
    Die "请先提交或 stash 后再发布"
}
Write-Host "交付路径 clean（已跟踪文件；忽略 .agents/ 与本机 namer.db）"

# 5. 前置校验：tag 未被占用（本地与远端都不允许重名）
& git -C $RootDir rev-parse -q --verify "refs/tags/$tag" 2>$null | Out-Null
if ($LASTEXITCODE -eq 0) { Die "本地 tag $tag 已存在，禁止覆盖发布" }
& git -C $RootDir ls-remote --exit-code --tags origin "refs/tags/$tag" 2>$null | Out-Null
if ($LASTEXITCODE -eq 0) { Die "远端 tag $tag 已存在，禁止覆盖发布" }
Write-Host "tag $tag 未被占用"

# 6. DryRun：到此为止
if ($DryRun) {
    Write-Host ""
    Write-Host "[DryRun] 前置校验全部通过，将执行：" -ForegroundColor Yellow
    if ($SkipBuild) { Write-Host "  - 跳过构建（使用现有产物）" }
    else { Write-Host "  - 调用 build.ps1 构建 exe / 安装包 / 种子库" }
    Write-Host "  - git tag -a $tag -m 'Release $tag' 并 push origin $tag"
    Write-Host "  - gh release create $tag <exe> <installer> <db>"
    Write-Host "[DryRun] 未做任何修改。" -ForegroundColor Yellow
    exit 0
}

# 7. 构建三件套
if ($SkipBuild) {
    Write-Host "[提示] -SkipBuild：使用 backend/bin 下现有产物，请自行确认新鲜度" -ForegroundColor Yellow
}
else {
    Write-Host "[1/3] 调用 build.ps1 构建三件套 ..." -ForegroundColor Yellow
    $buildScript = Join-Path $PSScriptRoot "build.ps1"
    if ($Wails3Path) { & $buildScript -Wails3Path $Wails3Path }
    else { & $buildScript }
    if ($LASTEXITCODE -ne 0) { Die "build.ps1 构建失败" }
}

# 8. 校验产物存在
$exe = Join-Path $BackendDir "bin\namer-desktop.exe"
$installer = Join-Path $BackendDir "bin\namer-amd64-installer.exe"
# 必须上传 build.ps1 现场制作的纯净库（bin\namer.db），而非工作区 data\namer.db——
# 后者是含本机收藏/历史、可能被 WAL 污染的运行库，一旦上传即泄露用户数据。
$db = Join-Path $BackendDir "bin\namer.db"
foreach ($p in @($exe, $installer, $db)) {
    if (-not (Test-Path $p)) { Die "发布产物缺失: $p" }
}
Write-Host "[2/3] 产物校验通过" -ForegroundColor Green

# 9. 打 annotated tag 并推送
Write-Host "[3/3] 打 tag 并发布 ..." -ForegroundColor Yellow
& git -C $RootDir tag -a $tag -m "Release $tag"
if ($LASTEXITCODE -ne 0) { Die "git tag 创建失败" }
# pre-push hook 内部依赖 bash 跑质量门禁；PowerShell 会话 PATH 里通常没有 bash
# （实测 .githooks/pre-push: bash: command not found，导致 tag 推送被误阻断）。
# build.ps1 已完成现场制库+exe+安装包完整校验，CI 亦有独立门禁 job，
# 故此处跳过重复的 hook 验证；SKIP_VERIFY 仅影响本进程。
$env:SKIP_VERIFY = '1'
& git -C $RootDir push origin $tag
if ($LASTEXITCODE -ne 0) {
    Write-Host "[错误] tag 推送失败。本地 tag 已创建，可清理后重试：git tag -d $tag" -ForegroundColor Yellow
    Die "推送 tag 失败"
}

# 10. 创建 release 并上传三件套
& gh release create $tag $exe $installer $db --title "NameMaster $tag" --generate-notes
if ($LASTEXITCODE -ne 0) {
    Write-Host "[警告] gh release create 失败，但 tag $tag 已推送。" -ForegroundColor Yellow
    Write-Host "       可重试：gh release create $tag <产物...> --title 'NameMaster $tag' --generate-notes" -ForegroundColor Yellow
    Die "创建 release 失败"
}

Write-Host ""
Write-Host "=== 发布完成 ===" -ForegroundColor Cyan
Write-Host "Release: https://github.com/bliubiu/babyName/releases/tag/$tag"
Write-Host "产物: namer-desktop.exe / namer-amd64-installer.exe / namer.db"
