# 44 - CLI/GUI 全业务功能端到端测试验证报告

> 测试日期：2026-11-08
> 测试环境：Windows 10.0.26100 x64 / Go 1.25.10 / Node 22.23.2 / pnpm 12.14.0
> 服务拓扑：后端 `backend/cmd/server`（127.0.0.1:8080，SQLite 权威库 `backend/namer.db`）+ 前端 `pnpm dev`（127.0.0.1:3000）
> 参考文档：docs/35（e2e 框架接入指南）、docs/22（E2E 巡检工具）

## 一、测试范围与结论总览

| 层级 | 工具/用例 | 结果 |
| --- | --- | --- |
| CLI 端到端 | namer-cli 全参数矩阵 14 用例（自编脚本）+ 既有 go test | **14/14 通过**（含 2 例修复后复验） |
| API 巡检 | `tools/e2e_check -strict`（18 项检查） | **18/18 通过** |
| API 契约 | tester-army/e2e `naming-api.e2e.ts` | **1/1 通过**（2.1s） |
| API 全业务冒烟 | 24 条注册路由 Python 冒烟 23 用例（含错误处理契约） | **23/23 通过** |
| GUI 既有用例 | `naming-ui.e2e.ts`（1）+ `naming-ui-conditions.e2e.ts`（4） | **5/5 通过** |
| GUI 全业务页面扩展 | `naming-ui-pages.e2e.ts`（7，本报告新增） | **7/7 通过** |
| 回归验证 | go test（middleware/namer-cli）、vitest 122 例、GUI e2e 12 例、e2e_check 复跑 | **全部通过** |

**发现并修复缺陷 3 个、文档偏差 1 项**（详见第四节），全部以真实 E2E 链路复现并修复后实证闭环。

## 二、环境准备记录

1. 物料库状态检查：`materials status` 发现 12 项物料齐全（namer 8134 行），但 `classics_extracts` 中 **shijing/chuci 提取数为 0**。执行 `materials import --data ../data --db ./namer.db` 重新导入后补齐（shijing 2734 条、chuci 3092 条，全部 17 个经典合计 46693 条）。
2. 服务启动：后端以 `backend/namer.db` 权威库启动（`/healthz` ok、`/readyz` ready/db connected）；前端 `NEXT_PUBLIC_API_URL=http://127.0.0.1:8080/api pnpm dev`。
3. 冒烟确认：`POST /api/v1/names/generate`（王/male/2024-1-15-12）HTTP 200、50 候选名。
4. 编码陷阱：Git Bash 终端下 curl `-d` 中文变 GBK 乱码导致"姓氏只能包含中文字符"400——为终端显示问题非服务缺陷，测试一律改用 UTF-8 Python 客户端。

## 三、分层测试结果

### 3.1 CLI 端到端（namer-cli，复用 Web 同一领域链路）

矩阵覆盖：text/json 输出、TOML 配置加载、命令行 > TOML 优先级、女宝/单字名、五行偏好、典籍来源 shijing（验证补齐后的提取链路）、笔画区间、避讳长辈（Top 全量断言不含 伟/秀/英）、字辈 middle 位置（候选全部以「李承」开头）、复姓欧阳、错误处理（缺姓氏/非法性别/非法年份均非零退出且中文错误信息）。

既有集成测试 `TestCLIBinaryJSONOutputPinyinFilled`（编译真实二进制 + JSON 结构 + 拼音回填断言）通过；新增回归用例见第四节缺陷 1。

### 3.2 API 端到端巡检（e2e_check -strict，18 项全过）

要点摘录：
- 8 场景耗时全部 HTTP 200 且候选名非空（双名基线/单名/女宝/论语来源/显式五行/避讳/复姓/笔画区间）
- 用字质量：推荐榜零门禁字命中、零三级/表外字混入
- 参数生效性：避讳字零命中、`source_classic` 与基线 Top10 差异生效
- 同步/异步一致性：异步 50 名、喜用神一致（木 水 金）、Top10 交集 10/10、bazi 字段集一致（21 个）
- 可复现性：6 次连发 Top10 完全一致
- 并发压测：16×2 轮 32/32 成功，加速比 3.79×，p95 7.44s 受控
- 测名与生成同源同分（85.20）；探索模式换一批零交集、会话淘汰后 400 提示重新生成

### 3.3 API 全业务冒烟（23 用例，按实际注册路由）

覆盖 9 大类：健康检查（/healthz /readyz）、生成（同步/异步/任务轮询/探索换一批/测名）、历史（增查删）、收藏（增查删）、统计（重名率/排行）、PDF 报告（二进制流响应）、黄历/农历/农历月历、规范汉字分组（29 组）、错误处理契约（空姓氏 400/不存在任务 404/缺参 400）。写操作测试数据全部自动清理。

### 3.4 GUI 端到端（tester-army/e2e，Playwright Chromium，零模型）

既有用例 5 个：主链路（首页填表→生成→结果页候选名）、空姓氏校验拦截、经典来源 toggle 回显、字辈「承」前缀、单字名。

新增扩展用例 7 个（`frontend/tests/naming-ui-pages.e2e.ts`）：
1. 导航栏可达性（首页→测名→黄历，aria-label 点击）
2. 测名链路（填姓名→开始测名→评分 + 八维评分依据）
3. 黄历页（今日黄历 + 宜/忌渲染）
4. 重名率查询（查询→重名人数 + 热门名字榜）
5. 收藏链路跨页（生成→心形收藏→收藏页出现→删除→空态）
6. 对比链路跨页（生成→选 2 个加入对比→对比页综合分析）
7. 历史页与 JSON 查看器可达（iframe 承载页以主导航断言）

## 四、发现并修复的缺陷（3 个）

### 缺陷 1（P2）：CLI `-config` + `-format json` 时提示信息污染 stdout JSON 流

- **现象**：`已加载配置文件: ...` 与 JSON 混在同一路径输出，`namer-cli -format json | jq` 类管道消费直接解析失败。
- **根因**：`main.go` 用 `fmt.Printf` 将加载提示打印到 stdout。
- **修复**：改走 `fmt.Fprintf(os.Stderr, ...)`；新增回归测试 `TestCLIBinaryConfigFileDoesNotPolluteJSONStream`（编译真实二进制断言 stdout 纯 JSON、提示在 stderr）。
- 位置：`backend/cmd/namer-cli/main.go`、`integration_test.go`。

### 缺陷 2（P0，核心功能不可用）：首次收藏永远静默失败——UI 显示已收藏、后端无数据

- **现象**：结果页点击心形后变红、toast 报"操作成功"，但**收藏页为空**、无任何 POST /favorites 请求、数据库无记录。
- **根因**：`result/page.tsx` 的 mutation 链路竞态——`onMutate` 的乐观更新先把**无 id 的乐观记录**写入 react-query 缓存，随后 `mutationFn` 重新读缓存判断 existing，读到刚写入的无 id 记录 → `existing.id` 为空 → 走入静默 no-op 分支返回，POST 永不发出。
- **定位过程**：e2e trace 显示 tap 成功但无网络请求 → 后端日志无 POST → SQLite 无记录 → 注入 fetch 记录器复现 → 加临时调试日志确认链路执行到 mutationFn start 后无 POST。
- **修复**：把 existing 判断从 mutationFn 内部挪到 `mutate()` 调用前求值（乐观更新前快照），以 `{ name, existingId }` 作为 variables 传入；onMutate/onError 同步适配 variables 类型。
- 位置：`frontend/src/app/result/page.tsx`。

### 缺陷 3（P0，数据归属错乱）：并发首访 Session Cookie 竞态——收藏落入"孤儿会话"

- **现象**：缺陷 2 修复后 POST 发出且落库，但同一浏览器收藏页仍查不到（后端按 session 隔离返回空）。
- **根因**：结果页并发发出多个 API 请求（GET favorites / POST history 等），各请求均无 cookie 时服务端 `EnsureSessionID` 为每个请求各生成一个新 `nm_sid` 并各自 SetCookie，浏览器只保留最后一个——先完成的写请求（收藏）落到被覆盖的"孤儿会话"。
- **修复**（前后端各一）：
  - 前端 `lib/api.ts`：新增 `getSessionId()`（localStorage 持久化 32 位 hex，key `nm_session_id`），`fetchWithRetry` 统一注入 `X-Session-ID` 头（后端头优先级高于 Cookie），同一浏览器永远同会话。
  - 后端 `middleware.go`：CORS `AllowHeaders` 默认列表两处（CORSConfig 与 fallback 字符串）放行 `X-Session-ID`，否则自定义头触发 preflight 失败（实测 NetworkError）。
- **验证**：清 sid 重走全链路——生成→收藏→DB `session_id` 与后续查询一致→收藏页显示"李勋炜"→删除→空态，端到端闭环。
- 位置：`frontend/src/lib/api.ts`、`backend/internal/infrastructure/middleware/middleware.go`。

### 回归验证（修复后）

- `go vet` + `go test ./internal/infrastructure/middleware/ ./cmd/namer-cli/` 通过
- 前端 vitest **122/122** 通过、`tsc --noEmit` 0 错误
- GUI e2e 全套 **12/12**（既有 5 + 新增 7）通过
- `e2e_check -strict` 复跑 **18/18** 通过

## 五、文档偏差与遗留建议

1. **docs/02-API接口文档.md 与实际路由严重不符**（12 个文档接口实际不存在）：`GET /names/:id`、`POST /bazi/analyze`、`GET /yijing/hexagram(/:id)`、`GET /zodiac(/:animal)`、`POST /history/batch`、`GET /favorites/check`、`POST|DELETE /favorites/batch`、`GET /namestat/:name/province`、`POST /report/html`、`POST /feedback`、`POST /feedback/request`、`GET /feedback/algorithm-performance`、`POST /names/generate/analysis`（已下线）。八字/卦象/生肖由前端本地 lib（hexagram.ts 等）实现，后端无对应路由。建议按实际路由重写 API 文档（本次未改动文档正文，仅登记）。
2. 生成成功后前端自动 `POST /history` 保存记录（浏览器实测发现），历史页非空态；API 文档未记载该自动行为。
3. 结果页直接刷新显示"没有可展示的起名结果"为设计内行为（页面文案已明示，结果存于内存 store）。
4. `materials import` 后 shijing/chuci 提取缺失的原因（源文件与 materials.lock 指纹比对）值得单独排查，避免其他环境复现同类问题。
5. IAB 内嵌浏览器对带 CSS 动画元素的高层 locator click 有 actionability 超时现象（DOM click 可用）；tester-army/e2e（Playwright Chromium）不受影响。

## 六、测试产物清单

| 产物 | 路径 |
| --- | --- |
| GUI 全业务扩展用例 | `frontend/tests/naming-ui-pages.e2e.ts`（7 用例） |
| CLI 回归测试 | `backend/cmd/namer-cli/integration_test.go`（新增 TestCLIBinaryConfigFileDoesNotPolluteJSONStream） |
| API 冒烟脚本 | 临时目录（`/tmp/api_smoke.py`，23 用例，可复跑） |
| CLI 矩阵脚本 | 临时目录（`/tmp/cli_e2e/run.sh`，14 用例，可复跑） |
| e2e 运行报告 | `frontend/.e2e/report.json` |
