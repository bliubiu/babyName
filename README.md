# 🍼 起名 (Name)

***

## 📖 项目简介

起名是一个免费的**桌面端起名工具**，帮新手父母为宝宝取一个寓意美好、符合中国传统命理与文字学的名字。

应用基于 **Wails v3** 构建，是一个 Windows 单文件桌面程序：命理排盘、字库检索、名字生成、收藏与历史全部在**本机**完成——不联网、不上传生辰数据，唯一数据源是本地 SQLite 库。

**核心功能：**

- 🔮 **八字五行分析** - 四柱排盘、五行旺衰统计、日主强弱判定、喜用神/忌神推导
- 🧘 **纳音五行** - 六十甲子纳音查询，分析名字与年命关系
- 🐭 **生肖属相** - 12 生肖喜忌、相合相冲
- 📜 **易经 64 卦** - 按姓名笔画起卦，解读卦辞与象辞
- ⭐ **九维智能评分** - 100 分制综合评分，逐维明细透传（`score_detail`）
- 🎯 **测名与探索** - 评测已有名字，或按五行偏好探索候选字
- ❤️ **名字收藏** - 收藏喜欢的名字，本地持久化
- ⚖️ **名字对比** - 多名字横向对比评分、五行、笔画等指标
- 📖 **诗词典故** - 名字关联《诗经》等典籍出典
- 🔍 **重名率查询** - 基于真实人名频率语料的重名率统计
- 📅 **黄历万年历** - 农历、节气、宜忌查询
- 📄 **PDF 报告** - 导出结构化起名报告（内置 NotoSansSC 字体），并支持分享快照

***

## 🏗️ 技术架构

### 技术栈

| 层级       | 技术                                                        |
| ---------- | ----------------------------------------------------------- |
| 桌面框架   | Wails v3.0.0（Windows WebView2）                            |
| 前端       | Vite 7 + React 19.2 + TypeScript 6.0                        |
| 样式       | Tailwind CSS 3.4                                            |
| 状态管理   | Zustand / TanStack Query / React Router 7                   |
| 后端       | Go 1.26，纯 Go 实现（`CGO_ENABLED=0`）                      |
| 数据持久化 | SQLite（`modernc.org/sqlite`，纯 Go，WAL 模式）             |
| 架构       | DDD（领域驱动设计）+ TDD（测试驱动开发）                     |
| 包管理器   | pnpm                                                        |

前端不再通过 HTTP 调用后端：Wails 在启动时生成 `frontend/bindings`，React 直接调用 Go 绑定方法（`internal/desktop`），错误以中文文案返回。

### 三层体系

```
        ┌─────────────────────────┐
        │  字象（血肉）            │
        │  选什么字，形音义配合     │
        │  ├ 部首五行判定（字形法） │
        │  ├ 字义五行纠正（字义法） │
        │  ├ 音韵流畅度            │
        │  └ 诗词典故出典          │
        ├─────────────────────────┤
        │  数理（骨架）            │
        │  笔画搭配，卦象数理       │
        ├─────────────────────────┤
        │  命理（根基）            │
        │  八字分析，喜用神定调     │
        └─────────────────────────┘
```

运作流程：八字排盘定喜用神 → 据此确定数理配置 → 在对应五行部首中选字 → 校正字义、搭配音韵、检查出典 → 校验姓名整体五行生克。

### 目录结构

```
name/
├── backend/                         # Go 后端（Wails v3 应用宿主）
│   ├── main.go                      # 入口：目录解析 → 装配 → 服务/窗口
│   ├── Taskfile.yml                 # wails3 build / package 任务
│   ├── build/                       # v3 产品元信息 + Windows NSIS 安装脚本
│   ├── frontend/dist/               # Vite 产物（//go:embed 目录，构建时同步）
│   ├── assets/                      # 内置字体（PDF 报告用 NotoSansSC 等）
│   ├── internal/
│   │   ├── desktop/                 # 绑定层：前端直接调用（替代原 HTTP handler）
│   │   ├── bootstrap/               # 装配层：存储 + 物料播种 + 服务装配
│   │   ├── application/             # 应用层：服务编排
│   │   ├── domain/                  # 领域层：bazi / yijing / ziwei / name / fate …
│   │   ├── infrastructure/          # 基础设施：SQLite 等
│   │   └── materials/               # 物料管线
│   └── cmd/                         # CLI 工具：namer-cli / materials / build_frequency …
│
├── frontend/                        # Vite + React 前端
│   ├── src/app/                     # 页面：结果 / 测名 / 对比 / 收藏 / 历史 / 黄历 / 分享
│   ├── src/components/              # 组件
│   ├── bindings/                    # Wails 生成的 TypeScript 绑定
│   └── scripts/sync-dist.mjs        # 产物同步进 Go embed 目录
│
├── data/                            # 物料源 + 唯一交付真库 namer.db
├── docs/                            # 按序号命名的中文文档（00–48）
└── scripts/                         # build / release / 质量门禁脚本
```

***

## 🚀 快速开始

### 方式一：安装包（推荐）

从 [Releases](https://github.com/bliubiu/babyName/releases/latest) 下载 `namer-amd64-installer.exe` 双击安装：

- 安装到当前用户作用域（无需管理员权限）
- 物料与纯净种子库随装，开箱即用
- 卸载时**保留**个人收藏/历史库

### 方式二：免安装绿色版

下载 `namer-desktop.exe` 与 `namer.db`，按如下结构放置后直接运行：

```
namer-desktop.exe
data\
  namer.db        # 唯一交付真库，首次启动即用，无需导入 JSON
logs\             # 首次运行自动创建
```

> 运行期只认 SQLite，不走 JSON。若 `data\namer.db` 缺失，首次启动会从物料源播种；**不要**用开发时含本机收藏的库替代纯净种子库。

### 方式三：源码构建

```powershell
# 项目根目录执行（需 wails3 CLI、Go 1.26、pnpm、makensis）
.\scripts\build.ps1

# 只出 exe，不出安装包
.\scripts\build.ps1 -SkipInstaller
```

产物统一落在 `backend/bin/`：

| 文件                          | 说明                              |
| ----------------------------- | --------------------------------- |
| `namer-desktop.exe`           | 单文件桌面程序（前端已内嵌）      |
| `namer-amd64-installer.exe`   | NSIS 安装包                       |
| `namer.db`                    | 现场制作的纯净种子库              |

`build.ps1` 会先备份开发库，再用 `build_seed` 现场生成干净库供打包，结束后恢复原库，避免把本机收藏/历史带进安装包。

### 前端开发

```powershell
cd frontend
pnpm install
pnpm dev          # Vite 开发服务器
pnpm test         # vitest 单元测试
pnpm e2e          # 先 build 再跑 Playwright 端到端
```

### 命令行参数

桌面端同样支持以参数覆盖默认路径，便于绿色版与多库共存：

```powershell
namer-desktop.exe -db D:\my\namer.db      # 指向新路径=自动建库并播种（干净库）
namer-desktop.exe -data .\data            # 指定物料目录
namer-desktop.exe -log .\logs             # 指定日志目录
namer-desktop.exe -help                   # 查看全部参数
```

### 数据目录与日志

按来源分级解析（详见 `bootstrap.DataDirSource`）：

| 类型        | 优先级                                                                 |
| ----------- | ---------------------------------------------------------------------- |
| 物料目录    | `-data` > `DATA_DIR` > 仓库根 `data/`（开发）> exe 同级 `data/`（打包） |
| SQLite 库   | `-db` / `NAMER_DB_PATH` > 开发=仓库根 `namer.db` > 打包=`data/namer.db` |
| 日志目录    | `-log` / `NAMER_LOG_PATH` > 开发=仓库根 `logs/` > 打包=`data/logs`      |

日志按天轮转，文件名形如 `namer-YYYYMMDD.log`，保留 32 天。

***

## 📋 功能说明

### 八字分析

- 计算年柱、月柱、日柱、时柱
- 统计五行分布（金木水火土）与旺衰
- 判断日主强弱，推导喜用神 / 忌神

### 纳音五行

- 根据出生年份查询纳音（如 2024 年 → "海中金"）
- 分析名字与年命的关系

### 生肖属相

- 根据出生年份确定生肖
- 提供相合 / 相冲生肖与喜忌字符

### 易经 64 卦

- 按姓名笔画起卦
- 解读卦辞、象辞
- 给出事业 / 财运 / 健康维度提示

### 名字评分

采用九维评分后汇总为 100 分制，逐维明细随结果一并返回；评分口径与门禁规则见 `docs/22`、`docs/33`。

### 名字收藏与对比

- 结果页点击 ❤️ 收藏，收藏页统一管理
- 勾选多个名字横向对比评分、五行、笔画等指标

### 诗词典故

- 选择「诗词经典」来源时，名字关联典籍出处
- 名字详解中展示原句与篇章来源

### 重名率查询

- 基于真实人名频率语料统计重名率
- 展示重名程度（极高 / 较高 / 一般 / 较低 / 未收录）

### 黄历万年历

- 公历 / 农历互查、节气与宜忌

### PDF 报告与分享

- 结果页一键导出结构化 PDF 报告（字体内嵌，无外部依赖）
- 生成分享快照，便于转发给家人

***

## 🎨 设计规范

| 颜色名称 | 色值      | 用途            |
| -------- | --------- | --------------- |
| 温润米白 | 米白底    | 主背景          |
| 暖白     | 暖白底    | 卡片背景        |
| 墨黑     | 深墨      | 主要文字        |
| 深红     | `#8B2323` | 强调 / 按钮 / 印章 |
| 金色     | `#BD9A5A` | 高亮 / 评分     |
| 黛青     | 青灰      | 次要文字        |

- 标题：Noto Serif SC（思源宋体）
- 正文：Noto Sans SC（思源黑体）
- 完整色板与组件规范见 `docs/07-UI设计规范.md`

***

## ✅ 质量保障

三道防线脚本，本地与 CI 共用：

```powershell
bash scripts/verify-quality.sh                # Go：单元 + 门禁审计 + 维度巡检 + E2E
bash scripts/verify-frontend.sh               # 前端：tsc + ESLint + vitest
bash scripts/smoke/cli_matrix.sh              # namer-cli 全参数矩阵 E2E（14 用例）
```

- `SKIP_SERVER=1 bash scripts/verify-quality.sh`：只跑不需服务的阶段（桌面迁移后 HTTP 服务已退役，CI 用此模式）
- pre-push hook（`.githooks/pre-push`）会在推送前自动跑快路径；`SKIP_VERIFY=1 git push` 可临时跳过
- 核心九包覆盖率门槛 75%，CI 见 `.github/workflows/quality.yml`

***

## 📦 发布

版本号唯一真源是 `CHANGELOG.md` 顶部首个 `## [YYYY.MM.DD.MICRO]`，脚本自动读取。

```powershell
powershell -ExecutionPolicy Bypass -File scripts/release.ps1              # 构建 + 打 tag + 发 Release
scripts\release.ps1 -SkipBuild                                           # 产物已就绪，仅打 tag 并发布
scripts\release.ps1 -Resume -UploadRetries 5                             # 断点续传，只补附件
scripts\release.ps1 -DryRun                                              # 只做前置校验
```

发布内容为四个附件：`namer-desktop.exe`、`namer-amd64-installer.exe`、`namer.db`、`SHA256SUMS.txt`。每个附件上传后都会用 GitHub digest 与本地 SHA256 回验；下载方可执行 `sha256sum -c SHA256SUMS.txt` 校验完整性。

> `backend/bin/` 已在 `.gitignore` 中，二进制**不进仓库历史**，只随 GitHub Release 分发。

***

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库 <https://github.com/bliubiu/babyName>
2. 创建特性分支（`git checkout -b feature/amazing-feature`）
3. 提交更改（`git commit -m 'feat: 新增某能力'`）
4. 推送分支（`git push origin feature/amazing-feature`）
5. 打开 Pull Request

提交前请确保 `scripts/verify-quality.sh` 与 `scripts/verify-frontend.sh` 全部通过，并同步更新 `CHANGELOG.md`。

***

## 📚 相关文档

- `docs/01-需求规格说明书.md` - 需求与验收口径
- `docs/05-部署使用手册.md` - 构建、分发、数据目录与企业部署
- `docs/10-起名规则.md` - 取名字表范围与避讳规则
- `docs/22-起名链路E2E巡检工具与质量防线审计.md` - 质量防线设计
- `docs/33-门禁层设计与运营闭环.md` - 门禁算法与运营
- `docs/47-桌面端Wails迁移方案与实施记录.md` - 迁移决策
- `docs/48-Wails-v2到v3升级评估与实施方案.md` - v3 升级细节

## 📄 许可

本项目仅供学习与个人使用；命理结论为传统文化视角，仅供参考。
