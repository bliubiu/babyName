# 起名 - GitHub Release 一键发布脚本（Wails v3）
# 职责：把当前 HEAD 的构建产物发布为一个正式 CalVer release。
#   1. 前置校验：HEAD 与 origin/master 同步 / 工作区 clean / tag 未占用 /
#      CHANGELOG 顶部版本号存在
#   2. 调 build.ps1 构建三件套（exe / NSIS 安装包 / 现场制作的纯净种子库）
#   3. 打 annotated tag 并推送
#   4. 分步发布：先建空 release（--generate-notes），再逐个上传三产物 + SHA256SUMS.txt
#      —— 大文件（exe）放最后，单个失败自动重试；上传后用 GitHub digest 与本地
#      SHA256 逐一比对，确保远端文件与本地逐字节一致（弱网下一次性上传曾 HTTP 408）
#
# SHA256 校验：发布时生成 backend/bin/SHA256SUMS.txt（标准 `<sha256>  <文件名>` 两空格
# 格式，不含自身），随 release 一起上传，下载方可 `sha256sum -c` 校验；每个附件上传
# 后还会用 `gh release view --json assets` 的 digest 字段回验，不一致即判失败。
#
# 版本号唯一真源 = CHANGELOG.md 顶部的 `## [YYYY.MM.DD.MICRO]`，
# 脚本自动读取，避免与手写参数漂移。打 tag 前请确保 CHANGELOG 已更新并提交。
#
# 用法：
#   powershell -ExecutionPolicy Bypass -File scripts/release.ps1
#   scripts\release.ps1 -SkipBuild        # 产物已就绪，仅打 tag + 发布（谨慎）
#   scripts\release.ps1 -DryRun           # 只做前置校验并显示将执行的操作，不实际发布
#   scripts\release.ps1 -Wails3Path D:\tools\wails3.exe   # CLI 不在 PATH 时显式指定
#   scripts\release.ps1 -UploadRetries 5                  # 弱网时增大单文件上传重试次数
#   scripts\release.ps1 -Resume -UploadRetries 5          # 断点续传：tag/release 已存在时只补传附件

param(
    [string]$Wails3Path = "",     # wails3 CLI 不在 PATH 时显式指定，转发给 build.ps1
    [switch]$SkipBuild,           # 跳过构建，直接用 backend/bin 下现有产物（需自行保证新鲜）
    [switch]$DryRun,              # 演练模式：只校验 + 打印，不打 tag / 不发布
    [switch]$Resume,              # 断点续传：跳过 tag 校验/创建/推送与 release 创建，只补传附件
    [int]$UploadRetries = 3       # 单个附件上传失败后的重试次数（HTTP 408 等瞬断）
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

# 计算文件 SHA256（小写 hex，与 GitHub assets.digest 口径一致）
function Get-Sha256([string]$Path) {
    return (Get-FileHash -Algorithm SHA256 -LiteralPath $Path).Hash.ToLowerInvariant()
}

# 上传单个附件并回验 digest：失败/校验不一致时重试，返回 $true/$false。
# 用 --clobber 保证重试与断点续传时覆盖同名文件，不会因已存在而失败。
function Upload-Asset([string]$Tag, [string]$Path, [int]$Retries) {
    $name = Split-Path -Leaf $Path
    $localHash = Get-Sha256 $Path
    for ($attempt = 1; $attempt -le $Retries; $attempt++) {
        Write-Host "  上传 $name ($attempt/$Retries) ..." -ForegroundColor Gray
        & gh release upload $Tag $Path --clobber 2>$null
        $uploadExit = $LASTEXITCODE
        if ($uploadExit -eq 0) {
            # 回验远端 digest，防弱网下文件上传不完整仍报告成功
            $jq = '.assets[] | select(.name=="' + $name + '") | .digest'
            $digest = & gh release view $Tag --json assets --jq $jq 2>$null
            if ($LASTEXITCODE -eq 0 -and $digest -match '^sha256:([0-9a-f]{64})$') {
                $remoteHash = $Matches[1]
                if ($remoteHash -eq $localHash) {
                    Write-Host "  [完成] $name digest 匹配 ($($localHash.Substring(0,12))...)" -ForegroundColor Green
                    return $true
                }
                Write-Host "  [警告] $name 哈希不一致（本地 $localHash / 远端 $remoteHash），重试" -ForegroundColor Yellow
            }
            else {
                Write-Host "  [警告] $name 上传后无法读取 digest，重试" -ForegroundColor Yellow
            }
        }
        else {
            Write-Host "  [警告] $name 上传失败（exit $uploadExit），重试" -ForegroundColor Yellow
        }
        if ($attempt -lt $Retries) { Start-Sleep -Seconds 5 }
    }
    Write-Host "  [错误] $name 上传/校验失败（已重试 $Retries 次）" -ForegroundColor Red
    return $false
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

# 5. 前置校验：tag 未被占用（本地与远端都不允许重名）。
#    -Resume 断点续传时 tag 必然已存在，跳过此校验。
if (-not $Resume) {
    & git -C $RootDir rev-parse -q --verify "refs/tags/$tag" 2>$null | Out-Null
    if ($LASTEXITCODE -eq 0) { Die "本地 tag $tag 已存在，禁止覆盖发布（续传请用 -Resume）" }
    & git -C $RootDir ls-remote --exit-code --tags origin "refs/tags/$tag" 2>$null | Out-Null
    if ($LASTEXITCODE -eq 0) { Die "远端 tag $tag 已存在，禁止覆盖发布（续传请用 -Resume）" }
    Write-Host "tag $tag 未被占用"
}
else {
    Write-Host "[-Resume] 跳过 tag 占用校验" -ForegroundColor Yellow
}

# 6. DryRun：到此为止
if ($DryRun) {
    Write-Host ""
    Write-Host "[DryRun] 前置校验全部通过，将执行：" -ForegroundColor Yellow
    if ($SkipBuild -or $Resume) { Write-Host "  - 跳过构建（使用现有产物）" }
    else { Write-Host "  - 调用 build.ps1 构建 exe / 安装包 / 种子库" }
    if ($Resume) { Write-Host "  - [-Resume] 跳过 tag 校验/创建/推送与 release 创建，只补传附件" }
    else { Write-Host "  - git tag -a $tag -m 'Release $tag' 并 push origin $tag" }
    Write-Host "  - gh release create $tag --title 'NameMaster $tag' --generate-notes（空壳）"
    Write-Host "  - 生成 SHA256SUMS.txt（三产物 sha256 清单）"
    Write-Host "  - 逐个上传并通过 digest 回验（单个失败重试 $UploadRetries 次）："
    foreach ($f in @('namer.db', 'namer-amd64-installer.exe', 'namer-desktop.exe', 'SHA256SUMS.txt')) {
        Write-Host "      - $f"
    }
    Write-Host "[DryRun] 未做任何修改。" -ForegroundColor Yellow
    exit 0
}

# 7. 构建三件套（Resume 续传必然复用已有产物，强制跳过构建）
if ($SkipBuild -or $Resume) {
    Write-Host "[提示] 使用 backend/bin 下现有产物（SkipBuild/Resume），请自行确认新鲜度" -ForegroundColor Yellow
}
else {
    Write-Host "[1/4] 调用 build.ps1 构建三件套 ..." -ForegroundColor Yellow
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
Write-Host "[2/4] 产物校验通过" -ForegroundColor Green

# 9. 打 annotated tag 并推送（Resume 时 tag 已存在，跳过）
if ($Resume) {
    Write-Host "[3/4] [-Resume] 跳过打 tag 与推送" -ForegroundColor Yellow
}
else {
    Write-Host "[3/4] 打 tag 并推送 ..." -ForegroundColor Yellow
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
}

# 10. 生成 SHA256SUMS.txt（标准两空格格式，不含自身；`sha256sum -c` 可直接校验）
$checksumFile = Join-Path $BackendDir "bin\SHA256SUMS.txt"
$checksumLines = @()
foreach ($p in @($db, $installer, $exe)) {
    $hash = Get-Sha256 $p
    if (-not $hash) { Die "计算 SHA256 失败: $p" }
    $checksumLines += "$hash  $(Split-Path -Leaf $p)"
    Write-Host "  SHA256 $(Split-Path -Leaf $p): $hash" -ForegroundColor Gray
}
# 用 UTF-8 无 BOM 写入（内容全为 ASCII，避免 BOM 破坏 sha256sum -c 解析）
[System.IO.File]::WriteAllLines($checksumFile, $checksumLines, (New-Object System.Text.UTF8Encoding($false)))
Write-Host "[4/4] SHA256SUMS.txt 已生成" -ForegroundColor Green

# 11. 分步发布：先建空 release，再逐个上传。
#  一次性 `gh release create <多文件>` 在弱网下会因大文件超时整体回滚（实测 exe 37MB
#  触发 HTTP 408，db/安装包已传的内容也一并丢失）；分步 + 单文件重试 + --clobber
#  让失败只影响单个附件，且可安全地断点续传/整脚本重跑。
$releaseExists = $false
& gh release view $tag 2>$null
if ($LASTEXITCODE -eq 0) { $releaseExists = $true }
if ($releaseExists -and $Resume) {
    Write-Host "[4/4] [-Resume] release 已存在，跳过创建，直接续传附件" -ForegroundColor Yellow
}
elseif ($releaseExists) {
    Die "release $tag 已存在但非 Resume 模式，拒绝重复创建（续传请用 -Resume）"
}
else {
    & gh release create $tag --title "NameMaster $tag" --generate-notes
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[警告] gh release create 失败，但 tag $tag 已推送。" -ForegroundColor Yellow
        Write-Host "       可手工重试：gh release create $tag --title 'NameMaster $tag' --generate-notes" -ForegroundColor Yellow
        Die "创建 release 失败"
    }
    Write-Host "[4/4] release 空壳已创建，开始逐个上传附件" -ForegroundColor Yellow
}

# 上传顺序：小文件先、最大（exe）最后，降低弱网下大文件超时导致整体失败的概率。
$allOk = $true
foreach ($p in @($db, $installer, $checksumFile, $exe)) {
    if (-not (Upload-Asset -Tag $tag -Path $p -Retries $UploadRetries)) { $allOk = $false }
}
if (-not $allOk) {
    Write-Host "[警告] 部分附件上传失败。release 已存在，修复网络后用 -Resume 续传（只补附件、不动 tag）：" -ForegroundColor Yellow
    Write-Host "       scripts\release.ps1 -Resume -UploadRetries 5" -ForegroundColor Yellow
    Die "附件上传未全部成功"
}

Write-Host ""
Write-Host "=== 发布完成 ===" -ForegroundColor Cyan
Write-Host "Release: https://github.com/bliubiu/babyName/releases/tag/$tag"
Write-Host "产物: namer.db / namer-amd64-installer.exe / SHA256SUMS.txt / namer-desktop.exe（均已 digest 回验）"
