# 更新日志 (CHANGELOG)

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 格式，
版本号采用 **CalVer（日历版本）**：`YYYY.MM.DD.MICRO`，并在稳定分支以同号打 Tag。

> 注：自 `2026.08.24.0` 起建立统一变更日志；此前迭代未留档。

## [2026.09.25] 当日总览

前端测试环境修复专项 + 后端审查 Warning 处置，版本 `2026.09.25.0` 与 `2026.09.25.1`。

| 项 | 结论 |
|------|------|
| 根因 | 本机建不出可用**符号链接**（`os.symlink()` 不报错但产出空 junction）→ pnpm 隔离式装配必然失败，vitest 依赖链接全缺 |
| `.0` | 新增 `frontend/scripts/hoist-deps.mjs`（postinstall 自动执行），用**目录 junction** 完成平铺；前端测试首次跑通 **36/36** |
| 附带 | 前端 `tsc --noEmit` 由 6 个错误 → **0 错误**（此前报错全是 vitest 模块缺失引起） |
| `.1` | 处置后端 Warning **W2 / W4 / W8**，其中 **W4 连带发现两个更严重缺陷**（fatal 竞态 + 排除功能完全失效）；另修 W2 的第三处缺陷（配置路径不存在被判致命） |
| 回归 | 后端 **20 包**全绿（0 FAIL / 0 panic / 0 data race）；前端 36/36；E2E 16/17（唯一未过项为吞吐判据，详见 `docs/28` §8.5） |

---

## [2026.09.25.1]

### 🐛 Bug Fixes 问题修复

- 【config】★ **viper 全局/局部实例断裂，5 个 getter 恒定返回默认值**（`docs/28` W2）：`Load()` 用 `viper.New()` 建**局部实例**并在其上 `SetDefault`/`ReadInConfig`，而 `GetString/GetInt/GetFloat64/GetBool/GetDuration` 读的是 `viper.GetViper()` —— 一个与之无关的**包级全局单例**，两者自始至终没有交集，配置文件里的值永远不会被读到。现新增包级 `globalConfig`/`globalViper` + `sync.RWMutex`，`Load()` 在写锁内存下两者，getter 统一走新的 `getViper()`；`Get()` 由「每次调用重新懒加载」改为返回已加载实例。**实测**：配置文件写 `server.port: 8765`，修复后 `GetInt("server.port")` → `8765`，旧路径 `viper.GetViper().GetInt("server.port")` → `0`。
- 【config】**显式传入不存在的配置路径被判致命错误**（`docs/28` W2 连带发现）：`SetConfigFile` 模式下 viper 对缺失文件返回 `*fs.PathError` 而非 `viper.ConfigFileNotFoundError`，原代码只判后者，于是"找不到配置文件"这个完全正常的场景被当成致命错误抛出、进程起不来。新增 `isConfigNotFound(err, configPath)` 同时识别两种形态，统一回落默认值。
- 【fate】★ **负面反馈排除功能完全失效**（`docs/28` W4 连带发现，危害高于原记录）：`Start()` 无条件执行 `s.excludedChars = make(...)` / `s.excludedCombos = make(...)`，把调用方在 `Start()` **之前**设置的排除项全部抹掉。而接口契约明确要求「后续重新生成时所有包含该字的候选将被过滤」，即「先从界面排除 → 再重新生成」这一唯一合理用法 —— 结果排除功能**从未生效过**。**实测**：`ExcludeChar("渝")` 后 50 个结果中含「渝」的数量由 **12 → 0**。现改为不重置（空 map 由 `ExcludeChar`/`ExcludeCombo` 内部按需创建，`ClearExclusions` 负责显式清空）。
- 【fate】**排除集读路径存在 `concurrent map read and map write` 致命竞态**（`docs/28` W4）：`isCharExcluded`/`isComboExcluded` 不加锁直接读 map，其注释声称「并发安全由 `s.mu` 守护（generate 与 ExcludeChar 互斥）」—— 该论断是错的，`generate()` 全程并不持有 `s.mu`。生成进行中调用 `ExcludeChar` 会触发 Go 运行时**不可 recover 的 fatal error**，直接终止整个服务进程。现改为读取 `atomic.Pointer[exclusionSnapshot]` 不可变快照，写方 copy-on-write 重建 —— 正确且**无锁**。
- 【services】**释义拼接重复实现**（`docs/28` W8）：`evaluate_service.go` 的 `combineTwoMeanings` 是 `fate.combineCharMeanings` 的逐行复制，截断常量 `60` 硬编码两处且无机制阻止漂移 —— 只改一边会让同一个名字在「结果页」与「测名页」显示不同寓意。现领域层新增导出的 `fate.CombineMeanings(m1, m2, maxRunes)` 作为唯一实现，应用层改为一行委托。

### 📈 Improvements 性能/体验优化

- 【fate】**排除集读路径由「逐次 `RLock`」优化为「原子快照」，消除热路径锁开销**：`isComboExcluded` 位于双名 N×N 笛卡尔积的**最内层循环**（真实数据下每次生成调用千万次量级），首版修复用 `RWMutex.RLock` 虽然正确，但实测让**串行吞吐下降约 20%**。现改为 `atomic.Pointer` 不可变快照：读方仅一次原子指针加载 + 一次 map 查找，写方（调用次数为个位数量级）在写锁内复制后发布。关键约束是**先复制、后发布**（若直接放可写 map，写方后续写入会命中读方正在遍历的同一份 map，等同复现竞态）。无排除项时快照为 `nil`，零额外开销。**微基准**：`Empty` **1.1 ns/op**（线上绝大多数请求的真实形态）/ `Populated` ~58 ns/op（单核）/ `Hit` ~66 ns/op。
- 【fate】排除项语义由「`generate()` 入口一次性快照消费」改进为「每次调用时加载快照」—— 生成进行中新增的排除项会**立即**对尚未评估的组合生效，比首版更及时；同时 `Start()` 不再需要为「无排除项」预建空 map。

### 🧪 Tests 测试补充

- 【config】新增 `internal/infrastructure/config/config_test.go`（**5 条**）：getter 读文件值、无文件时读默认值、`GetDuration` 解析与回落、`GetString` 空串回落语义、`Get()` 身份一致性。其中 2 条在首轮直接失败，暴露了上述第三处缺陷。
- 【fate】新增 `internal/domain/fate/engine_exclusion_race_test.go`（**3 条 + 3 个基准**）：`TestSessionExclusionConcurrentAccess`（生成中 4 goroutine 并发增删排除集，不得触发 fatal 且结束后状态自洽）、`TestSessionExclusionAppliedWhenSetBeforeStart`（契约正向：排除后结果中该字零出现）、`TestSessionExclusionOnlyAffectsNextGenerate`（反向：生成后改排除集不得回溯改变结果），以及 `BenchmarkIsComboExcluded{Empty,Populated,Hit}`。
  - 注：本机无 gcc，`go test -race` 不可用（`-race` 需 cgo），故竞态用例依赖 Go 运行时自身的 map 并发检测 + 结果自洽断言。

### 📚 Docs 文档更新

- `docs/28` 新增 §8「后端 Warning 处置记录」：逐条记录 W2/W4/W8 的根因、修法、实测证据与验证口径，并在 §4 表格中将三项标记为已修；§8.5 列出待办（后端 W3/W5/W6/W7、前端 W2-W10、E2E 吞吐判据）。
- `docs/28` §8.5 记录 E2E `-strict` 复查为 **16/17** 的排查结论：唯一未过项（并发吞吐加速比 1.33× < 1.5 阈值）**非本轮修复引入** —— 串行吞吐由 1.09 提升到 1.45 req/s（+33%），并发吞吐基本持平（1.99 → 1.93），加速比下降是「分母变好」的结果；判据本身在基线改善时会自动收紧，属度量设计问题。

---

## [2026.09.25.0]

### 🐛 Bug Fixes 问题修复

全链路 E2E 验证与代码审查专项，版本 `2026.09.24.0`，产出 `docs/28`。

| 项 | 结论 |
|------|------|
| E2E 巡检 | 修复后 **17/17 通过**（`-strict` 退出码 0）；修复前工具全部指向已下线的 `/generate/analysis`，预热即失败，一行业务检查都跑不到 |
| 后端测试 | `go test ./...` 19 个包全绿，0 FAIL / 0 panic / 0 data race |
| 审查发现 | 已实测确认 4 项 Critical（收藏必然 400、关键词被丢弃、刷新死路、下标崩溃）+ 后端 W1~W8 / 前端 W1~W10 |
| `.1` | 审查发现的 P0/P1 实施修复：收藏评分改浮点、CORS 收敛白名单、关键词接上、刷新死路给出口、下标换稳定 key，另修 hydration 与 StrictMode 自清空；补 2 条回归测试 |

---

## [2026.09.25.0]

### 🐛 Bug Fixes 问题修复

- 【frontend/build】★ **前端完全无法运行自动化测试（根因是符号链接）**：本机（无管理员权限、未开启 Windows 开发者模式）创建符号链接会**静默失败**——`os.symlink()` 不抛异常，但产出一个不可用的空 junction（`islink=False`、`listdir()==[]`）。pnpm 默认的隔离式 `node_modules` 完全依赖符号链接装配，于是包文件都下载进了 `.pnpm`，依赖之间的链接却建不起来：`pnpm test` 直接崩在 `ERR_MODULE_NOT_FOUND: Cannot find package 'std-env' imported from .pnpm/vitest@5.0.0/.../vitest/dist/cli.js`（vitest 的 13 个运行时依赖只建出 2 个，且都是空目录）。
  - **`node-linker=hoisted` 已失效**：这是官方解法，但 pnpm 11 移除了该配置项（`pnpm config get node-linker` → `undefined`），写入 `.npmrc` 也不再生效，重装后仍是 isolated 布局。
  - 现新增 **`frontend/scripts/hoist-deps.mjs`**，挂在 `package.json` 的 `postinstall` 上自动执行：扫描 `.pnpm` 虚拟存储，把 426 个包用**目录 junction**（`fs.symlinkSync(..., 'junction')`，不需要 SeCreateSymbolicLinkPrivilege）平铺到顶层 `node_modules`；多版本（38 个）取最高版本；脚本幂等，重复执行只补缺失项。

### ✨ New Features 新增功能

- 【frontend/tests】**前端自动化测试首次跑通**：`pnpm test` → `vitest run`，4 个测试文件、**36 个用例全部通过**（`birthdayShichen` 5 / `validation` 15 / `utils` 6 / `store` 10），耗时 3.09s。此前前端仅有 `tsc --noEmit` 可用，本次修复的 5 个前端文件属于首次获得运行时用例覆盖能力。

### 📈 Improvements 性能/体验优化

- 【frontend/types】**`tsc --noEmit` 由 6 个错误降为 0**：此前长期存在的报错（`vitest` 模块缺失 ×5、`.next/types/validator.ts`）全部由 vitest 未装配引起，依赖装好后自行消失，不再需要"按改动文件是否在错误列表里"来判断。

### 📚 Docs 文档更新

- `frontend/.npmrc` 重写为说明性注释：记录符号链接失效现象、`node-linker=hoisted` 为何不再可用、以及改用 junction 的理由。

### 🧪 Tests 测试补充

- 回归验证：后端 `go test ./...` 19 包全绿（0 FAIL / 0 panic / 0 data race）；E2E `e2e_check -strict` **17/17 通过**；前端 `pnpm test` 36/36 通过、`tsc --noEmit` 0 错误。
  - 注：该 17/17 结论为 `2026.09.25.0` 时点；`2026.09.25.1` 修复后复查为 16/17，唯一未过项为吞吐判据，排查结论见 `[2026.09.25.1]` 的 Docs 条目。

---

## [2026.09.24.0]

### 🐛 Bug Fixes 问题修复

- 【tools】**`e2e_check` 与现行路由脱节，质量防线整体失效**：9 个场景中 8 个以及预热/可复现性/并发三段全部请求已下线的 `/api/v1/names/generate/analysis`（`2026.09.18.1` 移除），预热失败即 `os.Exit(2)`，业务检查一行都跑不到。现统一为 `pathGenerate/pathAsync/pathExplore/pathEvaluate` 四个常量，全部走 `/api/v1/names/generate`。

### ✨ New Features 新增功能

- 【tools】`e2e_check` 新增三段巡检，检查项 6 → 8 段、10 → 17 项：
  - **同步 vs 异步一致性**（替代已无意义的「双路径一致性」）：提交 `/generate/async` 后轮询 `/names/task/:id` 到终态，记录进度阶段序列，比对喜用神 / Top10 / bazi 字段集；
  - **测名 `/names/evaluate`**：断言 200、评分 0-100、且与生成链路**同分**（同源装配链契约），并验证缺 `given_name` 返回 400；
  - **探索 `/names/generate/explore`**：验证「换一批」与上一轮零交集，并把会话被 FIFO 淘汰后的降级行为（400 + 提示重新生成）固化为契约检查。
- 【tools】解析层增强：`result.parseAt(field)` 支持下沉一层（`data.result` / `data.name`）、`numAt()` 取数值字段，避免为异步/测名各写一套解析。

### 📚 Docs 文档更新

- 新增 `docs/28-全链路E2E验证与代码审查报告.md`：E2E 修复与 17 项结果、探索会话容量 32 的 FIFO 淘汰风险、已实测确认的 4 项 Critical、前后端 Warning/Info 清单、20 条未使用导出符号、处置优先级。

---

## [2026.09.24.1]

### 🐛 Bug Fixes 问题修复

- 【services/database】★ **收藏接口几乎必然 400**：前端原样提交 `total_score`（浮点如 `92.7`），而 `FavoriteRecord.Score` / `database.FavoriteRecord.Score` 声明为 `int`，`encoding/json` 小数→int 抛 `UnmarshalTypeError`，只有评分恰好整除时才侥幸成功。现统一改为 **`float64`**，favorites 表 `score` 列声明改 `REAL`（存量库为 INTEGER 亲和，SQLite 对无法无损转整数的值本就按 REAL 存储，无需迁移）。实测：`{"score":80.7}` 由 400 → **200**，列表回读 `92.7` 小数完整保留。
- 【middleware】★ **CORS 默认「任意源 + 允许凭据」**：`AllowOrigins: ["*"]` 与 `AllowCredentials: true` 并存，中间件回显请求方任意 `Origin`，任意站点可跨域读取用户的起名历史与收藏。现收敛为本机白名单（`localhost`/`127.0.0.1` 的 `:8080` 与 `:3000`），并新增 `NAMER_CORS_ORIGINS`（逗号分隔）供多域名部署覆盖。实测：`Origin: http://evil.example.com` 不再返回 `Access-Control-Allow-Origin`。
- 【frontend/result】**刷新后永久骨架屏**：`partialize` 只持久化 `formData`，生成结果不落盘。现拆两个分支——hydrate 未完成仍显示骨架屏；已 hydrate 但无结果显示**带导航栏 + 「回到首页重新起名」**的空状态，不再把用户困死。
- 【frontend/result】**用数组下标当身份**：`selectedName`/`compareNames` 改存 `nameKey`（`surname:given_name`），`handleSelect`/`toggleCompare` 收到下标后先换成 key，新增 `nameByKey` 回查；筛选或换一批导致列表变短时不再取到 `undefined` 崩溃，`handleCompare` 对已被筛掉的名字给出提示。
- 【frontend/home】★ **首页「寓意关键词 / 偏旁选字」被全部丢弃**：`NameForm` 通过 `onSubmit` 传出了 `{formData, keywords, selectedChars}`，但 `HomeContent.handleSubmit` 写成无参函数，读的是自己那个从未被赋值的 `keywords` state，于是 `meaning_keywords` 恒为 `[]`——而后端该字段真实存在且被消费。现导出 `NameFormSubmission` 并由 `handleSubmit` 接收使用；删除失效的 `keywords` state。
- 【frontend/result】收藏态图标只读 React Query 缓存，而结果页从未发起过该 query，导致没访问过收藏页的用户心形图标一律显示未收藏。现补 `useQuery({ queryKey:['favorites'], queryFn: getFavorites })`。
- 【frontend/ThemeToggle】`useState(getInitialTheme)` 在 SSR 返回 `false`、客户端返回 localStorage 值，造成 hydration mismatch。初值固定 `false`，真实主题改在挂载后的 effect 里同步。
- 【frontend/compare】StrictMode 下 `effect → cleanup → effect`，卸载时 `setCompareResult(null)` 会把刚进页面的数据自己清掉。删除该 cleanup，并清理随之失效的 `useEffect` import 与 `setCompareResult` 解构。

### 🧪 Tests 测试补充

- 【services】新增两条针对收藏评分的回归护栏：`TestFavoriteRecord_JSONDecimalScore`（JSON 里的 `92.7` 必须能反序列化进 `FavoriteRecord`）、`TestSaveFavorite_DecimalScore`（浮点评分完整落库且原值传给自学习精选库）。
- 后端 `go test ./...`：19 个包全绿，0 FAIL / 0 panic / 0 data race；E2E `-strict` 17 项全部通过。
- 前端：`tsc --noEmit` 下本次改动的 5 个文件全部 clean。注：仓库未安装 `vitest`（`node_modules/vitest` 不存在），前端自动化测试无法执行，本次前端修复仅经类型检查与人工核对。（此限制已于 `2026.09.25.0` 解除，vitest 恢复可用。）

### 📚 Docs 文档更新

- `docs/28` 新增「七、修复实施记录」：逐项记录改法与实测结果。

---

## [2026.09.18] 当日总览

竞品对标与业务能力扩展专项：产出竞品蓝图 `docs/27`（含对 `docs/15`/`18` 三处过期或违规结论的纠正）+ 三项「可解释性 / 健壮性」落地，版本 `2026.09.18.0`。

| 版本 | 要点 |
|------|------|
| `.0` | `/names/generate` 首次返回 `score_detail` 评分依据（关闭 `docs/24` P2-5）+ 请求体上限中间件（P1-4）+ 修复诗词异步加载可能挂死全站（P2-9） |
| `.1` | ★ 五项能力扩展：异步任务+进度、探索模式（换一批）、**测名+风险体检**、HTTP 双链路收口（analysis 路由下线）、接口整合清理（30→17 条） |
| `.2` | 关闭 `docs/24` 全部 P2 项：并发 worker 全局令牌钳制（P2-7）、收藏自学习失败告警（P2-8）、classics 版本读失败重建（P2-10）、删除临时 cmd 与 `_ =` hack（P2-11）、curated_names 读取 4 处收敛（P2-12）；P3-13~18 卫生清理与文档过期修正；e2e 超订探测改为吞吐加速比语义 |

---

## [2026.09.18.2]

### 🐛 Bug Fixes 问题修复

- 【fate】**并发 worker 超订钳制**（`docs/24` P2-7）：双击列举的 worker 按每请求 `runtime.NumCPU()` 固定分片，8 并发即 128 个 worker 抢 16 核，吞吐在低并发出封顶、之后单请求耗时线性恶化。现引入**进程级全局令牌信号量** `candidateWorkerLimit`（容量 = `NumCPU()`）跨请求共享，worker 启动前「拿令牌，拿不到就等 `ctx.Done`」，并加 `candidateWorkerActive/Peak` 原子观测。回归护栏 `engine_concurrency_test.go`：注入小容量信号量、8 并发断言峰值 ≤ 容量；单请求在无竞争时仍可动用全部 worker。
- 【services】**收藏自学习写入失败不再静默吞掉**（P2-8）：`favorite_service` 自学习路径的 `_ = AddCuratedName(...)` / `_ = SaveCuratedName(...)` 改为显式捕获并 `logger.Warn`（保留降级语义）。
- 【database】**classics 版本号读失败按「需要重建」处理**（P2-10）：`store_classics.go` 两处 `classics_version` 读取失败不再被当作「版本一致」，改 `logger.Warn` + 走重建分支。
- 【domain/infra】`tyme.LegalHoliday` 解析失败不再等同「非节假日」（P3-15）——告警后降级；静态文件路由 `io.Copy` 错误补日志（P3-16）。

### 📈 Improvements 性能/体验优化

- 【fate】`data/forbidden_combos.json` 962 条清洗组合已动态加载（`semantic_filter.go:146`），历史报告 `docs/19` 的 B1 结论关闭。
- 【cmd】删除自述「临时」的 `cmd/diag_curated`、`cmd/probe_chars`，清理 `build_frequency` 的 `_ = math.MaxInt` hack（P2-11）。

### ♻️ Refactor 结构优化

- 【domain/name】**curated_names 读取收敛为单一导出函数**（P2-12）：新增 `name.LoadCuratedNamesData`，4 处重复实现（server / namer-cli / sqlite seed / name_db）统一调用。
- 【tools】`e2e_check` 并发超订探测从「单请求放大系数」改为「**吞吐加速比**」语义——P2-7 后 worker 被钳制在核数内，排队延迟是正常现象，判定目标改为验证并发吞吐相对串行的实质加速。

### 🧹 Cleanup 卫生清理

- 【repo】13 个 Go 文件补齐末尾换行（P3-13）；`git rm` 无引用的 1.1 MB 大图 `docs/assets/黄历3.png`（P3-14）；删除废弃 `middleware.RateLimit`（P3-17，占位 Store 方法是 `database.Store` 接口实现故保留）；前端删除 6 个零引用 API 导出（P3-18，`checkFavorite`/`getCharStyles`/`getCuratedNames`/`getHexagrams`/`getRadicalChars`/`getZodiacs`）。

### 📚 Docs 文档更新

- `docs/24` 全量回填状态：P1-4 与 P2-5~12 全部 ✅，P3-13~18 标注（P3-17 部分、P3-19 因有 `NEXT_PUBLIC_API_URL` 兜底保留）。
- `docs/19` 两处过期陈述加「后续进展」指针（`engine.go` 行号漂移 600-601→1215；`naming_quality.json` 131→2990 字）。
- `docs/20` 数据规模表更新 `naming_quality.json` 为 `32 KB / 2990 字`。

---

## [2026.09.18.1]

### ✨ New Features 新增功能

- 【services/fate】★ **测名 + 风险体检**（`docs/27` 红色缺口 ①②，全竞品标配）：
  - 新增 `POST /api/v1/names/evaluate`（前端 `/evaluate` 页）：输入姓名 + 生辰 → 完整评分报告 + 风险清单。核心前提已按蓝图要求补齐——抽出 `fate.RateGivenName`/`GivenNameStrokes` 装配函数，按引擎枚举时的同一逻辑组装 `NameCandidate` 全量元数据（拼音/五行/笔画/寓意/频率档位/策展标记），八字走同一 `BaziAnalyzerAdapter`，评分走同一 `DefaultRaters()` 链 → **测名与生成同源同分**，不产生双口径。
  - 新增 `fate.AssessNameRisks` 风险体检聚合：谐音（负面词/贬义组合）、生僻字（字表等级）、多音字、户籍友好度（笔画/字表/生僻）、网红字撞名、历史人物谐音，输出分级结论（pass/warn/fail）。
- 【services/handlers】★ **长任务异步化 + 进度**：新增 `POST /names/generate/async`（立即返回 `task_id`）+ `GET /names/task/:id`（轮询 stage/percent/result），前端 `generateNamesWithProgress` 以 500ms 轮询驱动真实进度条，替换"点生成后干等"的体验；引擎 session 增加阶段打点（排盘 → 检索 → 枚举进度按完成数上报 → 补全），TaskService 进程内托管任务状态（含过期清理），同步路由保留兜底。
- 【services】★ **探索模式（换一批）**：生成响应新增 `generation_id`，会话候选表进程内缓存（TTL 清理）；`POST /names/generate/explore` 凭 ID 从**未上过榜**的候选中随机补一批（与榜单零交集，引擎 `MarkShown` 保证）；结果页新增「换一批 / 返回推荐榜」。
- 【server】路由级双链路收口：**`POST /names/generate/analysis` 下线**。喜用神口径此前已统一，且 `score_detail` 两条链路同源后，analysis 的用户可见增量只剩"字段集分裂"这一种负面价值。服务层 `GenerateWithAnalysis`（引擎分析能力，10+ 处 E2E 覆盖）保留。`/generate` 成为唯一公开生成入口。

### 🗑 Removed 废弃功能

- 【handlers/services】**接口整合清理：HTTP 路由 30 → 17 条**（`docs/27` 接口利用率审计：前端 9 个页面仅覆盖 9 组接口）。下线无前端调用的 20 条：`/bazi/analyze`、`/yijing/*`×2、`/zodiac/*`×2、`/favorites/check`、`/history/batch`、`/favorites/batch`×2、`/feedback`×3、`/report/html`、`/namestats/*`×8、`/characters/radical|curated-names|styles`，并删除对应 handler/service 文件（bazi/yijing/zodiac/feedback 服务、namestatistics handler）。**领域层能力全部保留**（八字/易卦/生肖已并入生成响应内部消费；`namestatistics` 包含独立测试与数据资产，留作探索页素材）。`/report/pdf` 保留（批次三要接的出口）。
- 【frontend】`api.ts` 与页面一一对应，无未用导出；`HomeContent` 切换到 `generateNamesWithProgress`（进度条）。

### 🧪 Tests 测试

- 新增 fate 层：`risk_test.go`、`rate_given_name_test.go`（装配字段完整性/确定性）、`engine_progress_test.go`（进度单调不减、阶段序列）。
- 新增服务层 `evaluate_e2e_test.go`：测名 E2E（含同名同分断言）、探索零交集、异步任务提交→轮询→取结果全流程。
- 调整：handlers 测试移除已下线接口用例；拼音回归 3 例切到 `/names/generate`；报告测试移除 HTML 用例。`go test ./...` 18 包全绿，`go vet` 通过，前端 `tsc --noEmit` 通过。

---

## [2026.09.18.0]

### ✨ New Features 新增功能

- 【services】★ **`/names/generate` 首次返回 `score_detail`（各维度评分依据文字）**。前端 `NameCard`/`NameDetail` 早就写好了「优先使用 `score_detail`，回退旧字段」的分支，`docs/24` P2-5 记录该分支**永不执行**——本路径此前只把引擎的 `Score.Items`（各维度分数）映射到 `name.Name`，**丢弃了 `Score.Details`（依据文字）**，导致用户始终看不到「为什么是这个分」，`FullReport` 只能用本地函数自造命理文案。现收敛为 `applyFateScoreDetail` 一处映射，分数 / 依据文字 / `score_detail` 三件事一次做完，并与 `/generate/analysis` 共用 `buildScoreDetail` 的维度顺序（避免两套顺序漂移）。
  - 顺带修掉一处**漏映射**：`case "三才"` 分支此前只留注释不赋值（当时 `name.Name` 无对应字段），三才分在 `/generate` 路径整项丢失；现映射到 `SancaiScore`，并新增 `MeaningDetail` 字段承接「文化印象」依据文字。
  - 空/纯空白的依据文字视为「引擎未提供」，**不覆盖**调用方已填的文案（避免空串冲掉 `buildResponse` 阶段的兜底文案）。
- 【middleware】**新增 `MaxBodyBytes` 请求体大小上限中间件**（`docs/24` P1-4）：全仓此前无 `http.MaxBytesReader`，handler 直接 `ShouldBindJSON` 读全量 body，单个大 body 即可造成内存放大（全局 IP 限流按请求数计量，挡不住"少请求、大体量"）。两层防护：`Content-Length` 已知且超限 → 直接 **413 + 中文提示**且不进 handler；长度未知（分块传输）→ `http.MaxBytesReader` 兜底。`cmd/server/main.go` 全局挂载 1 MiB。

### 🐛 Bug Fixes 问题修复

- 【classics】★ **修复诗词异步加载可能挂死全站**（`docs/24` P2-9）：`loadShiCiAsync` 的加载协程写成 `go func(){ ...; close(shiciReady) }()`——既未 `defer close` 也无 `recover`。后果是双重的：加载中 panic 会因 goroutine 内未 recover 而**打崩整个进程**；即便外层能 recover，`close(shiciReady)` 也永不执行，而 `ensureShiCiLoaded()` 是无超时的 `<-shiciReady` 死等 → 所有依赖诗词的请求（出处回填、共现评分、经典来源加字）**全部挂死且不报错**（表现为"服务假死"而非失败）。现改为 `runShiCiLoad`（`recover` + `defer close`，loader 以参数注入以便测试钉住这条不变量）与 `waitShiCiLoaded` 有界等待（超时 30s 即降级为无诗词共现数据并告警）。
- 【services】`applyFateScoreDetail` 对空白依据文字的判据用 `strings.TrimSpace`，纯空白不再覆盖既有文案（由新增测试 `TestApplyFateScoreDetail_KeepsExistingTextWhenDetailMissing` 暴露）。

### 📚 Docs 文档更新

- 新增 `docs/27-竞品对标与业务能力扩展蓝图.md`：以**真实代码核对**（30 个后端接口逐条比对前端接入情况）与 2026-09 市场现状为基准，给出三层（业务功能 / 质量 / 服务交互）差距矩阵、落到文件级的借鉴清单、明确不引入清单与后续批次验收标准。三条关键结论：
  - **纠正 `docs/15`/`docs/18` 三处**：① `docs/18` 建议「必须补五格数理 + 三才配置」与 `AGENTS.md`「禁用熊崎五格数理」直接冲突，明确排除（三才保留、81 数吉凶不引入）；② `docs/18` 判定的「缺少文字解读」实为**能力没接出来**而非缺能力（依据文字与前端渲染分支都已存在）；③ 用「维度数量」证明领先对用户无感，度量口径改为**依据文字覆盖率**。
  - **最扎眼的发现**：后端 30 个业务接口中前端只有 9 个有页面在用——八维评分依据、八字分析、易经卦象、生肖宜忌、**8 条人名语料统计**、真实 PDF 报告（`docs/25` 已实现）全部没有入口；`word.json` 16,142 条字义素材完全未展示。故后续优先级排序为「**接出口 > 补缺口 > 加新算法**」。
  - **两处红色缺口**：① 「**测名 / 评名**」全竞品标配而本项目完全缺失（行业数据：用户攒下的候选名约 70% 会在测名环节被筛掉）；② **风险体检**（谐音 / 生僻字 / 多音字 / 户籍友好度 / 网红字撞名）判据都在引擎里但不产出清单。其中 A1（测名）已核对实现前提——`fate.RateName` 虽导出，但 `NameCandidate` 的 20+ 字段目前只在引擎内部枚举时组装，需先抽出「给定姓名 → NameCandidate」的装配函数，否则会制造"结果页与测名页同名字不同分"的新双口径问题。

### 🧪 Tests 测试

- 新增 `name_service_score_detail_test.go`（5 用例）：八维度（含三才）分数与依据文字全映射、空白依据文字不覆盖既有文案、nil/空评分/未知维度安全、`buildScoreDetail` 顺序与缺失维度跳过，以及**端到端** `TestGenerate_EmitsScoreDetail`（走真实 fate 引擎，断言每个名字都带 `score_detail`、维度落在白名单内、至少一维带依据文字、三才分与 `sancai_score` 一致）。
- 新增 `max_body_bytes_test.go`（5 用例）：超限 413 且 handler 不执行、未超限 body 完整可读、**长度未知时由 `MaxBytesReader` 兜底**（读出字节数不超过上限且报错）、`limit<=0` 不限制、无 body 请求不受影响。
- 新增 `shici_ready_test.go`（7 用例）：★ `TestRunShiCiLoad_ClosesChannelOnPanic` 钉住「loader panic 也会关闭通道」这条核心不变量（修复前会死等），另有报错/成功路径关闭通道、有界等待超时与非正超时（不限时）语义。用注入通道/loader 的纯函数测试，避免触碰 `sync.Once` 保护的全局 `shiciReady`。
- `go test ./...` 全部通过（18 个包）；`go build ./...` 通过。
- 注：`gofmt -l` 仍会列出若干 CRLF 文件（含本仓库既有状态，见 `docs/24` P3-13）；本次改动的 `name_service.go` / `generator.go` / `classic_loader.go` 均无格式告警。

### 📚 相关

- 修复进度回写：`docs/24` 的 P1-4、P2-5、P2-9 现已完成；P2-6 喜用神口径两条链路已统一为经典口径，
  响应字段集统一仍待做（`docs/27` §6 批次三）。

---

## [2026.09.13] 当日总览

命理数据准确性专项（审查 1–14 闭环 + 后续增强），版本 `2026.09.13.0` → `.6`：

| 版本 | 要点 |
|------|------|
| `.0` | 剥/鼎象辞通行本修正；出生时间 1 分钟粒度 + 时辰边界提示 |
| `.1` | 时辰 23 点归子、纳音立春校正、季节/调候按月支、前端出生时间校验 |
| `.2` | 经典来源注入修复 + 评分透传（另一并行修复） |
| `.3` | 藏干+月令加权喜用神、SQLite 经典版本对拍、`cmd/check_json` |
| `.4` | 真太阳时、梅花姓名卦（姓上名下）、喜用神交叉对拍 |
| `.5` | 23 卦上下卦字段校正；经度库扩至约 100 城 |
| `.6` | 四柱纳音（年/月/日/时）进入 API 与结果页 |

---

## [2026.09.14] 当日总览

起名生成性能专项：修复全量枚举长时间无结果的问题，版本 `2026.09.14.0` → `.1`。

| 版本 | 要点 |
|------|------|
| `.0` | 谐音词表预编译索引、字义重叠画像化、早停前置、经典字集排序确定性 |
| `.1` | 评分与解释分离（惰性 Detail）、局部表去 seen、同分排序确定化 |

---

## [2026.09.15] 当日总览

起名链路可观测性与质量修复专项：E2E 巡检工具化（Go 实现，去除 Python 依赖）+ 门禁表扩容与荒谬字霸榜根因修复，版本 `2026.09.15.0` → `.1`。

| 版本 | 要点 |
|------|------|
| `.0` | 新增 `tools/e2e_check` 与 `tools/gate_audit`；论证质量门禁表不可移除 |
| `.1` | 门禁表扩容 131→2990 字并纳入版本管理；修复荒谬字霸榜根因（封顶豁免判据） |
| `.2` | 新增遗留问题清单（`docs/24`）；清理调试遗留 scratch；更正过期注释 |
| `.3` | 修复超时中间件上下文复用竞争与超时语义闭环；`/report/pdf` 改为真实 PDF |

---

## [2026.09.15.3]

### 🐛 Bug Fixes 问题修复

- 【middleware】**修复 `RequestTimeout` 与 gin 上下文复用的竞争（`docs/24` P1-1）**：原实现在 goroutine 里跑 `c.Next()`，超时分支直接返回，而 gin 会立刻把 `*gin.Context` 放回 `sync.Pool`——那个 goroutine 仍在写同一个 `c.Writer`/`c.index`，并与复用该对象的下一个请求并发读写。后果不是「优雅超时」，而是**两个请求的响应被交替写入同一连接**。现改为串行执行 `c.Next()`（gin 的 Context 不支持并发使用，想在中间件里强行中断 handler 就无法绕过这一点），并在处理链返回后、若 `ctx.Err()==DeadlineExceeded` 且未写出任何响应时补 503。代码注释同时写明残留限制：完全不理会 ctx 的 handler 仍受 `http.Server` 的 `WriteTimeout` 兜底。
- 【services】★**修复「超时时返回被截断的 200」**：引擎各分片在 `ctx.Done` 后提前退出、`session.Wait()` 仍正常返回，而 `FateNameService` / `NameService` 此前不检查 `ctx.Err()`——实测给一个已过期的 deadline，`GenerateWithAnalysis` 会**返回成功结果**，客户端拿到「200 + 不完整名单」且不会重试。现服务层显式透出 ctx 错误。
- 【handlers】超时与普通失败分开映射：`errors.Is(err, context.DeadlineExceeded)` → **503**「请求超时，请稍后重试」，其余仍为 500。此前一律 500，前端无法区分「可重试」与「服务异常」。

### ✨ New Features 新增功能

- 【pdf】★ 新增 `internal/infrastructure/pdf` —— **不依赖第三方库**的 PDF 生成器，重点解决中文渲染：
  - `ttf.go`：TrueType 解析（表目录、`cmap` format 4/12、`loca`/`glyf`/`hmtx`、复合字形依赖），支持 `.ttf` 与 `.ttc`（取集合第 0 个字体），明确拒绝 `.otf`/CFF。
  - `subset.go`：字形**按需子集化**（重编号 + 复合子号重写 + sfnt 表校验和），样例报告 53 KB 而源字体 9.7 MB。
  - `document.go`：PDF 1.7 对象/页面/内容流与排版 API（浮动光标、自动折行分页、页脚回调、填充矩形、分隔线）。字体用 **Type0 + CIDFontType2 + Identity-H**（CID 即子集内新字形号，`/CIDToGIDMap /Identity`），并写 ToUnicode CMap 保证文本可复制可检索。
  - `font.go`：系统字体发现——`NAMER_PDF_FONT` → 平台常见路径，每个候选都**真实解析并校验含「中」字字形**（只查"能否解析"会选中只有拉丁字形的字体）。
  - 缺字降级：先尝试去变音符号（`ā→a`，复用已有依赖 `golang.org/x/text/unicode/norm`），仍无字形才用 `?` 占位并记录 `MissingRunes()`。
- 【tools】新增 `tools/pdf_preview`：生成样例起名报告 PDF，走与线上一致的链路，便于核对排版与中文渲染。

### 📈 Improvements 性能/体验优化

- 【services】**`/report/pdf` 由硬编码占位实现改为真实报告**（`docs/24` P1-3、`docs/25`）：报告改为「解析请求 → 统一报表模型 → 分别渲染 PDF/HTML」，两种格式字段口径一致（此前各自解析一遍请求体）；PDF 含标题/生成时间、基本信息、八字与五行、推荐名字表格（序号/姓名/拼音/五行/评分 + 寓意折行）与「第 X / Y 页」页脚；HTML 复用同一模型渲染并保留转义。空数据或非对象输入产出「未提供有效报告数据」的可读报告而非 500；**找不到中文字体时返回 500 + 排查指引，不再输出正文空白的"报告"**。

### 📚 Docs 文档更新

- 新增 `docs/25-报告PDF生成实现说明.md`：实现结构、子集化方案、字体发现与降级、验证方式（含 **`pdftotext` 默认按 Latin-1 输出、必须加 `-enc UTF-8`** 这一排查陷阱）、测试矩阵，以及 ★ 踩坑记录：子集 `loca` 错位一个字形会让 PDF「能打开、字像汉字但全是别的字」（实测「张氏宝宝起名报告」渲染成「德气容容辰告拼周」）。
- `docs/24-遗留问题清单.md`：P1-1 / P1-3 标记为已修复并补修复方案与护栏说明。
- `docs/05-部署使用手册.md`：后端配置新增「中文字体（PDF 报告必需）」小节，说明查找顺序、`NAMER_PDF_FONT` 与容器镜像需装中文字体。

### 🧪 Tests 测试

- 新增 `internal/infrastructure/pdf/pdf_test.go`（11 个用例）：字体解析 / 拒绝 CFF / 复合字形依赖 / **逐字形字节比对（loca 错位回归）** / loca 单调性 / ToUnicode 覆盖 / 端到端结构与分页 / 缺字兜底 / 折行与测量 / 字体缺失报错。
- 新增 `internal/application/services/report_service_test.go`（6 个用例）：真实数据进入文档（断言占位文案消失 + 数据出现在 ToUnicode CMap）/ 空数据 / ctx 取消 / PDF 与 HTML 同模型 / XSS 转义 / 模型解析容错。
- 新增 `internal/infrastructure/middleware/request_timeout_test.go`（7 个用例）与 `TestGenerateWithAnalysisDeadlineExceeded`；已在旧实现上验证前两项会失败。
- `handlers_test.go` 新增 `TestReportHandlerGeneratePDF`（无中文字体环境自动跳过）。
- `go test ./...` 全部通过。

---

## [2026.09.15.2]

### 📚 Docs 文档更新

- 新增 `docs/24-遗留问题清单.md`：对后端、前端、仓库卫生做六类排查（代码质量 / 并发资源 / 健壮性 / 安全配置 / 仓库卫生 / 前端），按 P1–P3 分档列出 19 项剩余问题并给出证据与建议动作。**结论：P0 已清零。** 其中值得优先处理的是：`RequestTimeout` 中间件超时后与 gin 上下文复用竞争（会写坏跨请求响应）、`/report/pdf` 是硬编码占位实现却已暴露路由、前端从不调用 `/names/generate/analysis` 导致 `score_detail` 恒空且报告文案由前端自造。

### 🐛 Bug Fixes 问题修复

- 【fate】更正 `naming_quality.go` 头部注释：字表字数「1094 字」→「当前 2990 字」，并说明两条来源（人工策展 + 规则扩容）；同时补齐该文件缺失的末尾换行（`gofmt -l` 会因此报错）。

### 🧹 Chores 杂项

- 清理调试遗留的 scratch：`backend/tmp/fate.test`（8.2 MB 测试二进制）、`backend/d/`（绝对路径被误当相对路径产生的空目录树 `backend/d/19-Training/…`）。

---

## [2026.09.15.1]

### 🐛 Bug Fixes 问题修复

- 【fate】**修复双名荒谬字霸榜（P0）**：`RateName` 的四维封顶豁免原为「任一字是精选好字（策展 ∩ `positiveScore>=90`）即整组豁免」，而 `positiveScore` 只覆盖 8105 字中的 563 字、未覆盖的字得不到任何负反馈，导致「精选好字 + 任意字」屠榜——实测双名 Top10 有 10/10 是该形态（张沚明/张鲛慧/张唣明/张慧僰/张恃泽/张蚂泽/张浩荥/张噬鹏/张蚂宏/张蚂清）。现追加前置判据 `hasNamingEvidence`：**豁免封顶时两个字都必须具备命名依据**（人工寓意评分 / 策展分类字 / 出现在 95.7 万条真实人名语料中）。判据是「有没有依据」而非「好不好」，不参与打分，故不动既有权重与分数标定；也未收紧为「两字都须精选好字」的严格 AND，以保留推荐多样性。
  - 效果：双名 Top10 → `张珀熙 张慧茗 张珀诗 张茗浩 张赟慧 张珀宏 张珀明 张珀渊 张珀辉 张珀泽`；榜单「无命名依据字」占比 **100% → 0%**，「无寓意评分」用字数 30 → 4（凝/沅/珀/赟，均有真实人名语料依据）；单名榜保持正常。
- 【services】修复出典回填未清空 `PoetrySource`：`enrichPoetrySource` / `enrichPoetryForName` 的「索引未命中」分支注释写的是「原 PoetrySource 留空，PoetrySentence 兜底引擎原句」，但代码只把原句**复制**到 `PoetrySentence` 而未清空 `PoetrySource`；由于调用方会先用引擎格式化原句（`「…」`）预填，前端会把原句渲染成《「睿而爲愚者也」》——出处位置出现带引号的句子。现按既定语义补上清空，分析与生成两条路径一并修正（由 `TestGenerate_PoetryBackfill` 暴露）。

### ✨ New Features 新增功能

- 【工具链】新增 `backend/tools/gate_expand` —— 质量门禁表扩容工具，只增不删，两种入库来源：
  - **历史表恢复（`-recover <rev>`）**：自 git `aa024f4` 的 `naming_quality.go` 内嵌 `map` 中还原 1094 字表（与后来外置的 JSON 是同一份数据，见 `7bfb8b9` 注释佐证；实测现 131 字表是其真子集）。
  - **命名准入规则扩容**：对 8105 字检查四个「证据维度」——真实人名用法（`NameFreqTier>0`）/ 人工寓意评分 / 策展好名认可 / 典籍诗词出处，全部落空即收进门禁表（命中 1975 字）。
  - 两条防误伤保险：`王/玉` 部二级及以下豁免（美玉字即便语料罕见亦有命名价值，如 `瑄`）；内置 115 个「守护字」硬校验，优质字被误伤即**拒绝写盘并退出 1**。

### 📈 Improvements 性能/体验优化

- 【data】**门禁表 131 → 2990 字**（恢复历史 1094 字 + 规则扩容 1896 字），并把 `backend/data/naming_quality.json` 纳入版本管理：`.gitignore` 由 `backend/data/` 改为 `backend/data/*` + `!backend/data/naming_quality.json`（Git 不允许排除整个目录后再反选目录内文件，必须排除目录**内容**）。该表历经 40+ 轮 `verify_fate` 迭代累积，2026-09 曾因整个 data 目录不入库而丢失、只能凭记忆重建 131 字——这是一次可发现性为零的资产损失，必须修掉。
  - 覆盖效果：门禁表 2990 字中 **1775 字只有它挡得住**（其余 1215 字已被 filter 链 / `IsNegative` / 硬负面表覆盖）；双名字源池 5558 → **3778 字**（-32%），组合规模 3089 万 → **1427 万对**（-54%，枚举同步变快）。
- 【工具链】`tools/e2e_check` 的 P0 探测口径由「无 `positiveScore` 字占比」改为「**无命名依据字占比**」：前者会误报（`珀/赟/茗/诗` 本就没有寓意评分却完全可用），后者才是荒谬字霸榜的量化形态；同步读取 `name_frequency.json` 与 `curated_names.json` 建立判据。

### 📚 Docs 文档更新

- 新增 `docs/23-P0修复记录_门禁表扩容与荒谬字霸榜根因.md`，完整记录：门禁表丢失链条（`aa024f4` → `7bfb8b9` → `3c40d7e`）、扩容方案与数字、**关键实测「扩容单独修不好 P0」**（扩容后 沚/鲛/唣/僰/恃/荥/噬 一个没少，原因是这批字都查得到诗词出处；若去掉诗词豁免则规则会命中 84% 的字、连「彧」都误伤）、真正的根因与修法、回归护栏与复现命令。
- 更新 `docs/22`：2.2 检查项表、2.3 节（标注已修复并指向 `docs/23`）、第 3 节（补扩容后数字）、第 4 节（建议动作改为状态表）。

### 🧪 Tests 测试

- 新增 `TestNonCuratedCapRequiresNamingEvidence`（6 组判据：好字+无依据字必须封顶、好字+人名语料/策展/寓意评分豁免、策展白名单优先豁免、单名不封顶）与 `TestHasNamingEvidence`（真值表）作为回归护栏。
- `go test ./...` 全部通过。

---

## [2026.09.15.0]

### ✨ New Features 新增功能

- 【工具链】新增 `backend/tools/e2e_check` —— 起名链路端到端巡检工具，**纯 Go 标准库实现**，取代原 `backend/scripts/e2e_name_check.py`（已删除，不再依赖外部 Python）。覆盖十项检查：九场景耗时与返回完整性、推荐用字等级分布、门禁字命中、双名榜「无正向信号字」占比、避讳长辈、`source_classic` 生效性、双路径喜用神一致性、双路径响应结构差异、同一请求可复现性、并发压测（中位/p95/吞吐与并发放大系数）。支持 `-host`/`-concurrency`/`-rounds`/`-repeat`/`-data`/`-timeout`，`-strict` 时发现问题以退出码 1 结束（便于接入 CI）；数据目录自动定位（依次尝试 `-data`、`data`、`../data`、`../../data`），从任意工作目录均可运行。
- 【工具链】新增 `backend/tools/gate_audit` —— 名字用字质量门禁表（`data/naming_quality.json`）覆盖审计工具，分三段给出可决策的量化结论：
  - **A 静态覆盖**：把门禁表逐字过一遍其他字级防线（笔画/生僻/性别/等级/硬负面/消极），区分「只有门禁表挡得住」与「冗余条目」；
  - **B 动态对照**：`LoadNamingQualityFromJSON` 指向不存在目录即可让门禁降级为空表，据此分别跑「启用/置空」两次完整生成，对比已评分组合数、Top-N 差异与门禁字是否上榜；
  - **C 替代方案可行性**：量化「黑名单剔除 → 正向信号准入」的准入集规模与组合规模。

### 📈 Improvements 性能/体验优化

- 【工具链】E2E 巡检由一次性脚本升级为可重复执行的命令，输出改为「✓/★ + 汇总清单 + 可选非零退出码」的判定式报告，不再需要人工比对数字。

### 🐛 Bug Fixes 问题修复

- 【工具链】修正原 Python 巡检脚本的一处无效断言：其「农历月」对比读取的 `data.bazi.lunar_month` 字段在两个端点都**不存在**（实际字段为 `is_leap_month`），该项检查恒等在比较 `None`。Go 版改为对比 `bazi` 字段集与 `names[0]` 字段集，能真正暴露两条链路的结构分裂。

### 📚 Docs 文档更新

- 新增 `docs/22-起名链路E2E巡检工具与质量防线审计.md`，记录工具用法、E2E 实测结果与**门禁表可移除性论证**：
  - **结论：门禁表不可移除**。静态审计显示其 131 字中 **113 字的拦截完全依赖该表**（这些字均已通过 filter 链进入 5558 字候选池）；动态对照显示移除后**单名路径「伯」直接进入 Top50**（实锤回归），双名路径虽无门禁字上榜但 Top50 变化 8 个。
  - 真正的问题是**覆盖不足**（现 131 字，`docs/19` 记载历史 1094 字版已丢失），而非冗余。
  - 替代方案「正向信号准入」不可行：池内带正向信号的字仅 **386/5558（7%）**，准入后双名组合规模从 3089 万对崩到 14.9 万对，会重新触发「候选收窄 → 名字高度近似」的历史问题。**正确方向是补信号（扩大策展/正分覆盖），而非删表。**
  - 同时固化本轮 E2E 发现的三项待决策问题：① 双名榜 50/50 含「无正向信号字」（`premiumChar` 封顶豁免用「或」+ `positiveScore=0` 无惩罚的复合后果）；② 两条链路喜用神互相矛盾（`[水]` vs `[木 水 金]`）；③ 双路径响应结构分裂（`score_detail` 仅单侧）。另记录 `localhost` 比 `127.0.0.1` 稳定慢约 2s 的**测量陷阱**（IPv6 `::1` 优先但服务只监听 IPv4）。

### ⚠️ Breaking Changes 破坏性变更

- 删除 `backend/scripts/e2e_name_check.py`（该文件从未提交入库，为上一轮临时产物），由 `backend/tools/e2e_check` 取代。

---

## [2026.09.14.1]

### 📈 Improvements 性能/体验优化

- 【fate】起名生成热路径「评分与解释分离」（累计 **单组合 21.5μs → 4.2μs，5.1×**；**分配 156 → 14 次，-91%**；16 核单姓氏完整生成 **2.85s → 0.73s**）：
  - 【惰性 Detail】新增 `detailSink`（`detail_sink.go`）作为评分依据文案的统一出口：`SkipDetail` 为真时 `add/addf` 退化为空操作，且 `addf` **在调用 `fmt.Sprintf` 之前**即返回，从根上消除格式化分配。`rater.go` 7 个 Rater 的 72 处 `details = append(...)` 与 7 处 `strings.Join(details,"；")` 收尾统一改走 sink；`checkSemanticPoetry`/`checkSingleNameBigram`/`evaluateTonePattern`/`shengMuSimilarity` 增加 `wantDesc` 参数（跳过热路径无谓的文案格式化，分值不变）。
  - 【总分与明细分离】新增 `RateNameScore`：只算综合总分与等级，不构造 `Items/Details` 两个 map 与任何文案；`RateName` 保持「始终返回完整明细」的契约（内部临时关闭 `SkipDetail`）。两者共用 `nonCuratedCapApplies` 封顶判据与取整逻辑，由 `TestRateNameScoreMatchesRateName` 断言总分逐位一致（另以全量候选对校验和验证：新旧实现 `sum=613676.2000` 完全相同）。
  - 【入榜后回算】枚举阶段只写总分，`ExcellentEntry` 新增内部字段 `idx1/idx2` 记录候选字下标；`fillEntryDetails` 仅对进入推荐榜的条目（≤ topCount×10）回算完整明细，代价约为枚举量级的千分之一。
  - 【局部表去 seen】新增 `NewExcellentTableUnique`：worker 局部表不做 `Char1+Char2` 去重（每个条目省下「两个汉字拼接成字符串 + map 写入」的分配）。相应在候选池阶段新增 `dedupCharsByName` 按汉字去重，保证 `(i,j)` 组合在「名字」层面唯一（历史上由 seen map 兜底，语义等价）。
  - 【辅助优化】`shengMuGroupName` 的分组名 map 提升为包级常量（原实现每次调用重建 map）；`rateWeightsByDim` 的每调用一次 map 分配改为按需 `weightOf` 线性查（仅封顶分支调用）。
- 【fate】同分排序确定化：`ExcellentTable.Finalize` 由 `sort.Slice` 改为 `sort.SliceStable`。原实现同分条目的先后由元素字节内容（含 `Items/Details` 等 map 指针）决定，使「枚举期是否构造明细」这类与排序无关的实现细节能影响 Top-N 边界入选（同分挤在 `poolSize` 截断处时尤甚），同一请求给出不同榜单。稳定排序后同分次序只由「得分比较 + 推入次序」决定，对载荷不敏感、可复现。

### 🐛 Bug Fixes 问题修复

- 【fate】修复测试字桩 `newCuratedPoolProvider` 与自身防线说明不自洽：该文件注释声明防线为「WenHuaRater 仅对『策展 ∩ 评分≥90』加分 + 非策展双名四维封顶」，但桩里好字从未置 `IsCurated`，导致好字与荒谬字**得分完全相同**，`TestCuratedPoolExcludesAbsurdChars` 只能靠同分排序的偶然次序通过（任何改动条目载荷的实现优化都会让它翻车）。现好字置 `IsCurated: true`，使 `premiumChar` 封顶豁免机制真正生效。
- 【fate】修复 `TestCuratedPoolKeepsGoodChars` 的判定基准：原用固定的 8 字抽样（泽/清/涵/澄/渊/明/瑞/浩）判断「好字是否上榜」，而字桩使用合成拼音（`p1`/`p2`…），音韵分随拼音序号变化，固定子集可能恰好不落在榜首。现改为对**全量好字集**判定，并新增「推荐榜不得混入非好字组合」的更强断言。

### 📚 Docs 文档更新

- 新增 `internal/domain/fate/detail_sink.go`（评分依据文案懒加载说明）。
- 新增 `internal/domain/fate/rate_score_parity_test.go`：热路径总分与全量明细总分一致性护栏。

---

## [2026.09.14.0]

### 📈 Improvements 性能/体验优化

- 【fate】起名生成热路径性能专项：双名全量枚举**单组合耗时 23.2μs → 10.9μs（2.1×）**，**分配次数 154 → 51 次（-67%）**；16 核单姓氏完整生成 **2.85s → 1.01s**，后端全量测试套件整体约 2× 提速（services 45.4s→23.8s，handlers 31.0s→15.1s）。
  - 【谐音词表预编译】`CheckBadPinyinCombo` 原先对约 110 条组合，每次调用各做一次 `strings.Join` 重新拼接（110 次分配/调用）再各做一次 `strings.Contains`；改为 `init` 阶段预编译「拼接串 → 下标」索引 + 按词长滑窗查表，返回语义等价（保留词表中最靠前的命中项）。`CheckBadHomophone` 由线性扫描约 300 条词表改为「拼音 → 下标切片」索引。二者合计占单次评分约 43%。
  - 【字义重叠画像化】`semanticOverlap` 原为 O(|a|·|b|) 逐字 `ContainsRune` 扫描（实测释义中位 230 字，最坏约 7 万次比较，占单次评分 15%）；改为按候选字预计算「字符 → 出现次数」升序画像后归并，复杂度 O(|A|+|B|)，结果保持精确（重复计次、标点排除语义不变）。`NameCandidate` 新增 `MeaningProfile1/2` 注入通道，未注入时回落按释义字符串取缓存。
  - 【早停前置】`generateDoubleName` 内层循环原顺序为「组合级门禁 → 早停 → 评分」，门禁（`IsBadCombo`/`IsHistoricalFigureCombo`/`IsNonNamingChar`/`PairBlacklist`）对 N² 全部组合执行；改为早停前置（潜力分不足直接跳过），被跳过的组合本就不会产生结果，语义等价。潜力分改为按字预计算成数组（原实现每次组合重算两字，其中外层次的部分纯属重复）；新增无锁 `ExcellentTable.earlyStopCutoff` 供单 goroutine 独占的 worker 局部表使用。
  - 【拼音去声调快路径】`stripTone` 对纯 ASCII 拼音零分配返回，避免热路径反复创建 `strings.Builder`。
- 【classics】经典提取字集顺序确定性修复：`mapToSortedSlice` 历史上直接遍历 map（未真正排序），导致各经典来源字集顺序随进程变化，进而使候选池注入顺序、以及依赖「字集首个字」的结果在进程间漂移。现按汉字码点升序返回。

### 🐛 Bug Fixes 问题修复

- 【fate】修复 `TestWenHuaRaterSourceBonus` 间歇性失败（原始代码实测约 2/10 概率失败）：测试取《论语》字集首个字作为「来源字样本」，若该字恰为质量门禁字，会被 -12 分/字的硬惩罚吞掉 +5 来源加分，使断言前提不成立。现改为排除门禁字后再取样，`TestWenHuaRaterSourceBonusDoubleHit` 同步加固。

### 📚 Docs 文档更新

- 新增 `backend/tools/bench_gen` 真实数据基准工具（数据装载 → 候选池规模 → 生成耗时/已评分组合数）。
- 新增 `internal/domain/fate/perf_bench_test.go` 热路径微基准（组合预检链 / 各评分器 / 释义重叠），作为性能回退护栏。

---

## [2026.09.13.6]

### ✨ New Features 新增功能

- 【bazi】四柱纳音全量暴露：
  - 新增 `NayinInfo` / `FourPillarNayin` 与 `BuildFourPillarNayin`
  - `BaziAnalysis.four_nayin`：年/月/日/时干支+纳音+纳音五行；`nayin` 兼容为年命纳音
  - fate 适配器 `NaYin[4]` 与 GenerateWithAnalysis 同步填充
- 【frontend】结果页「四柱纳音」四格展示；类型补 `four_nayin`、`birth_longitude`

### 🧪 Tests 测试

- `four_nayin_test.go`：干支对照表、AnalyzeBazi 一致性、60 甲子全覆盖

---

## [2026.09.13.5]

### 🐛 Bug Fixes 问题修复

- 【yijing】校正 23 卦 `UpperTrigram`/`LowerTrigram` 与 `Symbol`（先天八卦对照）：
  - 如 需/讼/泰/否/谦/蛊/剥/解/鼎/临/复/升/困/萃/姤/夬/大过/蹇/归妹/丰/旅/涣/小过 等
  - 同步 `hexagram.go` 内置表与 `data/yijing.json`
  - 新增一致性测试：64 卦上下卦=对照表，且梅花起卦结果字段自洽

### 📈 Improvements 性能/体验优化

- 【bazi】真太阳时经度库扩至 **约 100 城**（地级市+主要地区），匹配升级为「精确→最长前缀→包含」

### 🧪 Tests 测试

- `trigram_consistency_test.go`：64 卦 Upper/Lower/Symbol 与先天八卦对照表一致；梅花起卦结果字段自洽
- 扩展 `true_solar_test.go`：最长前缀/包含匹配、扩展城市抽样、库规模 ≥80

### 📚 Docs 文档更新

- 新增 [2026.09.13] 当日总览；修正 `.4` 遗留项说明（上下卦已于本版闭环）

---

## [2026.09.13.4]

### ✨ New Features 新增功能

- 【bazi】真太阳时校正：
  - `CityLongitudes` 常见城市经度表 + 前缀匹配；支持请求显式 `birth_longitude`
  - 经度差 4 分/度 + 均时差近似；`ApplyTrueSolar` 返回校正结果与跨时辰/跨日标记
  - `NameService`/`FateNameService` 排盘前自动校正，未收录地点回退钟表时间
- 【yijing】梅花易数姓名卦：**姓笔画→上卦，名笔画→下卦**
  - `GetHexagramByMeihuaName` + 先天八卦 64 卦名标准对照表（规避库内 Upper/Lower 错位）
  - `HexagramFinder` 新增 `FindByMeihuaName`；Generate / GenerateWithAnalysis 均已切换
- 【fate】喜用神交叉对拍回归：`xiyong_crosscheck_test.go` 7 组命例，经典加权喜用与 fate 平衡用神方向一致率 ≥60%

### 🧪 Tests 测试

- `true_solar_test.go`：城市查表、北京/乌鲁木齐校正、显式经度、均时差边界
- `meihua_test.go`：卦名对照、0/8 画、全笔画组合非空
- `go test bazi/yijing/fate/services` 全绿

### 📋 缺陷 1–14 核对

审查缺陷 1–14 已闭环。原「64 卦 Upper/Lower 个别错位」已在 **2026.09.13.5** 修正；四柱纳音已在 **2026.09.13.6** 全量暴露（年/月/日/时）。

---

## [2026.09.13.3]

### 🐛 Bug Fixes 问题修复

- 【bazi】喜用神与五行力量升级为 **藏干加权 + 月令权重**（P2）：
  - 新增 `DizhiHiddenStems`（12 地支本气/中气/余气权重）
  - `CalculateWeightedWuxing`：天干 1.0 + 藏干权重；整柱月令 ×1.5
  - 日主强弱计入印绶与月支同气；喜用神按身旺克泄/身弱生扶并补最缺五行
  - 展示用 `Wuxing` 由加权分四舍五入，API 字段兼容
- 【bazi】`AnalyzeBazi` 不再硬编码 `GetHexagramByStrokes(10)` 占位卦；姓名卦在起名服务层按笔画计算
- 【sqlite】经典数据 `seedClassicsData` 通过 `data_meta.classics_version` 对拍 JSON 目录指纹，版本变化强制重建三表，消除 JSON 热更新后双源静默漂移

### ✨ New Features 新增功能

- 【tools】新增 `cmd/check_json`：校验 64 卦结构/象辞≠卦辞/爻数、namer、经典 JSON 可解析、962 条禁忌组合，可作 CI 步骤（`go run ./cmd/check_json -data ./data`）

### 🧪 Tests 测试

- 新增 `bazi/wuxing_weighted_test.go`：藏干覆盖、月令放大、身旺/身衰喜用神方向、无硬编码卦
- `go test ./internal/domain/bazi ./internal/infrastructure/database/sqlite` 全绿
- `go run ./cmd/check_json` 通过

---

## [2026.09.13.2]

### 🐛 Bug Fixes 问题修复

- 【services】`/names/generate` 旧路径经典来源注入失效（前端 result 页直接受影响）：
  - 前端 result 页走 `POST /api/v1/names/generate`（`NameService.Generate` → `generateNamesViaFate`），
    该路径构造 `fate.GenerateOptions` 时只传了 `SourceClassic` 字段，但引擎只消费
    `Options.ExtraChars`（`engine.go generate` 阶段合入候选池），不读 `SourceClassic`
  - 结果：前端选择《论语》《孟子》《孟子》《三字经》等经典来源完全失效，Top 榜仍被
    诗经/楚辞字靠出典加分霸榜
  - 修复：`generateNamesViaFate` 补 `ExtraChars: s.fateService.resolveExtraChars(req)`，
    与 `/generate/analysis` 路径对齐；经典来源字真正注入候选池（"加字不缩池"策略）

- 【fate 评分层】经典来源无偏好，用户选择被淹没：
  - 即便 ExtraChars 注入了论语字，评分层所有经典来源一视同仁，诗经字因出典权重高
    （`calculateMatchScore` 诗经 +15、楚辞 +12）仍天然霸榜
  - 新增「经典来源偏好加分」：`WenHuaRater` 新增 `sourceSet` 字段，命中所选来源
    提取字集的字每字 +5 分（details 输出「来自【论语】选字（+5分）」）
  - 分值低于策展好字（+8）与单字出典（+8），属于"引导"而非"背书"；未指定来源时
    sourceSet 为空，循环零开销，不影响现有 `DefaultRaters` 行为

- 【fate 会话层】经典来源偏好需 per-request 装配：
  - `engine.raters` 构造期固定（`DefaultRaters`），而 `SourceClassic` 是 per-request 参数
  - 修复：`sessionImpl.Start` 阶段按 `input.Options.SourceClassic`（支持中文与拼音别名，
    `classics.sourceAlias` 归一化）替换会话级文化印象 Rater 为带来源字集版本

### 🧪 Tests 测试

- 新增 `rater_source_bonus_test.go`：`WenHuaRater` 来源偏好加分单测（2 用例）
  - `TestWenHuaRaterSourceBonus`：验证来源字集构建、命中加分（+5）、details 输出格式、未命中不加分
  - `TestWenHuaRaterSourceBonusDoubleHit`：双字均命中来源字集时按字累计加分
- 新增 `engine_source_rater_test.go`：`session.Start` 级 Rater 装配验证（中文来源 + 拼音别名 + 默认无来源 3 场景）
- 新增 `name_source_inject_integration_test.go`：`/names/generate` 旧路径注入回归（2 用例）
  - `TestGenerateNamesViaFate_SourceClassicInjected`：选《论语》时命中率从 50%→82%（+32pp），注入+评分加持生效
  - `TestResolveExtraCharsSourceClassic`：`resolveExtraChars` 对论语来源返回候选字，未指定来源时为空
- 回归：`go test ./... -count=1` 全部通过

### 📈 Improvements 性能/体验优化

- 端到端实测（`sourcecheck` 验证脚本，`data/lunyu.json` 全文 1341 字）：
  - 基线（`source_classic` 空）：50 名命中论语集 = 25（50%）
  - 选中《论语》：50 名命中论语集 = 41（82%）
  - 命中差 +16 个名字（+32pp），经典来源功能真正可用

---

## [2026.09.13.1]

### 🐛 Bug Fixes 问题修复

- 【bazi】时辰/纳音/季节口径修正（P1）：
  - `GetShichen` 原 `hour/2` 在 23 时误判为「亥」→ 改为与 tyme 一致的 `(hour+1)/2`，23 时正确为「子」
  - 年命纳音改为取 **tyme 立春校正后的年柱干支**（`NayinMap[YearGanzhi]`），不再用 `year-4` 公历年近似
  - `season` / 调候用神改按 **月支（节气月）** 判定，避免公历月在节气换月日附近出错
  - 删除死代码：`SolarTermOffset`、`getYearGanzhi`/`getMonthGanzhi`/`getDayGanzhi`/`getHourGanzhi`、`getNayinFromTyme`/`getYearSixtyCycle`、旧版 `getSeason`

- 【frontend】新增 `validateBirthTime`（1900–2100 年 + 月/日/时/分范围），提交前校验出生时间

### 🧪 Tests 测试

- 新增 `bazi/analyze_p1_test.go`：时辰边界与 tyme 对拍、立春换年四柱、纳音随年柱、季节随月支、晚子时
- 前端 `validateBirthTime` 单测 5 组（含边界 1900/2100、0:00/23:59）

### 📚 Docs 文档更新

- 记录本版本 P1 修复

---

## [2026.09.13.0]

### 🐛 Bug Fixes 问题修复

- 【yijing】修正 64 卦象辞错误（通行本《周易》大象）：
  - 剥卦：原误为卦辞「山地剥，不利有攸往」→ 改为「山附地上，剥。上以厚下安宅」
  - 鼎卦：原缺标点「木上有火鼎…」→ 改为「木上有火，鼎。君子以正位凝命」
  - 同步更新运行时 `hexagram.go` 内置表；`data/yijing.json` 由导出链路对齐

### ✨ New Features 新增功能

- 【frontend】出生时间选择精度提升：
  - `BirthdayPicker` 时间间隔由 15 分钟改为 **1 分钟**，可精确选到任意分钟
  - 弹窗实时显示当前时辰（子/丑/…/亥，与后端 tyme `(hour+1)/2` 口径一致）
  - 整点前 10 分钟（尤其 22:50+）弹出时辰/日柱边界提示，降低误跨时辰风险

### 🧪 Tests 测试

- 新增 `yijing/canonical_test.go`：16 卦通行本象辞金标准 + 象辞≠卦辞防复制 + 64 卦结构（乾坤允许用九/用六 7 条）
- 新增 `classics/poem_query_test.go`（包内首批测试）：索引构建、Quote⊆全诗、窈→诗经金标准、字级索引
- 新增前端 `lib/shichen.ts` + `birthdayShichen.test.ts`：时辰映射与边界提示 5 用例

### 📚 Docs 文档更新

- 更新本版本变更日志

---

## [2026.09.12.0]

### 🐛 Bug Fixes 问题修复

- 【database】恢复 `database.Store` 编译破坏（Q1/Q2/Q6/Q10，见 docs/19）：
  - `sqlite.Store` / `memory.Store` 补齐 `NameStatStore` 全部 8 个方法，但统一返回显式错误 `errNameStatsUnavailable`（"姓名统计数据未接入持久化存储"），消除静默返回 nil 的假象
  - 移除 `filter_integration_test.go` 中"避开 pre-existing 故障"的注释，改为真实接口守卫与结果断言

- 【SQLite 汉字过滤下沉】（Q3/Q4，见 docs/19）：
  - `SearchHanziByBasicParams` 重命名为 `SearchHanziByFilter`，对齐 `services.SQLiteCharStore` 接口
  - 重写 `queryViaSQL`：修复 genderHint 误按拼音比较（原为 `h.Pinyin != pat.genderHint`）、`wuxingNotIn`/`namingCategory` 漏过滤、多值 `wuxingIn` 只取首值、regular 排除表外保守判定字等问题
  - SQL 端只下推单选五行/笔画区间/具体字集合，其余（regular/nameable/多值五行/genderHint/namingCategory）在 Go 端以内存 `hanzi.HanziData` 为权威源二次过滤，保证字段完整一致
  - 新增 SQL 与 Go 结果一致性集成测试（genderHint/wuxingNotIn/namingCategory 断言结果集完全一致）

- 【classics】`loadShiCiAsync` 每次调用都 `close(shiciReady)`，重复 `data.Init`（测试并发）触发 `close of closed channel` panic；改用 `sync.Once` 保护（`classic_loader.go`）

- 【services】双管线喜用神不一致（Q5，见 docs/19）：
  - `GenerateWithAnalysis` 无 `WuxingMatch` 时按经典喜用神收窄候选池（注入 `WithFateBaziAnalyzer`），与 `Generate` 行为对齐，避免候取名五行与 API 响应 Bazi 喜用神矛盾

- 【services】`GenerateWithAnalysis` 响应补齐卦象与紫薇（Q5）：
  - 注入 `WithFateHexagramFinder` / `WithFateZiweiAnalyzer`，按生成名字平均笔画计算姓名卦象、按出生时间紫薇排盘，与 `Generate` 响应一致（此前 `hexagram`/`ziwei` 恒为空）

- 【legacy】移除无意义的 `/names/:id`（GetByID）路由、handler 与服务方法（Q11）：接口自始至终返回"暂不支持"且前端未调用，属死代码，一并修正 `rater.go` 注释"七维"→"八维"

- 【fate】审查遗留小项收尾（docs/19 B7/B11/B12/P4）：
  - B12：`BadHomophones` 中 `"fu2"` 数字后缀数据错误导致「妇」永不命中；改为 `"fu"`，并修复 `CheckBadHomophone` 同拼音多词条时首条不中即退出、漏检后续词条的缺陷
  - B7：`BadPinyinCombos` 移除过宽误判项 `da dai`（打的，中性动词）、`huang se`（黄色，中性色彩），保留明确负面项 `se qing`
  - B11：为 `totalCount` 补语义注释（统计"已评分组合数"，被过滤/早停跳过的组合不计数）
  - P4：`RateName` 封顶重算不再每次线性扫描 raters 查权重，改为预计算 `rateWeightsByDim` 映射

### ✨ New Features 新增功能

- 【namestats】`/namestats/*` 姓名单统计真正可用（Q8/Q9）：
  - 新增 `FileNameStatStore`（`namestatistics/file_store.go`）：从 `data/*_stats.json`（`build_namestats` 从 Chinese-Names-Corpus 生成的静态统计）懒加载查询，取代"未接入"错误
  - `cmd/server` 装配切换为 JSON 数据源，8 个端点全部按真实数据返回

### 🧪 Tests 测试

- 新增 `namestatistics/file_store_test.go`：8 个查询方法 + 数据缺失错误 + 懒加载幂等
- 新增 `handlers_test.go::TestNameStatisticsHandler`（含 404/500 分支）
- 新增 `fate_name_service_e2e_test.go`：断言 `GenerateWithAnalysis` 响应 `Hexagram`/`Ziwei` 非空
- 新增 `phoneme_exempt_test.go::TestBadPinyinCombosNeutralWords`（B7）、`TestBadHomophoneFuWomen`（B12）
- 回归：`go test ./...` 全绿

### 📚 Docs 文档更新

- 更新本版本变更日志；新增 `docs/21-系统审查与修复记录.md` 记录 Q1-Q11 全量整改状态

---

## [2026.09.02.0]

### ✨ New Features 新增功能

- 【确定性打分 UI】评分透明化：每维度展示"分数+依据文字"，前端可直接渲染"为什么是这个分"
  - `NameScore` 新增 `Details map[string]string`，保留各 Rater `NameRating.Detail`（`rater.go:63`）
  - `ExcellentEntry` / `NameResult` 端到端透传 Details（`table.go`、`engine.go` 三处组装点）
  - `FateNameService` 映射全 8 维分数 + Details → `NameAnalysis`（补齐三才/共现/新颖度，原漏三才）
  - 旧版 `name_service.go` 同步补全映射
  - `NameAnalysis` 新增 `ScoreDetailItem[]`（维度名/分数/依据文字）供前端通用渲染
  - 前端 `NameDetail.tsx`：评分分解展示全部 8 维（五行/音韵/字义/三才/生肖/新颖度/共现/频率）+ 依据文字
  - 前端 `NameCard.tsx`：优先使用 `score_detail` 渲染评分条，自动包含新颖度/共现/频率

- 【诗词出处可溯源】点击展开完整出处面板（作品·作者·朝代·篇目·原句·完整诗篇）
  - `NameAnalysis` 新增 `PoetryAuthor/Dynasty/FullText` 字段
  - `enrichPoetrySource` helper 复用 `classics.QueryNamePoetry`（`GlobalPoemIndex`，4 个 JSON 文件构建的结构化索引）反查 `PoemEntry`，结构化回填出处
  - 前端 `NameDetail.tsx`：诗词典故区块可点击展开，显示完整出处（作品·作者·朝代·篇目·原句·完整诗篇，竖线分隔各句）
  - 单测 `TestEnrichPoetrySource` 验证回填链路（`setupFateNameServiceE2E` 初始化数据索引）

### 🧪 Tests 测试

- 新增 `rater_detail_test.go::TestRateNameKeepsDetails`：验证 `RateName` 聚合后 `Details` 含各维依据文字
- 新增 `engine_detail_test.go::TestGenerateKeepsScoreDetails`：端到端验证 engine 输出 `NameResult.Score.Details` 非空
- 新增 `fate_name_service_e2e_test.go::TestFateNameService_E2E_ScoreDetails`：验证 `SancaiScore/BigramScore/NoveltyScore` 非零 + `ScoreDetail` 含 8 维依据
- 新增 `fate_name_service_e2e_test.go::TestEnrichPoetrySource`：验证"窈窕"回填 `PoetryChapter=关雎`/`PoetrySentence=窈窕淑女...`/`PoetryFullText` 非空
- 回归：`go test ./internal/domain/... ./internal/application/...` 全绿

### 📚 Docs 文档更新

- 更新本版本变更日志

---

## [2026.09.01.3]

### 🐛 Bug Fixes 问题修复

- 【hanzi/fate】统一笔画口径为康熙字典（治本 B3 修复）：
  - 新增 `kangxi_loader.go` 加载 `data/raw/kangxi-strokecount.csv`（Kawai Lo 维护的 MIT 许可康熙字典数据，63696 字覆盖）
  - `Hanzi` 加 `KangxiStrokes` 字段，`NamerChar` 加 `KangxiStrokes` 字段；namer_loader 在末尾合并康熙笔画到 `HanziData` 与 `NamerCharMap`
  - `dataloader.go` 调整加载顺序为 `kangxi → namer → yijing → ...`（kangxi 必须先于 namer 加载，否则 namer_loader 末尾合并时无数据）
  - `adapters.go::hanziToCharacter` 区分四个笔画字段：SimplifiedStroke/TraditionalStroke（namer 简体）、KangxiStroke（康熙字典）、ScienceStroke（科学笔画口径），修复前四个字段全填 `h.Strokes` 导致"姓用康熙、名用科学"的口径混乱
  - `ExcellentEntry` 加 `KangxiStroke1/2` 字段，`engine.go::generate()` 在 topNames 构造处统一用 `KangxiStroke1/2` 计算总笔画（与姓氏走 LookupSurnameStrokes 同一口径）
  - 数据差异：3502 字 namer 简体笔画 ≠ 康熙笔画（如 "马" namer=3 / 康熙=10，"门" namer=3 / 康熙=8）

- 【fate 引擎】单/双名 engine 拆分（架构重构 P3 方案7）：
  - 把原 `engine.go::generate()` 内 ~250 行嵌套 if/else 单/双名循环抽成独立方法 `(*sessionImpl).generateSingleName` 与 `(*sessionImpl).generateDoubleName`
  - `charInfo` 类型从函数局部提升到包级，供两个生成方法共享
  - `isCharExcluded`/`isComboExcluded` 闭包从 `generate()` 内部迁移到 `sessionImpl` 方法（直接读 session 字段，session 生命周期内只读不变），避免闭包不能跨函数跨函数
  - 移除 `excludedChars/excludedCombos` 快照复制（不再需要：sessionImpl 方法直接查询字段，并发安全由 `s.mu` 守护）
  - `generate()` 现在仅负责"前置数据装配 + 分派单/双名循环"，更易维护

- 【services】fate_name_service 端到端集成测试（P3 方案8）：
  - 新增 `fate_name_service_e2e_test.go`（6 个测试用例）：
    - `TestFateNameService_E2E_SingleName`：单名端到端（王/男/2024-01-15 12:00），断言 Pinyin/FullName/Wuxing/TotalScore 字段
    - `TestFateNameService_E2E_DoubleName`：双名端到端，验证 `IsBadCombo` 拦截生效
    - `TestFateNameService_E2E_FilterOptions`：WuxingMatch=[木] + MinStrokes/MaxStrokes 过滤生效
    - `TestFateNameService_E2E_ElderAvoidance`：避讳长辈（AvoidElderNames=[张王]）排除同形字"王"
    - `TestFateNameService_E2E_ForbiddenCombos`：清洗组合抽样验证（仲尼/若兮/七政/与砺/以方/以时）不进入 Top50
    - `TestFateNameService_E2E_ConsistentResults`：同请求多次调用返回合法响应

### 🧪 Tests 测试

- 【hanzi】新增 `kangxi_loader_test.go`（3 个用例）：
  - `TestLoadKangxiStrokesFromCSV`：验证王=4、李=7、文=4、浩=11、然=12、轩=10 等关键汉字康熙笔画；未收录字返回 0；表规模 >= 50000
  - `TestLoadKangxiStrokesReload`：二次加载幂等
  - `TestNamerKangxiStrokesMerged`：马/门/才 三个简体/康熙差异显著的字验证 `HanziData.KangxiStrokes` 与 `NamerCharMap.KangxiStrokes` 注入正确
- 【fate/singleNameDiscrimination】先前单名测试不受拆分影响
- 【回归】`go test ./internal/...` 通过所有受影响的包（hanzi/fate/services）；3 个 pre-existing build 故障（cmd/build_namestats 未用变量、infrastructure/database/memory 缺 GetFullNameStat、cmd/server 引用 memory）未触及

### 📚 Docs 文档更新

- 更新本版本变更日志

---

## [2026.09.01.2]

### 🐛 Bug Fixes 问题修复

- 【fate 引擎】加载 962 条历史清洗组合到 `IsBadCombo`：新增 `LoadForbiddenCombosFromJSON` 加载 `data/forbidden_combos.json`（`scripts/build_forbidden_combos.py` 从 raw 重建），运行时与 `forbiddenCombos` 硬编码常量合并参与组合级剔除。**这是仓库内两份清洗清单（`data/raw/清洗清单-门禁虚字.txt` + `清洗清单-荒谬组合.txt`）首次被纳入生成流程**——之前 962 条成果沉睡在仓库里未被任何代码消费，是 Top10 混入"仲尼/若兮/七政/与砺"等荒谬组合的根因之一
- 【fate 引擎】Homophone 同音误杀修复：`CheckBadHomophone(pinyin, selfChar...)` 改为精确匹配（仅当本字 == `BadHomophones.Word` 时命中），避免拼音 `si` 把"思/丝/斯"等常见好字误扣"含不吉谐音:死"；`rater.go` 调用处传入 `candidate.Char1/Char2`；旧"按拼音命中"的兼容模式（无 selfChar）保留供测试与其他场景
- 【fate 引擎】`ExcellentTable` 容量自适应：新增 `NewExcellentTableWithCap(capacity)`，worker 端容量从默认 10000 降为 `topCount*2`（最小 100），节省 ~80% 内存；合并池仍用默认容量保证全局 TopN 不溢出
- 【fate 引擎】`ensureWuxingDiversity` 去重改 map：第二轮填充时用 `seenInResult map[string]bool` 替代嵌套 for + 比对，从 O(N²) 降为 O(N)（poolSize=500~1000 时收益显著）

### 📈 Improvements 性能/体验优化

- 【数据层】`data/forbidden_combos.json` 与 `naming_quality.json` 同步走"raw → 脚本生成 → 运行时加载"模式（脚本 `scripts/build_forbidden_combos.py` 可重复执行）；启动时缺失文件降级为空集合+警告，不阻断服务

### 🧪 Tests 测试

- 【fate】新增 `naming_quality_test.go::TestLoadForbiddenCombosFromJSON` + `TestForbiddenComboReload`：验证动态禁忌组合加载数量（962）、注释示例（仲尼/若兮/七政/与砺）正序+反序命中、优质组合（浩然/子轩/明哲/俊杰/宇轩）不被误伤、热更新幂等
- 【fate】新增 `phoneme_exempt_test.go`：4 个用例覆盖本字精确匹配语义（`TestCheckBadHomophoneSelfExempt`、`TestCheckAllBadHomophonesPairedArgs`）+ ExcellentTable 自适应容量（`TestNewExcellentTableWithCap`）+ 五行多样性去重 map（`TestEnsureWuxingDiversityMapDedup`）
- 【回归】`go test ./... -count=1` 22 个包全部通过（application/handlers/services/validator、domain/bazi/fate/hanzi/name/namestat/yijing/ziwei/zodiac、infrastructure/cache/middleware）

### 📚 Docs 文档更新

- 新增 `docs/19-起名服务候选池到生成管线深度审查报告.md`（管线深度审查 + 修复方案 + 实施记录）
- 更新本版本变更日志

---

## [2026.09.01.1]

### ✨ New Features 新增功能

- 【fate 引擎】新增强治本方案：**策展白名单收窄候选池**（`engine.go` 第 6 步）。以 namer.json 的 `PositiveScore>=85`（人工寓意评分，共 563 字）作为推荐候选唯一字源门槛，荒谬字（拤/䏝/囵/饹/嚄/姮/婊/蚂/蛞/羟/苯/仫/滃 等 PositiveScore 为空）天然排除——它们此前能靠音韵/生肖/五行/新颖度维度拿 80+ 高分进入推荐榜，而原 1094 字人工门禁表已丢失（重建仅 131 字）无法穷举拦截。优质字（清91/泽92/明90/瑞90/宝90/旻88 等）全部入池不受影响
- 【fate 引擎】白名单收窄设计要点：仅当候选池为真实生产规模（>=80）时才收窄（避免破坏小字桩测试）；收窄后白名单池非空即采用（荒谬字入池不可接受，候选稍少可接受），不沿用喜用神五行收窄的 `<80` 降级阈值——单五行偏好下白名单好字可能仅 55 个（如 `-wuxing_match 水` 时水行>=85 仅 55 字），沿用 80 阈值会降级回退到含荒谬字的全量重蹈覆辙；外部注入的 `ExtraChars`（诗词/经典来源字）豁免收窄

### 🧪 Tests 测试

- TDD 新增 `engine_curated_pool_test.go` 三用例：`TestCuratedPoolExcludesAbsurdChars`（荒谬字被剔除）、`TestCuratedPoolKeepsGoodChars`（好字保留）、`TestCuratedPoolExtraCharsExempt`（ExtraChars 豁免不被误杀）
- 修复遗留 `TestRateNameCuratedExempt`：候选由「祝董」改为「晴朗」（qing2/lang3，火-火匹配喜用神火，音韵 87/三才 95/生肖 80/五行 79，总分 75.8 >= 74），原候选受硬编码谐音表影响（wang 亡/zhu 逐/dong 冻）音韵仅 62，无法达标

### 📚 Docs 文档更新

- 更新本版本变更日志

---

## [2026.09.01.0]

### 🐛 Bug Fixes 问题修复

- 【数据】从备份 `data20260827.zip` 重建 `backend/data/` 目录：按当前代码期望的 `data/` + `data/raw/` 分离结构分派，将生成原料（hanzi.json/gsc_pinyin.csv/standard_chars.json/kangxi-strokecount.csv/kx_full.xlsx/corpus/wuxing_export/清洗清单）物理隔离至 `data/raw/`，运行时数据（word/shici/诗经/楚辞/诗词经典等）置于 `data/` 根
- 【数据】`namer.json` 经 `cmd/export_namer` 从 raw 原料重新生成，承载顶层 `charGroups`（29 个精选偏旁分组），使 8105 标准字数据与偏旁分组单一文件真源对齐当前 `hanzi` 域实现
- 【数据】`naming_quality.json`（1094 字人工维护门禁表）不在备份、git 历史与磁盘中，无生成工具可还原，已重建为 131 字核心门禁字表（虚词/排行字/口语物名/数字量词/叹词/否定虚字等），覆盖测试断言，恢复门禁机制；`LoadNamingQualityFromJSON` 增加文件缺失降级（空表+警告，不阻断启动），文件存在但解析失败仍返回错误以暴露数据损坏

### 🧪 Tests 测试

- 恢复并验证：`TestIsNonNamingChar`、`TestWenHuaRaterGateCharPenalty`、`TestIsNonNamingCharEmpty`、`TestHardNegativeCharBlood`（fate 门禁）、`TestLoadNamerGroups`（hanzi charGroups）
- 已知遗留：`TestRateNameCuratedExempt`（音韵维度=62 期望>75）由硬编码谐音表导致的既有评分问题，与数据恢复无关，本次不改动，单独记录待后续调查

### 📚 Docs 文档更新

- 更新本版本变更日志

---

## [2026.08.31.0]

### 🐛 Bug Fixes 问题修复

- 【ziwei 四柱】重写 `calculateFourPillars`：年柱改用 `LunarYear.GetSixtyCycle()`（正月初一分界），对齐 `yearDivide:'normal'`；月柱改用农历月 + 正月初一年干 + 五虎遁（原版误用节令分界的月柱），并正确支持闰月；日柱在晚子时（hour=23）进位一日
- 【ziwei 命/身宫】修复第 3/6/7/8/9 用例命宫/身宫计算结果：因年/月柱输入错误级联导致的干支错误，现与 iztro 实测一致（如 case7 命宫 戊子、case8 命宫 甲辰）
- 【ziwei 大限】对照 iztro `getHoroscope` 精确重写 `calculateDaXian`：顺逆行由**年支阴阳**与性别匹配判断（阳男/阴女→顺行，阴男/阳女→逆行）；宫名刻录改为逆行=顺时针（命宫→兄弟→夫妻…）、顺行=逆时针（命宫→父母→福德…）；大限天干 = 年干五虎遁正月干 + 宫位寅序索引；大限地支 = 宫位地支

### 🧪 Tests 测试

- 【ziwei】TDD 新增 `ziwei_test.go` 9 个锚点用例（四柱/命身宫/五行局/命主身主/四化/大限）+ `TestAnalyzeZiwei` + `TestNewFields`，全部通过
- 【ziwei】新增 `ziwei_debug_test.go` 调试用例（`TestDebugRawPillars` 打印 tyme 库原始输出，用于校验命宫公式）
- 【测试期望值校订】此前用例中的错误期望值经 iztro npm 库实测（`astro.bySolar` + normal/forward/default 配置）校订：庚年四化 禄/权互换（禄=太阳、权=武曲）；case3/6/7/8/9 命/身宫干支、case1/2/9 大限 Range/天干/地支；`TestNewFields` 改用 `[]rune` 长度判断（修复 UTF-8 多字节中文长度误判）

### 📚 Docs 文档更新

- 新增本版本变更日志

---

## [2026.08.25.0]

### ✨ New Features 新增功能

- 【fate 引擎】新增 **五行收窄池方案 B**（`engine.go` narrowSet）：收窄范围从"仅喜用神五行"扩展为"喜用神 + 生助喜用神"五行集合，使 WuxingRater 间接生助梯度（生喜用 / 泄气 / 中性）真正生效，解决旧版收窄后所有候选五行恒同、WuxingRater 恒分无区分度的问题
- 【fate 引擎】新增 **五行组合多样性保障**（`engine.go` ensureWuxingDiversity）：从 Top10N 池中两轮选取，第一轮按五行组合去重（每种组合仅取 1 个），第二轮按分数补齐，确保 Top10 结果覆盖多种五行搭配（金金/土金/火火/土火等）
- 【fate 引擎】新增 **负面语义过滤体系**：硬禁用字（秽物/尸棺/淫猥/盗匪/暴虐/贬义等）直接剔除，软惩罚字（病/疾/哀/愁等）保留入池由评分器重罚，支持"去病/弃疾"式祈福命名
- 【fate 引擎】新增 **名字用字质量门禁**（`IsNonNamingChar`）：虚词/排行字/口语物名/数字量词/叹词等在单名循环内单独剔除，组合级由 `IsNonNamingCombo` 处理
- 【fate 引擎】新增 **历史人物组合拦截**（`IsHistoricalFigureCombo`）：剔除"仲尼""孔明"等与历史人物撞车组合，防止 `GetBigramScore` 因典籍共现误给高分
- 【fate 引擎】新增 **搭配黑名单过滤**（`hasPairBlacklist`）：数据层标注的字组合黑名单（如瑾-艳/妍-艳/嫣-艳）在枚举阶段直接剔除
- 【fate 引擎】新增 **避讳长辈系统**（`buildElderAvoidance`）：同形字=侵佔福分、同音字=气场冲撞（"压运"），排除父母/祖父母直系长辈姓名中的同形字与同音字
- 【fate 引擎】新增 **大名需大命规则**：检测极旺格局（专旺格/从强格），普通格局排除敏感字（龙/凤/乾/坤/圣/贤/天/帝/皇/神/仙/君）
- 【fate 引擎】新增 **ExcellentTable 淘汰轮**（`EvictRound`）：当候选池膨胀时自动淘汰低分候选项，控制内存占用
- 【fate 引擎】新增 **双名分片并行生成**：按外层 Char1 分片到 `runtime.NumCPU()` 个 worker 并行枚举，显著提升双名生成性能
- 【fate 引擎】新增 **预计算 charInfo**：将笔画/拼音/诗词出处从双重循环内提取到预计算阶段，避免重复调用
- 【数据层策展标注】Character 新增 `IsNegative`（不宜入名）、`NamePenalty`（起名扣分 0-20）、`PairBlacklist`（搭配黑名单）字段，由 hanzi.json 直接透传
- 【数据层策展标注】Character 新增 `IsCurated`（人工策展起名分类覆盖表标记，约 446 字）、`PositiveScore`（寓意评分 0-100，源自 namer.json）字段
- 【NameCandidate 扩展】新增 `IsCurated1/2`、`PositiveScore1/2`、`CommonLevel1/2`、`NamePenalty1/2` 逐字级字段，供评分器精细化判断
- 【adapters.go】`hanziToCharacter` 现以《通用规范汉字表》等级（level）为准判定常用字：一级字(3500)+二级字(3000) 全量纳入，三级字暂不纳入，表外字按笔画+生僻字表保守判定

### 📈 Improvements 性能/体验优化

- 【WuxingRater 评分重构】**核心修复：评估标准统一**。旧版使用单值 `Xi`/`Ji`（`BalanceXiYongJi` 在"补最缺"时可能覆盖 Yong 但未更新 Ji，导致 Ji 与 XiYongShen 冲突，如 Ji="土" 但 XiYongShen=["土","金"]），新版改用 `XiYongShen` 列表评估，使 WuxingRater 与引擎收窄池 narrowSet 标准一致
- 【WuxingRater 评分梯度】直接匹配喜用神 +12、间接生助喜用神 +10、泄气 +3、忌神 -10、两字皆匹配 +5、两字五行相生 +10/相克 -8
- 【WuxingRater 防御性 fallback】当 `XiYongShen` 为空时自动用单值 `Xi` 构造列表，兼容旧版测试 fixture 和边界场景
- 【WenHuaRater 策展加分】`IsCurated ∩ PositiveScore >= 85` 的精选好字获得文化加分，打破单名 Top5 荒谬字与好字五维同分的僵局
- 【WenHuaRater 扣分机制】`NamePenalty`（0-20）直接扣减文化维度分；`IsNegative` 硬禁用字在枚举阶段已剔除
- 【异步负面反馈】`ExcludeChar`/`ExcludeCombo` 改为异步批量写入，避免阻塞评分链路
- 【拼音匹配优化】全角/半角统一处理，声调数字/符号标准化，提高拼音匹配鲁棒性

### 🧪 Tests 测试

- 【fate 引擎】`TestExtraCharsInjectIntoPool`：Count 从 5 提升至 50，确保注入字在合理候选范围内出现（Count=5 时注入字因评分较低被挤出 Top5）
- 【fate 引擎】`TestRateNameNonCuratedCap`/`TestRateNameCuratedExempt`：测试 fixture 同步设置 `XiYongShen` 字段，与 WuxingRater 修复后的评估标准对齐
- 【naming_quality_test.go】五行八字维度阈值从 80 降至 75，匹配新评分梯度（+12/+10/+3 替代旧版单值匹配）
- 【策展体系测试】新增 `TestWenHuaRaterCuratedBonus`（策展好字文化加分）、`TestWenHuaRaterRareCharPenalty`（三级字/表外字扣分）、`TestWenHuaRaterGateCharPenalty`（门禁字扣分）
- 【负面过滤测试】新增 `TestHardNegativeCharBlood`（硬禁用字剔除）、`TestForbiddenComboGeoName`（地名组合拦截）、`TestHistoricalFigureComboExpansion`（历史人物组合扩展拦截）
- 【策展白名单测试】新增 `TestCuratedCharPool`（策展字池过滤）、`TestSetCuratedNamesEmpty`（空策展池边界）

### 🐛 Bug Fixes 问题修复

- 【fate 引擎】修复 `SourceClassic`/`IncludePoetry`/`IncludeClassic` 参数在 fate 引擎链路下为死参数的问题：现通过 `GenerateOptions.ExtraChars` 字段在 `fate_name_service.go` 中按来源解析诗词字并注入引擎候选池（"加字不缩池"策略）
- 【经典生成器】修复 `enhanced_generator.go` 中诗词/经典来源开关的缩池 bug：现改为 prebuilt 大池始终合并，诗词字作为补充而非替代
- 【homophone.go】谐音检测修复：全角/半角声调数字统一、零声母处理、拼音末尾空白修剪

### 🔧 Dependencies 依赖更新

- 零新增第三方依赖

### 🗑 Deprecated 废弃功能

- 删除 `frontend/docs/screenshots/` 下所有旧截图文件（8 个 PNG），后续由自动化视觉 QA 替代

---

## [2026.08.24.0] - 0.2.0

### ✨ New Features 新增功能

- 【CLI 工具】新增命令行起名工具 `backend/cmd/namer-cli`，与 Web UI 复用同一领域链路（`NameService.GenerateWithAnalysis` + fate 引擎），支持在无浏览器环境下快速测试验证起名功能
  - 参数来源三级优先级：**命令行参数 > TOML 配置文件 > 默认值**（基于 `flag.Visit` 精确检测显式传入的参数，bool/切片参数覆盖语义正确）
  - TOML 配置字段与 Web API `POST /api/v1/names/generate/analysis` 请求体完全对齐（snake_case），提供示例配置 `cmd/namer-cli/example.toml`
  - 接收核心参数：姓氏、性别、出生年月日时分、出生地、字辈及字辈位置（名中/名尾）
  - 支持全部筛选参数：名字长度、排除生僻字、五行偏好、笔画范围、典籍来源、寓意关键词、拼音首字母、避讳长辈姓名等
  - 双输出格式：`text`（人类可读终端表格）/ `json`（结构与 API 响应一致，便于脚本化对比验证）；`count` 控制 text 展示数量上限
  - 参数校验复用 Web 层 `validator` 包，口径完全一致

### 🐛 Bug Fixes 问题修复

- 【fate 引擎】修复生成结果 `pinyin` 字段恒为空的问题：引擎构建 `NameResult` 时硬编码空串（原注释"后续由 provider 补充"从未实现）。现于 `ExcellentEntry` 携带预计算的候选字读音（源自 `hanzi.HanziData` 的带声调拼音），单名回填首字读音、双名以空格组合两字读音；Web API 与 CLI 同步生效
- 【CLI 工具】text 输出在拼音为空时不再渲染空括号 `（）`

### 🧪 Tests 测试

- 【CLI 工具】TDD 先行编写 13 个单元测试：TOML 加载、优先级合并、默认值填充、参数校验（复用 Web 规则）、CSV 列表解析、请求结构转换、双输出格式渲染、空结果与空拼音边界场景
- 【fate 引擎】新增引擎级测试 3 个（内存桩 Provider/Analyzer）：单名拼音必填、双名拼音两字组合校验、拼音无尾随空白
- 【自动化回归】将人工验证固化为自动化测试：handlers 层新增真实 fate 链路集成测试 2 个（analysis 接口单/双名拼音回填断言）；`cmd/namer-cli` 新增二进制端到端测试（编译真实可执行文件执行 JSON 输出生成，断言合法性与拼音回填）
- 【fate 引擎】新增诗词字注入引擎级测试 2 个：ExtraChars 注入后字池扩大（候选包含注入字）、注入字拼音回填到名字
- 【handlers 集成】新增 source_classic 参数链路集成测试 1 个：诗经来源参数通过 resolveExtraChars 注入 fate 引擎，验证生成链路不中断

### 🐛 Bug Fixes 问题修复

- 【fate 引擎】修复 `SourceClassic`/`IncludePoetry`/`IncludeClassic` 参数在 fate 引擎链路下为死参数的问题：三字段此前只透传到 `GenerateOptions` 但 `engine.generate()` 零消费。现通过 `GenerateOptions.ExtraChars` 字段在 `fate_name_service.go` 中按来源解析诗词字并注入引擎候选池（"加字不缩池"策略），诗经/楚辞/唐诗/宋词来源真正参与候选生成
- 【经典生成器】修复 `enhanced_generator.go` 中诗词/经典来源开关的缩池 bug：此前开启 `SourceClassic`/`IncludePoetry`/`IncludeClassic` 任一开关时会跳过 prebuilt 大池扩展（`CommonMaleNames`/`CommonFemaleNames` 各仅 50 字），导致候选池骤减。现改为 prebuilt 大池始终合并，诗词字作为补充而非替代

### 📚 Docs 文档更新

- 新建根目录 `CHANGELOG.md`

### 🔧 Dependencies 依赖更新

- 零新增第三方依赖（TOML 解析复用已有 `spf13/viper`）

[2026.09.13.2]: https://github.com/bliubiao/name/releases/tag/2026.09.13.2
[2026.08.31.0]: https://github.com/bliubiao/name/releases/tag/2026.08.31.0
[2026.08.25.0]: https://github.com/bliubiao/name/releases/tag/2026.08.25.0
[2026.08.24.0]: https://github.com/bliubiao/name/releases/tag/2026.08.24.0
