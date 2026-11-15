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

### 3.3 目录语义（桌面端）

| 项 | 优先级 |
|---|---|
| 物料目录 | `-data` 参数 > `DATA_DIR` > 仓库根 data/（dev）> cwd/data（打包） |
| 数据库 | `-db` 参数 > `NAMER_DB_PATH` > 物料目录上一级 namer.db |
| 日志目录 | `-log` 参数 > `NAMER_LOG_PATH` > 物料目录上一级 logs/ |

> 打包后的用户数据目录迁移（os.UserConfigDir + 首跑引导）留待发布切片。

## 4. 迁移切片与进度

| 切片 | 内容 | 状态 |
|---|---|---|
| S1 | Wails 骨架 + 绑定层 + 装配层（TDD） | ✅ commit 1c4abc2 / 1164241 |
| S2 | 前端 Vite 迁移（Next.js → Vite + React 19 + react-router） | ✅ commit 9ef6d47 |
| S3 | 移除 Gin / cmd/server / middleware HTTP / docker / nginx / certbot | ✅ |
| S4 | 用户数据目录（UserConfigDir）+ 安装器（NSIS）+ 图标 | ⏳ |
| S5 | e2e 迁移（wails dev :34115 + Playwright）+ 文档收尾 | ⏳ |

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

## 6. 遗留与风险

- 绑定生成时 `Not found: time.Time` 告警：wails TS 绑定不映射 time.Time
  （HistoryRecord/FavoriteRecord.CreatedAt），前端拿到的是 ISO 字符串，可用
- `backend/frontend/` 是 embed 目标占位（构建期被同步/还原），不要手工放源码
- 物料随 exe 分发策略（内嵌 vs 外置 data/）在 S4 决策
