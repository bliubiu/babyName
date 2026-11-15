# 47 桌面端 Wails 迁移方案与实施记录

> 状态：进行中（切片式迁移，每切片全测试通过 + 独立 commit）
> 决策：仅桌面端（Wails），彻底移除 Gin HTTP 层与 Next.js，前端换 Vite + React 19

## 1. 背景与决策

原架构：Go (Gin + modernc sqlite) + Next.js 16.2 (React 19) + Docker/nginx，浏览器访问。

评估后确定迁移目标：**单机桌面应用（Wails v2）**，理由：

- 后端已是 Go，Wails 绑定零新增语言栈（对比 Tauri 需 Rust / Electron 需 Node）
- modernc.org/sqlite 纯 Go 无 CGO，单 exe 打包不受影响
- Web 部署链（docker×4、nginx、certbot、TLS）随桌面化全部退役
- 移除 Next.js SSR 心智负担，Vite 是 Wails 官方推荐前端形态

替代方案否决记录：

| 方案 | 否决理由 |
|---|---|
| Electron | 需 Node 运行时，包体 ~120MB+，与"纯 Go 轻量"冲突 |
| Tauri | 核心用 Rust，保 Go 后端需 sidecar 方案，通信复杂 |
| 保留 PWA | 无法提供原生窗口/安装器体验 |

## 2. 版本适配（重要）

| Wails 版本 | go.mod 要求 |
|---|---|
| ≤ v2.16.0 | go 1.25.0 及以下 |
| **v2.17.0** | **go 1.25.0（当前采用）** |
| v2.17.1 | go 1.26.0（强制） |

- 项目约束 go 1.25.10+，故锁定 **Wails v2.17.0**
- 曾误用 v2.17.1，连带把 go.mod 顶到 1.26，已降级回 v2.17.0 并回落 x/text v0.39.0 / x/net v0.56.0 / x/sys v0.46.0

## 3. 新架构

```
backend/
  ├ main.go                  Wails 入口（目录解析 → bootstrap 装配 → wails.Run）
  ├ wails.json               Wails 配置（frontend:dir = ../frontend）
  ├ build/                   打包资产（appicon.png / windows icon+manifest）
  ├ frontend/dist/           embed 目标：wails build 自动从 ../frontend/dist 同步
  └ internal/
      ├ desktop/             绑定层（替代 /api/v1 handler，不依赖 gin）
      ├ bootstrap/           装配层（存储 → 物料播种 → 服务装配 → 绑定）
      ├ domain/…             领域层（未动）
      └ infrastructure/…     基础设施（domain 之外保留，HTTP 部分待删）

frontend/                    Vite + React 19 + TS + Tailwind + Zustand（迁移中）
  └ wailsjs/                 wails build 自动生成的 TS 绑定（勿手改）
```

### 3.1 绑定层（internal/desktop）

覆盖原 `/api/v1` 全部 9 个在用接口（docs/27 接口利用率审计）：

| 绑定方法 | 原 REST |
|---|---|
| GenerateNames / EvaluateName / ExploreNames | POST /names/generate、/names/evaluate、/names/generate/explore |
| SubmitGenerateTask / GetGenerateTask | POST /names/generate/async、GET /names/task/:id |
| GetHistory / SaveHistory / DeleteHistory | GET/POST/DELETE /history |
| GetFavorites / SaveFavorite / DeleteFavorite | GET/POST/DELETE /favorites |
| GenerateReportPDF | POST /report/pdf |
| CreateShare / GetShare | POST /share、GET /share/:id |
| GetNameStats / GetTopNames | GET /namestat/:name、GET /namestat |
| GetHuangli / GetLunarCalendar / GetLunarMonthCalendar | GET /huangli、/lunar、/lunar/month |
| GetCharGroups | GET /characters/groups |

约定：

- **错误**：绑定方法返回的 error 是字符串文案，统一中文、可行动。映射与 HTTP
  `writeServiceError` 同口径：超时→"请求超时，请稍后重试"、`AppError`→透传
  原消息、`ErrTooManyTasks`/`ErrShareNotFound`/`ErrInvalidShare`→原消息、
  其余→兜底文案。内部错误细节只落日志，绝不回传前端
- **会话**：桌面单用户，固定会话 `desktop`（跨重启稳定，收藏/历史归属依赖）
- **校验**：与 HTTP 层同口径复用 `validator` 包（姓氏/性别/日期/时辰/分页上限 100）
- **窄接口**：TaskRunner/TaskView/ShareStore/CharacterSource/NameStatSource
  均为 local interface + 适配器，测试用替身不触碰 services 具体类型

### 3.2 装配层（internal/bootstrap）

```
ResolveDataDir（显式 > DATA_DIR > 探测仓库根 data/ > cwd/data）
  → initStore（SQLite 优先，失败/未配置回落内存）
  → seedMaterialsIfEmpty（首跑库空则 JSON 播种，Import 指纹短路幂等）
  → loadCulturalData（ADR-008：SQLite 唯一真源，必需物料缺失拒绝启动）
  → classics.BuildPoetryIndex（诗词索引启动期预热）
  → assembleServices（fate 引擎 + NameDB 自学习 + 门禁闭环，同原 newServices）
  → desktop.New(...)
```

### 3.3 目录语义（桌面端，S4 起按来源分级）

数据目录来源（`bootstrap.DataDirSource`，五级）：
`-data` > `DATA_DIR` > 仓库根 data/（dev 探测）> exe 同级 data/（打包）> cwd/data（兜底）

| 项 | dev（命中仓库结构） | 打包（exe 同级 data/ / cwd/data） |
|---|---|---|
| 数据库 | `-db` > `NAMER_DB_PATH` > 仓库根 namer.db | `-db` > `NAMER_DB_PATH` > data 文件夹内 namer.db |
| 日志目录 | `-log` > `NAMER_LOG_PATH` > 仓库根 logs/ | `-log` > `NAMER_LOG_PATH` > data/logs |

> 打包态安装目录整体自包含（exe + data/ 含物料、库、日志），可整体拷贝迁移。

## 4. 迁移切片与进度

| 切片 | 内容 | 状态 |
|---|---|---|
| S1 | Wails 骨架 + 绑定层 + 装配层（TDD） | ✅ commit 1c4abc2 / 1164241 |
| S2 | 前端 Vite 迁移（Next.js → Vite + React 19 + react-router） | ✅ commit 9ef6d47 |
| S3 | 移除 Gin / cmd/server / middleware HTTP / docker / nginx / certbot | ✅ |
| S3.5 | 物料管理页（状态查看 + 手动重导，ADR-008 方案 B） | ✅ commit bdd5332 |
| S4 | 用户数据目录（exe 同级 data 便携语义）+ NSIS 安装器 + 图标 | ✅ commit 79d46d2 + 本切片 |
| S5 | e2e 迁移（wails dev :34115 + Playwright）+ 文档收尾 | ✅ 本切片（见 S5 小节） |

### S2 前端 Vite 迁移清单

1. package.json：移除 next，新增 vite / @vitejs/plugin-react；build → `vite build`（outDir dist）
2. `next/link` → `react-router`（或 hash 路由）；`next/navigation` hooks 逐个替换
3. `next/image` → `<img>`；`"use client"` 指令删除；`next/head` → `index.html`
4. 环境变量 `NEXT_PUBLIC_*` → `import.meta.env.VITE_*`（桌面端多数可删：直连绑定）
5. API 层替换：fetch 封装 → `import { ... } from "../wailsjs/go/desktop/App"`
6. React Query 保留：queryFn 直接调绑定方法（缓存/加载态仍有价值）
7. wails.json 填回 `frontend:install: pnpm install` / `frontend:build: pnpm build`
8. e2e（webui_e2e_test.py / @e2e-dev/web）随 Next 一起退役，S5 重写

### S3 移除清单（已完成）

Go 侧：

- `backend/cmd/server/`：HTTP 入口与装配测试（装配逻辑已收编 internal/bootstrap）
- `backend/internal/application/handlers/`：Gin handler 层（desktop 绑定层替代，
  错误映射同口径；整合测试由 desktop/bootstrap 测试承担）
- `backend/internal/application/response/`：HTTP 信封层
- `backend/internal/infrastructure/server/`：前端静态托管 + embed placeholder
- `backend/internal/infrastructure/middleware/`：CORS/限流/会话 Cookie/请求体上限
- `backend/webui_e2e_test.py`：web e2e（S5 重写为桌面版）
- go.mod 依赖收缩：gin、labstack/echo（间接）、相关 golang.org/x 依赖随 tidy 移除

部署链：

- `backend/Dockerfile`、`docker-entrypoint.sh`、`.dockerignore`
- 根目录 `docker-compose*.yml` ×4、`deploy.sh`、`nginx/`（3 份 conf）
- `scripts/start.bat`、`start.ps1`、`start_services.ps1`（web 启动脚本）
- `scripts/build.ps1` 重写为 Wails 桌面版（pnpm build → wails build → build/bin）
- `scripts/verify-quality.sh` 加注：非 SKIP_SERVER 模式随 HTTP 层退役，
  CI 保留 SKIP_SERVER=1 快路径

保留说明：

- `backend/application.yml` 与 `internal/infrastructure/config`：cmd/materials
  物料工具仍在使用，仅 server/rate_limit/redis 段随 HTTP 层失效
- `cmd/namer-cli` 等物料/导出 CLI：与传输层无关，全部保留

### S3.5 物料管理页（插入切片，ADR-008 方案 B）

S3 与 S4 之间插入的桌面端专属功能：物料库状态查看 + 手动重导。

后端：

- `internal/desktop`：新增 `MaterialManager` 窄接口与 `GetMaterialStates` /
  `ImportMaterials` 绑定方法；内存存储模式降级——状态返回空列表、重导报
  明确提示（"当前未使用 SQLite 存储，物料管理不可用"），绝不静默空转
- `internal/bootstrap`：`sqliteMaterialManager` 适配器（窄接口 local interface，
  仅 SQLite 存储接线）；重导按装配时物料目录执行（指纹短路幂等）

前端：

- `/materials` 页面：状态表格（版本/指纹缩略/行数千分位/导入时间）+ 重导
  按钮 + 逐项导入明细（写入/跳过/失败三分支）+ 内存模式提示
- 重导只写 SQLite，运行期内存缓存（诗词索引/字频统计）需重启生效——页面常驻说明
- 导航入口（数据库图标）；wailsjs TS 绑定重新生成

### S4 用户数据目录 + NSIS 安装器（已完成）

决策记录：

- **物料分发：外置随安装器**——data/ 顶层 36 物料 JSON + materials.lock +
  raw/kangxi-strokecount.csv（约 63MB）打进安装器，装到安装目录 data/。
  运行期零 JSON 依赖（ADR-008，`InitWithDB` SQLite 策略不读文件），
  data/ 仅首跑播种与手动重导使用；内嵌 embed 方案否决（exe 80MB+，
  违背"从源头减小体积"）
- **用户数据目录：exe 同级 data 文件夹**（便携语义，不用 %AppData%）——
  打包态 namer.db 与 logs/ 收敛到 data 文件夹内，安装目录整体自包含；
  dev 仓库布局不变（db/日志仍在仓库根）
- **安装作用域：user**（`%LocalAppData%\Programs\NameMaster`，免管理员）——
  便携语义依赖安装目录可写，Program Files 不可写与之冲突

实现：

- `paths.FindDataDirFromExe`：exe 同级 data/ 探测（namer.json 标记判定，
  纯函数可测）
- `bootstrap.ResolveDataDirDetailed`：五级来源解析；`DefaultDBPath` /
  `DefaultLogDir` 按来源推导（dev 沿用仓库根，打包收敛 data 文件夹内）；
  原 `ResolveDataDir` 薄包装不变
- `build/windows/installer/project.nsi`（wails 模板定制，wails 不覆盖）：
  - `SetCompressor /SOLID lzma`：20MB exe + 63MB 物料 → 安装器 13.4MB
  - 安装 Section 物料随装：`File ..\..\..\..\data\*.json` + materials.lock +
    raw/kangxi CSV（makensis CWD = installer 目录，4 级上行到仓库根）
  - 卸载 Section 选择性删除：程序/物料/快捷方式/注册表键全删，**保留
    namer.db / *.db-wal / *.db-shm / logs/**（防卸载抹掉收藏/历史），
    `RMDir $INSTDIR` 仅空目录生效
  - `INFO_PRODUCTNAME "NameMaster"`：避免中文+空格进安装路径/快捷方式
  - 简体中文第一安装语言
- `scripts/build.ps1` 步骤 [3/3]：`wails build -nsis -installscope user`；
  makensis 缺失时提示 winget 安装并跳过（`-SkipInstaller` 可强制跳过）
- 图标：appicon.png / icon.ico / manifest 已齐备，无新增

NSIS 环境注意（四处编码/获取坑）：

- makensis 获取：sourceforge 在当前网络不可达（winget/curl 均超时），
  改取 npmmirror electron-builder-binaries 的 NSIS 3.05（makensis + Plugins
  合并解压至 `~/tools/nsis`，已加入用户 PATH）
- `project.nsi` 必须带 **UTF-8 BOM**：makensis 以 BOM 识别 UTF-8，
  BOM 缺失按 ANSI 解析，中文注释直接报 `Bad text encoding`
- `wails.json` 的 info 字段改 ASCII（productName/copyright/comments）：
  wails_tools.nsh 每次构建 **BOM-less 重新生成**，含非 ASCII 必炸编译；
  窗口标题不受影响（来自 main.go 的 options.Title）
- SimpChinese.nlf/.nsh 需补 BOM（官方包 BOM-less；纯 ASCII 语言不受影响，
  含中文的语言文件必须 BOM）
- `wails_tools.nsh` / `installer/tmp/` 已加入 .gitignore（每次构建重写）

### S5 桌面端全量 E2E 验证（已完成）

桌面端没有 HTTP 端口，旧 `e2e`（@e2e-dev/web / webui_e2e_test.py）随 Next.js
一起退役。S5 改用「wails dev 浏览器通道 + Playwright」覆盖 UI 全链路，再补一层
Go 侧真实装配 E2E 覆盖绑定层全部方法，避免 UI 盲区。

**运行方式**

```
cd frontend && pnpm e2e        # 脚本已前置 pnpm build，杜绝 dist/ 旧产物假通过
cd backend  && go test ./internal/bootstrap/ -run TestAppE2E -count=1 -v
```

`playwright.config.ts` 由 webServer 自动拉起 `wails dev -assetdir ../frontend/dist`
（:34115），按「显式 env > 固定安装位 > PATH」探测 Wails 2.17.0 CLI；
`NAMER_DB_PATH` 指向系统临时目录的隔离库，不污染仓库根 namer.db；
`workers: 1` 强制串行（收藏/历史共享单实例后端，跨用例会互扰）。

**覆盖**

| 层 | 用例 | 范围 |
|---|---|---|
| UI（Playwright） | 18 例 / 4 spec | 起名主链路与条件表单（字辈/单字名/经典来源）、跨页收藏与对比、测名、黄历、重名率、历史/JSON/物料页、结果页换一批 / PDF 导出 / 分享往返 / 历史删除 |
| Go 绑定层 E2E | 12 组 / 22 个方法 | `bootstrap.Run` 真实 SQLite + 63MB 物料装配后逐一调用全部 Bind 方法：同步生成、换一批、测名、异步任务、历史/收藏 CRUD、PDF、分享、黄历/农历/月历、汉字分组/统计、物料状态与重导，另含 10 项参数校验与错误分支 |

**E2E 暴露并修复的契约缺陷（HTTP → Wails 迁移遗留）**

1. 固定会话标识 `desktop` 字面量未通过服务层 `NormalizeSessionID`（要求
   16–64 位 hex）→ 收藏/历史全链路报「会话标识无效」；改为 `sha256("desktop")`
   的 64 位十六进制，语义仍为跨重启稳定的单用户会话。
2. `exportReportPDF` / `createShareSnapshot` 仍按 HTTP 信封传 `{data:...}`，
   而 Wails 绑定直接透传：后端分享字段白名单把顶层唯一的 `data` 键剔除，
   报「分享内容不合法: 没有可分享的内容」；PDF 的 `parseReportModel` 也读不到
   真实字段（生成空壳报告）。改为传扁平对象。
3. Go `[]byte` 返回值经 Wails `json.Marshal` 实际是 **base64 字符串**，而
   typescriptify 生成的 TS 声明写 `Array<number>`；`new Uint8Array(字符串)`
   静默得到空数组 → 前端报「PDF 内容为空」且不触发下载。前端兼容两种形态。
4. 分享链接返回 `/share?id=`：资产服务器对未知路径返回 404（HashRouter 无
   SPA fallback），改为 `/#/share?id=`。

以上 2~4 项由 `src/lib/api.test.ts`（5 例）与 Go 绑定层 E2E 的分享/PDF
子用例共同固化，防止回归。

**已知噪声**：外部浏览器加载时 Wails dev 每次打印
`Unknown message from front end: runtime:ready`。这是浏览器通道不是原生
WebView 造成的握手噪声，绑定调用与全部用例均正常，不影响判定。

## 5. 验证记录

- [x] `wails build` 出包 18.8MB（纯 Go，无 CGO/Node 运行时）
- [x] `go test ./...` 34 包全绿（含新增 desktop 21 例、bootstrap 8 例）
- [x] golangci-lint（v2.15）新包 0 issues；仓库级 .golangci.yml 为 v1 格式，
      与 golangci-lint v2 不兼容，需后续单独迁移配置
- [x] Wails 降级 v2.17.0 后重建通过（go 1.25.0）
- [x] S2：tsc 无错、vitest 151 例全绿、vite build 7.5s（gzip 主包 112KB）
- [x] S3：HTTP 层移除后 `go build ./...`、全量 `go test ./...`、wails build 均通过
- [x] S3.5：go test 29 包全绿（desktop 37 例、bootstrap 12 例，含物料接线 2 例
      经 Run 全链路）；tsc 无错、vitest 156 例全绿（含物料页 5 例）；
      gofmt 干净（desktop/bootstrap 包）
- [x] S4：便携冒烟通过——DATA_DIR 指向冒烟布局，真实 63MB 物料播种成功
      （kangxi 63696 行、name_bigram 124994 行…），db/logs 收敛 data 文件夹
      内，WAL 生效
- [x] S4：安装器 E2E——静默安装布局正确（namer.exe + 38 项物料 + raw/ 康熙表）；
      静默卸载程序/物料/快捷方式/注册表键全删而 namer.db/-wal/-shm/logs 保留；
      安装器 13.4MB（LZMA 固实压缩）
- [x] S4：go test 34 包全绿（paths +4 例、bootstrap +10 例），go vet 干净
- [x] S5：Playwright 桌面端 E2E **18/18**（wails dev :34115，4 spec，含结果页换一批 / PDF 导出 / 分享往返 / 历史删除）
- [x] S5：Go 侧真实装配全量绑定 E2E **12 组 / 22 个方法**（`go test ./internal/bootstrap/ -run TestAppE2E`，13.7s）
- [x] S5：`go test ./...` 52 包全绿、`go vet ./...` 干净、`verify-quality.sh`（SKIP_SERVER=1）全绿（九包覆盖率 76.3%~94.9%）
- [x] S5：前端三件套 tsc 0 错 / ESLint 0 error（≤34 warning）/ Vitest **161/161**

## 6. 遗留与风险

- 绑定生成时 `Not found: time.Time` 告警：wails TS 绑定不映射 time.Time
  （HistoryRecord/FavoriteRecord.CreatedAt），前端拿到的是 ISO 字符串，可用
- `backend/frontend/` 是 embed 目标占位（构建期被同步/还原），不要手工放源码
- 物料分发策略已在 S4 决策：外置随安装器（见 S4 小节）
- dev 机器上运行的打包 exe 仍命中 dev 仓库探测（编译期源码路径经
  runtime.Caller 嵌入二进制，仓库结构存在即优先），exe 同级 data/ 分支仅在
  无仓库结构的干净机器上生效——探测逻辑由单测覆盖（临时布局注入）
- Wails 生成的 TS 绑定把 `[]byte` 声明为 `Array<number>`，运行时却是 base64
  字符串（Go `json.Marshal` 语义）。新增二进制返回值的绑定时须在前端做形态
  兼容，不可照抄 .d.ts 直接 `new Uint8Array(...)`（已在 `src/lib/api.ts` 处理）
- `pnpm e2e` 已前置 `pnpm build`；若手动 `wails dev` 调试，改了前端源码必须
  自行重建 `frontend/dist`，否则浏览器测到的是旧产物
- Wails dev 浏览器通道的 `runtime:ready` 噪声见 S5 小节，不影响绑定调用
