# 48 Wails v2 → v3 升级评估与实施方案

> 状态：**阶段 0~4 已在隔离分支 `refactor/wails-v3` 实施完成**（实施记录见 §9）；
> master 仍保持 Wails v2.17.0 可交付，未完成合并前不切换主线
> 日期：2026-11-15 评估，2026-11-16 实施
> 关联：`docs/47`（v2 桌面端迁移实施记录）、2026-11-15「Wails vs MyGo 选型决策」

## 1. 结论

**可以升，但不要在 master 上直接升；建议隔离分支按 5 个阶段推进，阶段 0 作为 go/no-go 判据。**

上一轮选型决策把「Go 工具链 → Wails v3」列为唯一值得考虑的升级路径，并判定 Go 版本是硬约束。本轮实测**推翻了这条阻塞结论**：本机已缓存 Go 1.26.x / 1.27.1 工具链，`v3.0.0` 模块可下载，`wails3` CLI 可编译安装。技术门槛已解除，剩下的是工程成本与稳定期风险：

- 官方明确把 v2→v3 定性为 **"a real port rather than a version bump"**，不是改依赖号；
- v3.0.0 于 **2026-10-26 GA**，至本轮仅约 3 周，之后是 nightly + 每周次版本节奏；
- 项目 2026-11-15 刚完成 S1–S5 全链路迁移并推送 `5fc2217`，处于稳定期；
- 官方「典型应用 1–4 小时」不适用于本项目（23 个绑定方法、12 组 Go 装配 E2E、23 例 UI E2E、定制 NSIS + 63MB 物料、质量门禁双套）。

因此：**不做「一次性重写」，做「隔离分支 + 每阶段可回滚 + 每阶段独立验收」**。v2 分支保持可交付，直到 v3 分支全量回归通过。

## 2. 事实基线（官方，2026-11-15 核实）

| 项 | 结论 | 出处 |
|---|---|---|
| 版本与状态 | v3.0.0 GA 2026-10-26；Windows/macOS/Linux 桌面稳定；Android/iOS 实验性 | v3 发布公告 / Roadmap |
| Go 门槛 | **所有平台 Go ≥ 1.26**；v3 模块 `go 1.26.0` / `toolchain go1.26.10` | 安装页 / `v3@v3.0.0/go.mod` |
| v2 维护 | v2 不消失，继续收 **critical** 更新；无迁移截止期 | 发布公告 / 迁移指南 FAQ |
| CLI | `wails3` 独立二进制，模块路径 `wails/v3`，与 `wails`（v2）可同机并存 | 发布公告 |
| 发布节奏 | v3.0.0 后 nightly 预发布 + 每周次版本 → **必须精确 pin**（Go module / CLI / `@wailsio/runtime` 三者同版本） | 发布公告 |
| 四大架构变化 | 应用与窗口生命周期；服务取代 context-bound 绑定；直接 app/window API 取代 v2 runtime 包；前端绑定重新生成 | 迁移指南 |
| 迁移量级 | 官方估计「典型应用 1–4 小时」 | 迁移指南 |
| 打包 | `wails3 package GOOS=windows` → NSIS；源脚本仍在 `build/windows/nsis/project.nsi`；产物 `bin/` | Windows Packaging / Building |

> 注意：官方**迁移指南 Step 1 的 `go.mod` 示例仍写 `go 1.25.0`**，与安装页、Roadmap、v3 模块实际 `go.mod` 冲突。以 **Go ≥ 1.26** 为准，示例属文档陈旧。

## 3. 本机实测基线（本轮新增证据）

以下均为只读探测或 scratch 目录（`.workbuddy/tmp/`）内操作，**未修改仓库任何文件**：

| 探测 | 命令 | 实测结果 |
|---|---|---|
| 当前工具链 | `go version` / `go env GOTOOLCHAIN` | `go1.25.10 windows/amd64`，`GOTOOLCHAIN=auto` |
| 已缓存 toolchain | 列 `GOMODCACHE/golang.org/toolchain@*` | 1.26.0 / 1.26.7 / 1.26.9 / 1.26.10 / 1.26.11 / 1.27.1 |
| 切换可用 | `GOTOOLCHAIN=go1.26.11 go version` | `go version go1.26.11 windows/amd64` ✅ |
| 依赖可获取 | `go mod download -json github.com/wailsapp/wails/v3@v3.0.0` | exit 0，模块已落缓存 ✅ |
| CLI 可安装 | `GOBIN=... go install .../cmd/wails3@v3.0.0` | exit 0，产出 `wails3.exe`（2m50s）✅ |
| 绑定生成 | scratch 最小项目 + `wails3 generate bindings -ts` | `250 Packages, 1 Service, 4 Methods, 0 Enums, 2 Models in 6.4s` ✅ |

**结论：上一轮「无法新增 Go 依赖 / 必须等 Go 工具链升级」的判断在当前环境已不成立**，可直接进入阶段 0。

## 4. 契约对照（v2 现状 → v3 目标）

| 层 | v2 现状（本项目） | v3 目标 | 改动性质 |
|---|---|---|---|
| 应用入口 | `backend/main.go:96` `wails.Run(&options.App{...})` | `application.New(...)` + `app.Window.NewWithOptions(...)` + `app.Run()` | 重写单点 |
| 绑定注册 | `Bind: []interface{}{res.App}` | `Services: []application.Service{application.NewService(res.App)}` | 改写 |
| 生命周期 | `OnStartup: res.App.Startup`（`desktop.go:221` 导出方法） | `ServiceStartup(ctx, application.ServiceOptions) error` | **必须改名** |
| 资源 | `//go:embed all:frontend/dist` + `assetserver.Options{Assets: assets}` | `//go:embed frontend/dist` + `Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)}` | 改写 |
| 配置 | `backend/wails.json` 扁平键 `frontend:install` 等 | 嵌套 `frontend: {dir, install, build, dev, devServerUrl}` + Taskfile | 重写 |
| 前端导入 | `src/lib/api.ts:22` `../../wailsjs/go/desktop/App` | `./bindings/<module>/app` | 一处改（`api.ts` 是唯一封装） |
| 调用机制 | `window['go']['desktop']['App']['X'](arg1)` | `$Call.ByID(数字ID, args)`，import `@wailsio/runtime` | 生成物变更 |
| `[]byte` 返回 | 运行时 base64 字符串，`.d.ts` 却写 `Array<number>`（S5 已修前端兼容） | 生成器明确 `Promise<string>` + `$Create.ByteSlice()` | **契约漂移从源头修掉** |
| `json.RawMessage` | JSON 对象透传 | 实测生成 `export type RawMessage = any` | **不变**，分享链路前端几乎无需改 |
| 导出 `Startup` | 私有生命周期 hook | 实测被生成为 `Startup(): Promise<void>` 前端可调用方法 | 污染 API 面，须改名 |
| 事件 / runtime | 前端 `wailsjs/runtime` 调用数 **0** | `@wailsio/runtime` | 无迁移成本 |
| 开发服务器 | `wails dev -assetdir ../frontend/dist`（34115 浏览器通道，带 `runtime:ready` 噪声） | `wails3 dev`（Vite dev 9245 反代）或 server build | 重搭 |
| E2E 通道 | `playwright.config.ts` webServer 拉起 `wails dev` | 官方 **server build**：`Server: application.ServerOptions{...}` + `go run -tags server,production .` | 改配置，非重搭 |
| 打包 | `backend/build/windows/installer/project.nsi`（`wails.*` 宏 + 63MB 物料 + 选择性卸载） | `build/windows/nsis/project.nsi` | 路径/宏兼容性待 spike 验证 |
| CI | `.github/workflows/quality.yml` `go-version-file: backend/go.mod` | go 指令升 1.26 自动带过；需加 wails3 安装与绑定生成步骤 | 增量改 |

### 4.1 绑定生成实测摘录（scratch 探针）

```
INFO  Processed: 250 Packages, 1 Service, 4 Methods, 0 Enums, 2 Models, 0 Events in 6.4s
INFO  Output directory: .../frontend/bindings

frontend/bindings/encoding/json/models.ts : export type RawMessage = any;
frontend/bindings/v3probe/app.ts:
  CreateShare(data: json$0.RawMessage): $CancellablePromise<$models.ShareResult | null>
  GenerateReportPDF(data: any): $CancellablePromise<string>   // $Create.ByteSlice($result)
  GetShare(id: string): $CancellablePromise<json$0.RawMessage>
  Startup(): $CancellablePromise<void>                         // v2 的导出 Startup 被误暴露
```

补充证据：`v3@v3.0.0/pkg/application/bindings.go:195-200` 的 `internalServiceMethods` 只排除
`ServiceName / ServiceStartup / ServiceShutdown / ServeHTTP`，不含 `Startup`；
`internal/generator/render/type.go:56-59` 仅对 `types.Identical(typ, []byte)` 渲染为 base64 `string`，
`json.RawMessage` 是命名类型，实测走 `any`。

### 4.2 新增构建约束

- 绑定用 `$Call.ByID(数字ID)`，ID 由方法全名计算 → **Go 方法增删/改名后必须重新生成绑定**，否则前后端 ID 错位。可评估 `-names` 生成 `$Call.ByName(...)` 换取稳定性。
- `frontend/bindings/` 是生成物，应入库并禁止手改（与现有 `wailsjs/` 同样的纪律）。
- 新增前端依赖 `@wailsio/runtime`，用 pnpm 安装并**与 CLI / Go module 精确 pin 同一版本**；AGENTS.md 的「禁用 npm」仍成立，改 Taskfile 使用 pnpm。

## 5. 影响面清单（文件级）

| 档 | 文件 | 说明 |
|---|---|---|
| **必须改** | `backend/main.go` | 唯一 import wails 的 Go 文件；目录解析 / 日志 / `bootstrap.Run` 装配逻辑全部保留 |
| 必须改 | `backend/go.mod` / `go.sum` | `wails/v2 v2.17.0` → `wails/v3 v3.0.0`、`go 1.26.0`；`go mod tidy` 会重解析 modernc sqlite / x-text 等 |
| 必须改 | `backend/internal/desktop/desktop.go` | `Startup` → `ServiceStartup`；方法体与 22 个业务方法不动 |
| 必须改 | `backend/wails.json`、新增 `Taskfile.yml` / `build/config.yml` | v3 配置与构建任务 |
| 必须改 | `frontend/src/lib/api.ts`（1 处 import）、`api.test.ts`（mock 路径）、`frontend/wailsjs/` → `frontend/bindings/` | 生成物替换；业务调用封装逻辑不变 |
| 必须改 | `frontend/playwright.config.ts` | webServer 改 `go run -tags server,production .`，baseURL 改 8080，保留隔离库 `NAMER_DB_PATH` |
| 必须改 | `backend/build/windows/**`、`.github/workflows/quality.yml`、`AGENTS.md` | NSIS 定制段、CI 步骤、技术栈表 |
| **原样复用** | `backend/internal/bootstrap/**`（含 `z_app_e2e_test.go` 12 组真实装配 E2E） | 框架无关，不依赖 GUI，是迁移期最可靠的回归网 |
| 原样复用 | 全部 domain / application / infrastructure、`data/` 物料与 `namer.db` 语义 | 与桌面框架解耦 |
| 原样复用 | 11 个页面与 `docs/06` / `docs/07` 设计系统 | WebView 仍是同一套 React 19 + HashRouter |
| 原样复用 | 23 例 Playwright spec 的断言 | 只换承载通道，不改业务断言 |
| **待验证** | `wails.*` NSIS 宏、`wails3 package` 产物、material 随装与选择性卸载 | 阶段 4 实测 |

## 6. 分阶段方案（每阶段独立 commit，失败即停）

### 阶段 0：可行性 spike（go/no-go，不改 master）

在隔离分支 `refactor/wails-v3` 上：

1. `go.mod` 改 `go 1.26.0` + `require github.com/wailsapp/wails/v3 v3.0.0`，`GOTOOLCHAIN=go1.26.11 go mod tidy`；
2. 验收：`go test ./... -count=1` 与 `SKIP_SERVER=1 bash scripts/verify-quality.sh` 全绿（确认 toolchain 切换没有打破现代 c sqlite / 覆盖率门禁）；
3. 用 scratch 探针的 `wails3 generate bindings -ts` 对**真实 `desktop.App`** 跑一次生成，核对 23 个方法、`json.RawMessage`、`[]byte`、`Startup` 的实际输出；
4. 验收：生成物可 `tsc --noEmit`；若 `Startup` 被误暴露或服务注册报错，先改 `ServiceStartup`。

**判据**：阶段 0 通过才继续；若 Go 1.26 导致依赖或门禁不可控，回退分支，维持 v2.17.0。

### 阶段 1：应用外壳与服务

- `main.go` 改 `application.New` / `app.Window.NewWithOptions` / `app.Run()`，`res.App` 注册为 service，`OnStartup` → `ServiceStartup`；
- `wails.json` 改嵌套 frontend 配置，补 Taskfile（`pnpm build` / `pnpm dev`）；
- 验收：`wails3 build` 产出 exe，手工启动能打开首页并调用一次 `GenerateNames`。

### 阶段 2：绑定与前端

- 生成 `frontend/bindings/`，删除 `frontend/wailsjs/`；`api.ts` / `api.test.ts` 改 import；pnpm 安装并 pin `@wailsio/runtime`；
- 重点对齐 4 处契约：PDF base64、`createShareSnapshot` 扁平对象、分享链接 `/#/share?id=`、`GetShare` 返回 `any`；
- 验收：`bash scripts/verify-frontend.sh --require` 全绿（tsc / ESLint / Vitest 161 例，其中 `api.test.ts` 5 例是契约守门）。

### 阶段 3：验证链重搭

- `main.go` 加 `Server: application.ServerOptions{Host: "127.0.0.1", Port: 8080}`；
- `playwright.config.ts` webServer 改 `go run -tags server,production .`（`generate bindings -f '-tags=server'`），baseURL 8080，保留隔离库；
- 验收：`pnpm e2e` **23/23**；Go 绑定 E2E 12 组仍全绿；`runtime:ready` 噪声应消失。
- 官方边界：server build **不覆盖**原生 WebView 窗口、菜单、托盘、原生对话框，viewport ≠ 原生窗口尺寸；本项目 23 例均为页面级，可平移，但 exe 级冒烟仍需人工做一次。

### 阶段 4：打包、CI 与文档

- `wails3 package GOOS=windows`，把「63MB 物料随装」与「卸载保留 `namer.db`/WAL/SHM/logs」两段移植到新 `project.nsi`；
- CI 加 `wails3` CLI 安装与 `generate bindings`，`go-version-file` 自动跟随 go.mod；
- 更新 `docs/47 §2` 版本表、`AGENTS.md` 技术栈、`CHANGELOG`；
- 验收：安装器在干净目录安装 → 首跑播种 → 收藏/历史落库 → 卸载重装数据仍在；`git push` 前 pre-push 门禁全绿。

## 7. 收益、风险与「暂不做」条件

**实质收益（相对 v2）**

1. 生成器从源头修掉 `[]byte` 契约漂移（实测 `Promise<string>` + `ByteSlice`），并保留参数名 / 注释 / 枚举，TS 类型可信度提升；
2. server build 让 Playwright 直连真实 Go 服务，E2E 可上无 GUI 的 CI，去掉 `wails dev` 浏览器通道噪声；
3. 内置 updater（签名验证 + 原子替换 + 回滚）、WebView2 进程崩溃恢复、多窗口 / 托盘 / 原生菜单（后三项本项目当前不需要）；
4. <1ms bridge、启动/内存优化（官方口径，本项目未实测，不作为决策依据）。

**风险**

- v3 GA 仅约 3 周 + 每周次版本，需精确 pin 三者版本；`GHSA-2p5f-2gx8-mh8m` 仅部分修复（同源 child frame 可达 bound methods）——本项目纯本地静态资源、无第三方 iframe，风险低但需在安全评审中记录；
- `go 1.26` 切换会重解析依赖，覆盖率门禁偶发失败（2026-11-15 已见 `yijing`）可能重现；
- `Startup` → `ServiceStartup` 属于生命周期语义变更，若遗漏会让取消信号传播失效（服务层用 `ctx` 传播取消）；
- 定制 NSIS 与 CI 是最容易被「1–4 小时」估计漏掉的两块。

**暂不做的条件**：若近期没有 macOS/Linux、自动更新或多窗口需求，维持 v2.17.0 的成本最低；v2 继续收 critical 更新，没有迁移截止期。

**复查触发**：① 需要自动更新或第二平台 ② v3 发布 3.1+ 且每周节奏稳定 ③ Go 工具链全项目统一到 1.26+ ④ 出现与主链路无关的小工具可作隔离试点。

## 8. 本轮未处理项

- 仓库代码、配置、CI、NSIS **一行未动**；仅新增本文档与 CHANGELOG 条目。
- 工作区存在**非本轮产生**的 `AGENTS.md` 未提交改动（Go 版本标注改为 `1.25.10+|1.26.11+`）；顺带提示：Markdown 表格内未转义的 `|` 会破坏表格列，建议写成 `1.25.10+ / 1.26.11+`。本轮未修改该文件。
- `.workbuddy/tmp/` 下的 `tools/wails3.exe` 与 `v3-probe/` 是 scratch 探针产物，可保留供阶段 0 复用。

---

## 9. 实施记录（2026-11-16，分支 `refactor/wails-v3`）

> master 未改动：v2.17.0 主线保持可交付，本节只描述隔离分支。

### 9.1 阶段 0：可行性 spike（通过）

- `go.mod` 升 `go 1.26.0` + `wails/v3 v3.0.0`；`go mod tidy` 连带 `modernc.org/sqlite` 1.29.5→1.44.3、`golang.org/x/text` 0.42.0。`go test ./...`（52 包）、`go vet ./...`、`SKIP_SERVER=1 bash scripts/verify-quality.sh` 全绿。
- 真实 `desktop.App` 绑定生成：`1 Service, 22 Methods`；`json.RawMessage` → `any`、`[]byte` → `Promise<string>` + `$Create.ByteSlice()`；改名后的 `ServiceStartup` 被生成器正确排除。

### 9.2 阶段 1：应用外壳与服务

- `main.go` 改为 `application.New` / `application.NewService(res.App)` / `BundledAssetFileServer(assets)` / `app.Window.NewWithOptions` / `app.Run()`；`Startup` → `ServiceStartup(ctx, application.ServiceOptions) error`，`desktop_test.go` 与 `z_app_e2e_test.go` 同步调用。
- 新增 `Server: application.ServerOptions{Host: "127.0.0.1", Port: 8080}` 供 E2E。**已核实**（`pkg/application/application_options.go:125-134`、`application_server.go:1` 的 `//go:build server`）该字段只在 `-tags server` 构建下生效，桌面生产 exe 不会监听 8080。

### 9.3 阶段 2：绑定与前端

- `frontend/bindings/` 生成物入库、`frontend/wailsjs/` 删除；`api.ts`（唯一封装）改 import，`api.test.ts` 改 mock 路径，`eslint.config.js` / `vitest.config.ts` 的忽略/排除同步；`@wailsio/runtime@3.0.0` 精确 pin。
- 新增 `frontend/scripts/sync-dist.mjs` 并挂到 `pnpm build` 末尾：v3 不再由 CLI 自动把 `../frontend/dist` 同步进 embed 目录，缺这一步会重现 docs/47 S5 的「改了 src 却服务旧 dist」E2E 假通过。
- 4 处契约对齐：PDF base64 解码、`createShareSnapshot` 扁平对象、分享链接 `/#/share?id=`、`GetShare` 返回 `any`。

### 9.4 阶段 3：验证链重搭

- `playwright.config.ts` webServer → `go run -tags server,production .`（127.0.0.1:8080），保留隔离库 `NAMER_DB_PATH`、`workers=1`；`pnpm e2e` 前置 `pnpm build`。
- 23/23 通过，`runtime:ready` 握手噪声消失。官方边界不变：server build **不覆盖**原生窗口/菜单/托盘/原生对话框，exe 级冒烟仍需人工。

### 9.5 阶段 4：打包 / CI / 文档

**v3 CLI 要求 Taskfile（评估阶段未预见的硬门槛）**：`wails3 build` 内部是 `wails3 task build`，原目录无 Taskfile 直接报 `No Taskfile found at backend`。官方 Taskfile 假设 Go 模块根即项目根、`frontend/` 是其子目录，而本项目前端在 `../frontend`，故**不套用官方多平台模板**，改自建精简 `backend/Taskfile.yml`（`build` / `build:frontend` / `package` / `create:nsis:installer` / `run`）。v2 的 `scripts/build.ps1` 改为薄封装，构建逻辑只有一份。

**NSIS 目录迁移**：v3 官方路径是 `build/windows/nsis/project.nsi`（v2 为 `build/windows/installer/project.nsi`）。`wails_tools.nsh` 的 `wails.*` 宏名与 v2 完全一致（含 `wails.associateCustomProtocols` / `wails.unassociateCustomProtocols`），故「LZMA 固实压缩 / 63MB 物料随装 / 卸载保留 `namer.db`+WAL+SHM+logs / 简体中文优先」四段定制原样移植；新增 `!define PRODUCT_EXECUTABLE "namer-desktop.exe"` 与 `INFO_COPYRIGHT`，`OutFile` 改 `..\..\..\bin\...`，UTF-8 BOM 保留。`build/config.yml` 取代 `wails.json` 承载产品元信息。

**go-task 坑**：`dir` 只在**任务级**生效，写在 `cmd` 对象里会被忽略，首次实测报 `makensis: Can't open script project.nsi`；拆出独立任务 `create:nsis:installer`（`dir: build/windows/nsis`）后通过。

**物料瘦身**：`File /nonfatal /x "*_stats.json"` 排除 4 个运行期不读取的统计 JSON（约 30MB）；安装器 install data 54.9MB、LZMA 压缩后 11.85MB（21.5%）。

**CI 硬约束（本轮新增发现）**：v3 的 linux 平台代码（`menu_linux.go` / `webview_window_linux.go`）依赖 GTK/WebKit，`GOOS=linux CGO_ENABLED=0 go build ./...` 报 `undefined: pointer`。解法是全局 `GOFLAGS=-tags=server`（框架编译无窗口实现，业务代码不变）；已实测 linux 下 `go build -tags server ./...` 与 `go test -tags server -c` 测试二进制链接均 exit 0，Windows 下 `go test -tags server ./...` 全绿。注意这与 `verify-quality.sh` 的 `SKIP_SERVER=1`（跳过已退役 HTTP E2E 段）语义不同。

CI 最终三个 job：`quality-fast`（Go 快路径门禁）、`quality-bindings`（`go install wails3@v3.0.0` → 重新生成 TS 绑定 → `git diff --exit-code -- frontend/bindings`，防 `$Call.ByID` 方法 ID 漂移）、`quality-frontend`（tsc/eslint/vitest）。原 `quality-e2e` 仍引用已退役的 `cmd/server`，且因仓库只入库最小物料而永远跳过，已移除；Playwright 23 例需要全量 `data/` 物料与 Chromium，继续由本地 `cd frontend && pnpm e2e` 承担。

### 9.6 实测产物与未覆盖项

| 项 | 结果 |
|---|---|
| `wails3 build` | exit 0，`backend/bin/namer-desktop.exe` = 19,508,224 B |
| `wails3 package` | exit 0，`backend/bin/namer-amd64-installer.exe` = 11,852,207 B |
| `GOOS=linux CGO_ENABLED=0 go build -tags server ./...` | exit 0 |
| server build E2E | Playwright 23/23 |
| Go / 前端门禁 | 全绿（见 CHANGELOG `[2026.11.16.0]`） |

- **未做（需人工 GUI 冒烟）**：干净目录安装 → 收藏/历史落库 → 卸载重装数据延续；原生菜单/托盘/对话框。exe 双击启动已于 2026-11-16 复核（见 §9.7）。
- **pin 纪律**：Go module / `wails3` CLI / `@wailsio/runtime` 三者同为 `3.0.0`；任一升级须同步三者并重新生成绑定，否则 `quality-bindings` 会拦截。

### 9.7 exe 双击静默退出（人工 GUI 冒烟暴露，已修复）

- 现象：`backend/bin/namer-desktop.exe` 双击 / 命令行运行后无窗口、无报错；受控探针捕获 `exit 1` + `错误: 未找到传统文化数据目录，请通过 DATA_DIR 环境变量或 Options.DataDir 指定`。
- 根因：`-trimpath` 把 `runtime.Caller(0)` 的源文件绝对路径抹成模块相对路径，dev 侧基于源码位置的 `paths.FindDataDir` 在产物中必然失效；`FindDataDirFromExe` 又只查 exe 同级 `data/`，直接运行 `backend/bin` 下的 exe 时该目录不存在 → 所有目录来源落空，`run()` 在日志初始化之前退出。`-H windowsgui` 使 GUI 子系统无控制台，stderr 不可见，于是「静默」。
- 修复：新增 `paths.FindDataDirFromExeWithParents`（以 `os.Executable()` 为锚点上溯 ≤12 层，`namer.json` 为标记）供 `bootstrap.findDataDirFromExe` 使用；`main.go` 启动失败分支增加 exe 同级 `startup-error.log` + Windows `user32!MessageBoxW` 错误框（按平台分文件，非 Windows 空实现）。详见 CHANGELOG `[2026.11.16.2]`。
- 复核：重新 `wails3 build` 后无参数启动创建 `起名 NameMaster` 主窗口（首跑播种约 80 秒）；项目外孤立目录启动弹出错误框并落盘日志。
- **遗留体验**：安装器不预置 SQLite 库，首次启动播种约 80 秒内没有窗口 / 进度提示，易被误判为「没反应」；后续可预置种子库或在装配前显示进度。
