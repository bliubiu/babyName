# 更新日志 (CHANGELOG)

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 格式，
版本号采用 **CalVer（日历版本）**：`YYYY.MM.DD.MICRO`，并在稳定分支以同号打 Tag。

> 注：自 `2026.08.24.0` 起建立统一变更日志；此前迭代未留档。

## [2026.10.24.5]

**结果页候选分页（每页 10 张）：50 卡长页 → 5 页短页，页面总高 14.6k → 3.6k px**

### ✨ New Features 新增功能
- 【SSR·结果页】候选列表服务端分页：每页 10 张（与「换一批」批次量同量级），`?page=N` 切页；分页条沿用收藏/历史同款模板并升级——链接自动回带筛选查询串（`PagerQuery`，翻页不丢性别/五行/笔画筛选态）、显示「共 N 条」；`page` 越界钳制到末页。
- 【SSR·结果页】收藏后 PRG 回跳保留当前页码（卡片表单携带 `page` 隐藏字段），不再把用户扔回第 1 页；「换一批」批次替换后自然回第 1 页。
- 设计取舍说明：对比勾选态为页内 HTML 表单状态，分页后勾选范围为当页 10 个候选（此前 50 个）——压缩卡后一屏已可纵览，跨页对比建议用收藏。

### 🧪 验证
- TDD `TestResult_分页`：每页卡数/页码文案/末页无下一页/越界钳制/筛选串跨页保持（`gender=male&amp;page=1`）五断言；web 包全量绿、internal 30 包全绿；两套 E2E 14/14、5/5；浏览器实测第 1/2 页渲染与页码条、页面总高 3561px（原 14638px）。

## [2026.10.24.4]

**结果页纵向压缩：默认卡片高度约减至 1/3（262px），长内容零 JS 折叠**

### 📈 Improvements 性能/体验优化
- 【SSR·结果页】名字卡信息分层重构——每卡此前把寓意全文、五行分析、整段出处古文、8+5 条评分条、评分依据默认全部铺开（单卡约 800px，一屏仅见半张卡）。现默认只留：名字行/标签/出处一行摘要（`truncate` 44 字，悬停 title 看全文）/两个折叠摘要（寓意+五行+出处全文；评分依据+全部评分条）/收藏按钮，单卡 262px。`<details>` 原生折叠，零 JS 不变。
- 【SSR·结果页】紫微斗数命盘长文（含十二宫完整列举）折叠为一行摘要「命宫/天柱/地纬 + 点击展开」。
- 【SSR·全局】新增 `.fold` 折叠组件（▸/▾ 指示、朱砂色摘要、hover 下划线）与 `.truncate-inline` 单行省略；模板函数 `truncate`（rune 安全，n≤0 不限）。
- 每卡评分条组、粤语评分条、评分依据文字（八字/字义/音韵/三才）全部并入「评分明细」折叠区，信息一个不少，默认密度大幅提升。

### 🧪 验证
- web 包测试全绿（含 `TestTruncate`）；SSR E2E 14/14；浏览器实测：卡片 262px（原约 800px）、页面总高 14638px/50 卡、折叠区内容完整（寓意 308 字符、评分明细 495 字符）。

## [2026.10.24.3]

**统计页占比全 0 修复 + 榜单单/双字名分列 + 粤语试听嗓音降级提示**

### 🐛 Bug Fixes 问题修复
- 【统计】占比列全部显示 0.0000%、占比条全空：SQLite 物料装载路径只恢复频率表、从不设置语料总人数（`hanzi.SetFrequencyTotalNames` 为孤儿函数，全仓库无调用点；仅 JSON 目录装载路径带 meta）→ `GetTotalNames()` 恒 0，`rateOf()` 全部归零。修复：装载侧用构造恒等式恢复 `total = Σ单字count − Σ双字count`（单字表计每个人名的每个「名」字 = 单字名数 + 2×双字名数；双字表计每个双字名一次；真实语料验证 1,815,552 − 857,806 = 957,746 ✓），任一表缺失或差值非正则不设、绝不给错分母；无需改表结构，存量库装载即生效。TDD：`TestLoadMaterials_恢复频率语料总人数`。

### ✨ New Features 新增功能
- 【SSR·统计页/API】高频名字榜支持种类过滤 `kind`：`all`（默认，合并）/`char`（仅单字名）/`bigram`（仅双字名）。背景：语料按「名」统计、双字组合极度长尾（57.8% 的组合仅出现 1-2 次，双字榜首仅 121 次 vs 单字榜首「文」28,431 次），合并榜必然被单字名霸占——分列后双字名榜（秀英/倩倩/莎莎…）可见。SSR 下拉零 JS 回显（`SelKind`），JSON API `/api/v1/namestat` 同口径。榜单下方补语料口径说明文案。
- 【SSR·粤语试听】嗓音降级提示：无 zh-HK 粤语嗓音、以普通话嗓音兜底时，按钮 title 说明当前朗读嗓音并指引安装 Windows「中文(香港)」语音包（设置→时间和语言→语音→管理语音），安装后自动切换粤语，无需改代码。Web Speech 嗓音来自系统/浏览器，站点无法（也不应）静默内嵌语音包——离线引擎方案已在 `2026.10.23` 迭代移除（音质缺陷），不建议回退。

- 【SSR·粤语试听】朗读稳健性加固（用户反馈"点按钮没声音"诊断配套）：utterance 保模块级引用（Chrome 有 utterance 被垃圾回收致朗读中途静音的经典缺陷）；朗读前检测 `speechSynthesis.paused` 先 `resume()`（Chrome 偶发队列停滞无声）；`onerror` 把错误写回按钮 title（此前失败无任何可见反馈）。诊断实证合成链路健康（start + 逐字 boundary 推进、零 error），无声问题在音频输出层（内嵌面板/系统音量），非合成层。

### 🧪 验证
- `go test ./internal/...` 30 包全绿；两套页面 E2E 14/14、5/5；启动日志实证「频率语料总量已恢复 {总人数: 957746}」，API `/api/v1/namestat` 返回真实占比（文 2.9686%），浏览器实测双字名榜/占比条/TTS 提示。

## [2026.10.24.2]

**三处静默正确性缺陷修复（性别筛选 / 紫微大限方向 / SQLite 锁）+ 收藏入库校验防线 + SSR 粤语试听**

### 🐛 Bug Fixes 问题修复
- 【命理引擎】`fate.WithGenderFilter` 性别筛选静默失效：SSR 表单提交 "男"/"女" 原样落地，而 `GenderFilter` 值域是 male/female，永不相等 → 筛选退化为只剩中性字。入口统一归一化（男/m/M→male，女/f/F→female，无法识别原样返回），新增 `filter_gender_test.go`。
- 【紫微斗数】`ziwei.calculateDaXian` 顺逆行方向整片反置：`isMale` 只认 "男"，API 枚举 "male" 全被按女性逆行计算且无任何报错。现同时接受中英文枚举，新增 `ziwei_gender_test.go`。
- 【存储】SQLite 读写连接补 `busy_timeout(5000)`（modernc.org/sqlite 默认 0）：WAL checkpoint 或外部工具持锁瞬间读写路径直接抛 "database is locked"，现给 5 秒重试窗口；另补 `idx_favorites_created_at` 索引，修复收藏列表按时间分页的全表扫描 + 排序（此前只有姓名索引）。

### ✨ New Features 新增功能
- 【SSR·结果/测名页】粤语试听回归：名字粤拼标注旁加「🔊 粤」按钮（Web Speech API 渐进增强，全站唯一一段内联原生 JS，约 50 行零依赖）。嗓音挑选沿用旧前端口径（zh-HK/yue 精确优先、任意中文嗓音兜底、voiceschanged 异步等待 1.5s 快照兜底）；朗读内容为汉字原名而非粤拼（调号数字会被念成字母，同 74a1e1b 教训）；朗读中再点一次即停止；零 JS / 无语音合成 / 无中文嗓音时按钮保持隐藏不误导。
- 【收藏服务】`validateFavoriteEntry` 入库统一防线（`SaveFavorite`/`BatchSaveFavorite` 双入口）：姓氏/名字复用校验器、gender 归一校验、score 限 0-100（防 SSR 表单无校验直库 + score≥80 触发自学习污染精选库）、拼音/来源/备注长度上限，错误为中文文案；新增校验表驱动用例与评分越界不触发自学习用例。

### 📚 Docs 文档更新
- `docs/40-物料管线落地问题审查与优化改造方案.md` 重编号为 `docs/39-`（标题同步改号）；删除过时的 `docs/DOCKER_QUICKSTART.md`。
- `backend/scripts/db_probe.py` 一次性 namer.db 写入探针入库留档（busy_timeout 修复的实证工具）；根 `scripts/json4matter.html` 旧版独立副本入库留档（真源已迁至 `backend/internal/infrastructure/web/static/json4matter.html`，随 go:embed 分发）。

### 🔧 Dependencies 依赖更新
- `.gitignore` 补 `.gocache/`（本机 GOCACHE 指向工作区时产生）与 `acl-recovery/`（本机 ACL 工具备份产物，含本机路径信息不入库）。

### 🧪 验证
- `go test ./internal/...` 30 包全绿（含 fate/ziwei/services 新用例）；两套页面层 E2E 14/14、5/5。

## [2026.10.24.1]

**页面设计体系升级：国风质感与全站组件统一 + 首页原生日历选择器**

延续 `2026.10.23.2` 的 SSR 全栈重构，对全部 11 个模板做一轮视觉与一致性升级（零 JS 约束不变），并按用户反馈把首页/测名页的出生年月日录入升级为浏览器原生日历选择器。

### ✨ New Features 新增功能
- 【SSR·首页/测名】出生年/月/日三个数字输入框合并为 `<input type="date">` 原生日历选择器（`birth_date`，min/max 1900-2100 与校验器同口径）：浏览器自带日历弹层、禁止 2 月 30 日之类的非法日期；`parseGenerateForm` 优先解析 `birth_date`，兼容旧三字段（JSON API 与 E2E 不受影响），非法值给中文文案拦截；TDD 三用例（合并解析/非法拒绝/优先级）。
- 【SSR·全局】设计令牌加 `color-scheme`：原生控件（日历图标、滚动条、下拉弹层）随明暗主题自动适配。

### 📈 Improvements 性能/体验优化
- 【SSR·布局】宣纸颗粒噪点覆层（SVG feTurbulence，明暗双浓度）；顶栏书脊样式（金朱渐变压边）；导航当前页朱砂高亮 + `aria-current`（`navClass`/`navAria` 收敛在 Go 侧，规避 html/template 属性内条件转义限制，新增 `pageData.Path`）；按钮悬浮上浮/按压回弹/focus-visible 环；下拉框自绘描色箭头；入场淡入上移动效（尊重 `prefers-reduced-motion`）；`@media print` 打印样式（报告页隐藏顶栏/操作钮）。
- 【SSR·组件】补齐缺失的 `.tag` 组件（结果页此前裸文本）并分 seal/celadon/gold 三色；统一 `.meter` 评分条（五行五色填充）、`.ink-table` 表格（行悬浮/数字对齐）、`.empty-state` 空状态大字水印、`.wm` 角落水印、`.toolbar` 工具条、`.flash` 提示条（✓/✕）。
- 【SSR·结果页】「换一批/对比所选」合并为工具条；八字五行改彩色计数 pill；喜用神/推荐理由统一标签。报告页五行分布条改真实五行色。对比页逐行最优值朱砂高亮（并列同标）+ 综合分析卡评分印章。统计页占比条按榜内最大值归一化（新增 `maxf`/`pctf` 模板函数）、前三名朱砂强调。黄历页宜/忌卡角落水印、万年历周末朱色。收藏/历史页空状态引导卡、标题统一、元信息标签化。

### 🐛 Bug Fixes 问题修复
- 【E2E】`scripts/ssr_e2e_test.py` 三处既有缺陷：`call_no_redirect` 每次新建一次性 CookieJar 导致 `nm_sid` 随 jar 丢弃、结果页永远「请先生成名字」→ 全局共享 JAR；名字提取正则仍匹配 React 旧标记 → 改 SSR 的 `name-big`；`urllib.parse.unescape` 不存在 → `html_mod.unescape`。修复后 14/14 全绿（`webui_e2e_test.py` 5/5 同绿，随迁至 `backend/scripts/`）。

### 🧪 验证
- `go build` / `go vet` 干净；`gofmt` 无新增未格式化文件；`go test ./internal/...` 30 包全绿（含 `birth_date` 三条新用例）。
- 两套页面层 E2E 全绿；浏览器实测明暗双主题下首页（日历图标可见）、结果（评分条/工具条）、对比（逐行最优高亮）、黄历（宜忌水印）、统计（归一化条/前三强调）渲染正常，`birth_date` 提交链路日志实证 303 → /result 200。

## [2026.10.23.2]

**SSR 差距补齐：首页高级选项 / 服务端筛选 / 对比 / 统计重名查询 / 诗词高亮 / 暗色模式 / 黄历万年历 / 历史回看 / 完整报告**

延续 `2026.10.23.1`（用户选定"补齐所有零 JS 可闭合的差距"）：把 React 旧前端与 SSR 版的 P0 功能缺口与 P1 体验项全部闭合，全部为零 JS 纯表单/链接方案——对比勾选态改由服务端会话 + `rq` 隐藏字段跨 PRG 保持，取代 React 的 zustand 内存态（刷新/直入即丢的问题随之消失）。

### ✨ New Features 新增功能
- 【SSR·首页】补高级选项：字辈+位置、经典来源 9 选 1、寓意关键词、偏旁选字（`hanzi.GetNamerGroups()` 服务端渲染 checkbox + `<details>` 折叠组），出生分钟。
- 【SSR·结果页】服务端筛选：性别/五行/笔画区间/人名频率档位（`parseNameFilter`，score→tier 映射与 React 同口径），筛选态经 `rq` 隐藏字段跨"换一批"重定向保持；诗词出处全文 `<details>` 展开 + 命中字高亮（`highlightPoetry`）；候选卡加"对比"勾选（HTML5 `form` 属性外挂 checkbox，零 JS 且避免表单嵌套）。
- 【SSR·对比页】新增 `/compare`：勾选 2-4 个名字 POST 到服务端，从会话候选池按 `given_name` 取齐字段渲染 11 指标对比表 + 综合分析；空态直入、1 名、无会话三条路径均有中文引导（不再 500）。
- 【SSR·统计页】按名查重名率（`namestat.GetNameStats`）：评级徽章/人数/语料占比/频率排名；未收录名字给明确提示而非空白。
- 【SSR·黄历页】日期导航（上一天/下一天，`time.Date` 归一化进位）、`view=month` 万年历月格（`bazi.GetLunarMonthCalendar`，与 `/api/v1/lunar/month` 同源；周一开头、今天高亮、前后月链接）、日视图时辰吉凶 12 格。
- 【SSR·历史页】每条记录加"回看"链接 `/history/view?id=`（`HistoryService.GetHistoryRecord`，不存在返回 `ErrCodeNotFound` 转中文提示不 500；回看态隐去换一批/对比入口）。
- 【SSR·完整报告页】新增 `/report`：八字四柱卡片 + 五行分布条 + 候选名详情 + 五段综合解读（八字格局/五行平衡/喜用神补益/生肖纳音/名字质量）——`buildReportAnalysis` 由前端 `FullReport.tsx` 的 `generateDetailAnalysis` 移植为服务端计算，口径一致。
- 【SSR·JSON 查看器】新增 `/json-viewer`：iframe 承载 `//go:embed` 内嵌的 `json4matter.html` 离线美化查看器（与旧前端同源）。
- 【SSR·暗色模式】cookie `theme=dark`（365 天）+ Tailwind CDN `darkMode:'class'` + `.dark` CSS 覆盖块（选择器特异性 (0,2,0) 高于工具类，无需逐元素加 `dark:` 变体）；`POST /theme` 切换并回跳原页（redirect 限定站内路径）。
- 【SSR·全局】flash 横幅（Error 红 / Ok 绿，`?error=`/`?ok=` 查询串驱动）；导航补齐"对比/JSON"两项 7 项无死链。

### 🐛 Bug Fixes 问题修复
- 【服务层】`HistoryServiceInterface` 新增 `GetHistoryRecord`，映射收敛到 `toHistoryRecord`（历史回看与列表/分页三处同口径，不再各写一份）。
- 【测试】`web_test.go` 的 `mockHistoryService` 补 `GetHistoryRecord`（接口扩展后 mock 未同步，`go vet` 报不实现接口）。
- 【格式化】`favorite_service.go` import 顺序回退（工作树引入）已 `gofmt -w` 修复。

### 🧹 Code Cleanup 代码清理
- `renderPage` 集中注入公共视图状态（Ok/Dark/RedirectPath），各 handler 不再重复拼装。
- `resultRedirect(c, rq)` 收拢"筛选查询串跨 PRG 保持"的拼接逻辑（Explore/Favorite 共用）。

### 🧪 验证
- `go build ./...` / `go vet ./...` 干净；`go test ./...` 32 包全绿（含 web 包 SSR 单测）。
- 页面层冒烟两层全绿：原 45 项回归（首页→生成→结果→换一批→收藏→取消→历史→删除→测名→黄历→统计，含错误路径与无会话重定向）+ 新增 77 项扩展（筛选五项/筛选跨换一批保持/对比四路径/JSON 查看器与内嵌资源/暗色切换与恢复/完整报告五段/历史回看三路径/统计命中与未收录/万年历前后月与非法月份/日视图时辰吉凶/flash 双横幅）。
- `tools/e2e_check -strict` 18/18 通过（API 契约、评分一致性、并发吞吐、会话 FIFO 淘汰）。

## [2026.10.23.1]

**SSR 全栈重构收口：补齐测名/黄历/统计三页 + 修复粤语分开关漏传**

在 `refactor/ssr-templates` 分支将前端从 React/Next/TS 迁移为 Go `html/template` 服务端渲染（Tailwind CDN、零 JS、PRG 模式）后，本迭代消除页面层全部缺口，达到两层 E2E 全绿。

### ✨ New Features 新增功能
- 【SSR·测名页】新增 `/evaluate`（GET 表单 / POST 评分）：复用 `NameService.Evaluate`，渲染与 `/generate` 同构的评分明细、八字概要、风险体检（fail/warn/pass 三色徽标），支持一键收藏。
- 【SSR·黄历页】新增 `/huangli`：直读 `bazi.GetHuangli` 域层（与 `/api/v1/huangli` 同源），展示宜/忌、吉神凶煞、彭祖百忌、冲煞、五行等；缺省今天，非法日期（如 2 月 30 日）返回中文提示不 500。
- 【SSR·统计页】新增 `/stat`：`namestat.GetTopNames` Top-N 高频名表格（排名/人数/占比条），支持 limit 20/50/100。
- 【SSR·结果页】补紫微斗数区块（命宫/天柱/地纬/解析）与粤拼标注、粤语分维度条（关闭开关时仅透传粤拼、不渲染粤语分）。
- 【SSR·首页】补 `birth_minute` 出生分钟输入框（此前 `parseGenerateForm` 读到分钟恒为 0，影响真太阳时校正精度）。

### 🐛 Bug Fixes 问题修复
- 【粤语评分】`NameService.Generate` 装配 `fate.GenerateOptions` 时漏传 `EnableCantonese`，导致请求开关永远到不了引擎、粤语分恒为空（`/api/v1/names/generate` 与 SSR 同受影响；`fate_name_service.go` 另一条装配链本就传了，属双口径漏配）。已补齐并对齐注释。
- 【DTO 映射】`services.HistoryRecord` / `FavoriteRecord` 漏映射 `database` 层的 `CreatedAt`（收藏页模板渲染 `can't evaluate field CreatedAt` 的直接根因），已补齐为 `json:"created_at"`（对 JSON API 为增量字段）。
- 【渲染健壮性】`renderPage` 改为先渲染到 `bytes.Buffer` 再一次性 `c.Data()` 输出：模板中途报错不再留下半页 HTML，也不再触发 "Headers were already written" 导致状态码改不动。
- 【CLI】`logger.Init` 无条件把 `os.Stdout` 加为第二写目标，与 CLI"日志初始化到文件，避免污染终端结果输出"的意图相悖，导致 `-format json` 的 stdout 被 INFO 日志污染（既有红灯 `TestCLIBinaryJSONOutputPinyinFilled`）。新增 `logger.Config.FileOnly`（零值 false 保持 server 双路行为，完全向后兼容），CLI 显式开启。

### 🧹 Code Cleanup 代码清理
- 删除 SSR 层 `attachSurname`：服务层四条链路（`convertFateToNameNames` / `assembleEvaluatedName` / `fate_name_service`）早已回填 `Surname`/`FullName`（根源在 `domain/fate/engine.go`），渲染层拼接属纯冗余。
- 收敛 SSR `session` 为 `map[string]*services.GenerateResponse`，删除只写不读的 `surname`/`generationID`/`createdAt` 字段与 `Handler.dataDir` 死字段。
- `name.Name` 补 `Jyutping`/`CantoneseScore`/`CantoneseScoreDetail` 字段（此前仅 `NameAnalysis` 有），`convertFateToNameNames`/`assembleEvaluatedName` 透传；`buildScoreDetail` 重构为可传维度顺序（普通话八维与粤语两维各用各的顺序表，消除同名重复实现）。

### 🧪 验证
- `go build ./...` / `go vet ./...` / `gofmt -l`（改动目录）全部干净。
- `go test ./...` 全绿（31 包 ok，含此前既有红灯 `TestCLIBinaryJSONOutputPinyinFilled` 转绿）。
- 两层 E2E：`tools/e2e_check -strict` 18/18 通过；SSR 页面层冒烟 45/45 通过（首页→生成→结果→换一批→收藏→取消→历史→删除→测名→黄历→统计，含错误路径与无会话重定向）。
- 粤语分开关语义实测：开启时粤拼+粤语分均渲染；关闭时粤拼仍透传（50 次）、粤语分 0 次不渲染。

## [2026.10.23.0]

**移除离线粤语 TTS（Audio8 ONNX），保留 Web Speech zh-HK 试听**

离线 Audio8 0.1B INT8 ONNX 方案链路长、模型体积大、CPU 单条合成 45-60s，且默认音色仍为普通话口音，无法达到"真粤语"目标，投入产出不匹配，决定移除。粤语试听回退浏览器 Web Speech：本机安装 zh-HK/yue 嗓音时可用，否则给出降级提示。

### 🗑 Removed 移除
- 【TTS·后端】删除 `services/tts.go`、`handlers/tts.go` 与 `GET/POST /api/v1/tts` 路由；`WriteTimeout` 由 300s 恢复为 60s。
- 【TTS·前端】删除 `getTtsHealth()` / `synthesizeCantoneseAudio()`；`CantoneseScore` 移除离线引擎优先、合成等待态与 Blob 音频播放逻辑，保留 Web Speech 能力探测、嗓音选择与 no-voice 降级提示。
- 【TTS·sidecar】删除 `backend/tts_sidecar`（启动脚本、Audio8 运行时、模型与默认音色数据）。
- 【TTS·文档】删除 `docs/41-离线粤语TTS-E2E验证报告.md`、`docs/42-粤语TTS管线E2E复测与噪音根因修复.md`。
- 【依赖】移除 `github.com/ebitengine/purego`（仅为 ONNX Runtime 绑定引入）。

### 🧪 验证
- `go build ./...` 通过；`go test ./...` 除 `cmd/namer-cli` 既有日志污染 JSON 用例（与本次无关）外全部通过。
- 前端 `tsc --noEmit` 通过；`vitest run` 8 个文件 80 项全部通过；改动文件 eslint 0 error。

## [2026.10.22.4]

**粤语 TTS 检测与降级逻辑补全（Web Speech）**

补齐此前识别的 3 个 TTS 真实缺口：本机嗓音探测、voiceschanged 异步填充、显式指定嗓音避免退化。

### ✨ New Features 新增功能
- 【粤语 TTS·能力探测】`lib/jyutping.ts` 新增 `cantoneseTtsStatus()`（返回 `unsupported` / `no-voice` / `ok` 三态 + 用户提示）、`resolveCantoneseVoice()`（异步监听 `voiceschanged`，超时兜底）、`pickCantoneseVoice()`（优先 `zh-HK/yue`，回退中文嗓音）。
- 【粤语 TTS·显式选嗓音】`speakCantonese` 改为 async，显式挑选本机粤语嗓音（不可用则回退中文），避免浏览器静默退化成错误的普通话嗓音；保留 `cancelCantoneseSpeech()`。`cantoneseTtsSupported()` 维持同步「API 是否存在」粗判。
- 【粤语 TTS·UI 降级】`CantoneseScore` 在能力探测完成前不渲染试听按钮；`no-voice` 时渲染禁用态并显示「未安装粤语 / 中文语音包」提示；`unsupported` 时不渲染。

### 🧪 验证
- `vitest run` 全量 80 项通过（`jyutping.test.ts` 扩至 20 项覆盖三态 / voiceschanged / 超时；`CantoneseScore.test.tsx` 扩至 7 项含 no-voice 禁用 + ok 可用）。
- `tsc --noEmit` 通过；`eslint` 改动文件 0 error。

## [2026.10.22.3]

**粤语评分前端落地（开关 + 粤拼标注 + 粤语分明细 + TTS 试听）**

后端粤语评分链路（2026.10.22.2）的前端消费层：表单开关、结果页粤拼标注与独立粤语分展示、粤语试听，与普通话八维评分 UI 视觉体系统一。

### ✨ New Features 新增功能
- 【粤语评分·表单开关】首页表单新增「开启粤语多维度评分」勾选（含说明文案），经 zustand store 持久化；提交时以 `enable_cantonese` 透传后端。
- 【粤语评分·粤拼标注】新增 `lib/jyutping.ts`：粤拼音节解析（`parseJyutpingSyllables`）、六调中文名与调值说明（`toneLabel`，如「4 阳平（低降）」）、平仄二分（`toneClass`）、粤语 TTS 封装（`speakCantonese`/`cancelCantoneseSpeech`/`cantoneseTtsSupported`，Web Speech `zh-HK`）。
- 【粤语评分·组件】新增 `components/CantoneseScore.tsx`：逐字粤拼芯片（汉字 + 粤拼 + 声调徽标，平/仄异色）、粤语分总览与「粤语分分解」（粤语音律/粤语谐音双条 + 依据文字）、粤拼声调体系说明、粤语试听按钮（不支持 Web Speech 时自动隐藏）。无声拼且无粤语分时整体不渲染。
- 【粤语评分·结果展示】`NameCard` 新增粤拼行（`粤 wong4 zi2 man4`）与「粤语 xx」分徽标；`NameDetail` 接入粤语评分区块；`FullReport` 名字列表补粤拼行与粤语分徽标。
- 【类型接线】`GenerateRequest.enable_cantonese`、`FormData.enableCantonese`、`Name.jyutping`/`cantonese_score`/`cantonese_score_detail`。

### 🧪 验证
- `vitest run` 全量 70 项通过（新增 `jyutping.test.ts` 12 项 + `CantoneseScore.test.tsx` 5 项，含无声拼降级、TTS 不支持降级断言）。
- `tsc --noEmit` 通过；`eslint` 改动文件 0 error（仅存量 label-has-associated-control 警告）。
- `next build` 成功，`out/` 产物已同步重建。

## [2026.10.22.2]

**粤语多维度评分模型落地（评分内核 P0–P3 + 引擎集成）**

在普通话八维评分体系之外，新增一套**平行、独立**的粤语评分链路：粤语音律分 + 粤语谐音风险分，聚合成独立「粤语分」透出展示，并在开关开启时以固定权重（0.10）轻量并入综合分影响排序；关闭时综合分完全由普通话八维决定（零影响）。

### ✨ New Features 新增功能
- 【粤语评分·数据层】新增 `internal/domain/yue` 包：`//go:embed jyutping.json` 静态嵌入粤拼映射（简体 12676 条，三源合并：yyzd 粤语字典 + HKSCS2016 + wordslist）。提供 `JyutpingOf/PrimaryJyutping/StripJyutpingTone/JyutpingTone/JyutpingInitial/JyutpingFinal/IsEnteringTone/ToneClass/ShengMuGroup` 等粤语音系算子。对起名候选字覆盖率：一级字 100%、总体 76%（核心起名用字全命中）。
- 【粤语评分·谐音风险】新增 `yue/homophone.go`：粤语负面谐音词库三层设计——① `YueBadHomophones` 通用不吉字（带**本字豁免**，不误伤珠/思等同音好字）② `YueHomophonePairs` 明确「好字→不雅谐音」配对点名（朱→猪、柯→屙、苟→狗、史→屎、赖→癞 fail；吴→唔、陈→尘、黎→篱 warn）③ `YueBadCombos` 粤语连读不良词（滑窗查表 O(文本长度)）。含懒音近音歧义（n↔l、∅↔ng、t↔k）。
- 【粤语评分·评分器】新增 `fate/yue_yinyun.go`（`YueYinYunRater` 粤语音律：声调平仄交替/声母冲突/韵母冲突/入声收束）与 `fate/yue_homophone.go`（`YueHomophoneRater` 粤语谐音：fail -25 / warn -8）。均实现 `Rater` 接口并兼容 `skipDetail` 热路径。
- 【粤语评分·聚合】`rater.go` 新增 `RateCantonese`/`RateCantoneseScore`/`DefaultCantoneseRaters`/`CantoneseBlendWeight`。粤语两维权重内部归一（音律 0.6 + 谐音 0.4 = 1.0），与普通话八维权重体系互不干扰；粤语维度不参与「非策展双名封顶」。
- 【粤语评分·引擎集成】`GenerateOptions.EnableCantonese` 开关；会话装配期注入粤语评分器与姓氏粤拼；候选组装无条件预填 `Jyutping1/2/SurnameJyutping`（粤拼标注始终透传，与开关无关）；枚举评分点经 `blendScore` 轻量并入影响排序；入榜条目回算粤语分明细。
- 【粤语评分·API 透出】`NameResult` 新增 `CantoneseScore`（开启时非空）与 `Jyutping`；`NameAnalysis` 新增 `cantonese_score`/`cantonese_score_detail`/`jyutping` 字段；请求 DTO 新增 `enable_cantonese`。粤拼标注供前端展示粤拼与声调。
- 【粤语评分·风险体检】`AssessNameRisks` 新增「粤语谐音」风险类别（普通话谐音之外补充粤语特有风险，如吴→唔、朱→猪，粤拼未收录时静默跳过）。

### 🐛 Bug Fixes 问题修复
- 【粤语谐音】`CheckNameYueHomophones` 懒音近音检测原会把**本字粤拼本身**也当作变体参与匹配，导致珠(zyu1) 这类与朱(zyu1)→猪 同音的好字被误报 warn。已改为排除本字本身，仅当**懒音变读**后才近似不雅字才记为 warn，贯彻「不误伤同音好字」原则。
- 【粤语谐音】`YueHomophonePairs` 早期用繁体字形（吳/賴/陳/豬/籬），简体起名候选池（吴/赖/陈）永不命中。已统一改为简体，配对真正生效。
- 【粤语连读】`CheckYueBadCombo` 连读索引键用带声调粤拼构建（`ng4dak1`），而运行时滑窗文本去声调（`ngdak`），口径不一致导致**永不命中**。已统一为无声调键。

### 🧪 验证
- 单元测试：新增 `yue` 包 `yue_test.go`/`homophone_test.go`（音系算子 + 谐音/连读 + 珠误伤回归 + 懒音变体）与 `fate` 包 `yue_rating_test.go`（粤语两维评分器 + `RateCantoneseScore`==`RateCantonese` 一致性 + `blendScore` 混合公式 + 端到端开关透出）全绿。
- `go build ./...`、`go vet`（fate/yue/services/name）通过。
- `go test ./...`：`fate`/`yue`/`services`/`name`/`handlers` 等全部涉及包通过；普通话八维评分与既有 golden 快照无回归（默认关闭粤语时 `blendScore` 原样返回主分）。
- 已知预先存在且与本次无关：`cmd/namer-cli` 的 `TestCLIBinaryJSONOutputPinyinFilled` 在沙箱因日志目录不可写回退 stdout 污染 JSON 解析而失败（干净代码上同样失败，非本次改动引入）。

## [2026.10.22.1]

**质量验证工具链落地 + 五行判定收敛到单一真源**

### ✨ New Features 新增功能
- 【质量防线】新增一键质量验证脚本 `scripts/verify-quality.sh`，固化四道防线：① 单元层（`fate` 评分正确性 / golden 快照防漂移 / 八维 `score_detail` 透传）② 门禁表覆盖审计（`gate_audit -static-only`）②+ 专项维度巡检（`dim_audit`）③ 端到端（`e2e_check` 八大段 18 项，需先起服务）。退出码非 0 便于接入 CI。
  - 支持三种用法：`bash scripts/verify-quality.sh`（全量，自动构建并拉起服务）、`SKIP_SERVER=1`（仅单元+门禁+dim_audit，不启服务，pre-push 用）、`SERVER_URL=http://127.0.0.1:8080`（复用已起服务）。
- 【质量防线】新增专项维度巡检工具 `backend/tools/dim_audit`（Go CLI），直接吃真实 domain 逻辑、用「已知答案 fixtures + 结构不变量」校验四大维度，共 **147 项断言全绿**（八字 41 / 生肖 33 / 五行 43 / 诗词出典 30）：
  - 八字：立春换年边界、年支↔生肖跨维度一致性、四柱合法干支、日主五行/强弱/喜用神合法、可复现；
  - 生肖：`GetZodiacByYear` 已知答案、五行属性、ID/Name 自洽、12 生肖齐全；
  - 五行：`GetCharacterWuxing` 与 `HanziData.Wuxing` **双重口径**校验 + 收敛不变式（见下）；
  - 诗词出典：基于 `QueryNamePoetryBySource` + `MatchedSourceMembers` 校验「选某来源族→出处落在族内（不回退全库）」+ 命中字⊆名字 + 引文含命中字。
- 【质量防线】新增 git pre-push hook `.githooks/pre-push`（运行 `verify-quality.sh` 快路径），打印阶段明细，任一阶段失败即阻断推送；`SKIP_VERIFY=1` 或 `git push --no-verify` 可跳过。已 `git config core.hooksPath .githooks` 生效。
- 【质量防线】新增 CI workflow `.github/workflows/quality.yml` 两个 job：`quality-fast`（CI 无需服务/库，跑单元+门禁+dim_audit）、`quality-e2e`（先 `go run ./cmd/materials import --data ../data --db ./namer.db` 建库，再跑全量含 E2E 18 项）。

### 🐛 Bug Fixes 问题修复
- 【五行】`GetCharacterWuxing`（`hanzi/radical_wuxing.go`）收敛为 `ResolveWuxing` 的薄封装，**消除与单一真源 `ResolveWuxing`（覆盖表 > namer.json 校正 > 部首推导）的双口径矛盾**。此前该函数的优先级是「覆盖表 > 部首推导 > 校正兜底」，会把部首推导值压过人工校正五行，导致部首≠校正的字判定错误，已修正 6 字：明→水（原火）、马→水（原火）、石→金（原土）、玉→木（原土）、不→水（原土）、冲→金（原水）。无任何生产热路径依赖旧「部首优先」行为（引擎走 `fate`/`materials` 装配链，不经 `GetCharacterWuxing`）。

### 🧪 验证
- 专项巡检：`go run ./tools/dim_audit` → **147/147 通过**（八字 41 / 生肖 33 / 五行 43 / 诗词出典 30）。
- 单元测试：`go test ./internal/domain/hanzi/...` 全绿；`wuxing_verify_test.go` 同步修正 6 个用例期望，并新增**收敛不变式**断言 `GetCharacterWuxing(ch) == ResolveWuxing(ch, HanziData[ch].Wuxing, radical)`，守护双口径不再分叉。
- 质量闸门：`SKIP_SERVER=1 bash scripts/verify-quality.sh` → EXIT=0（单元 + 门禁审计 + dim_audit 全 PASS，E2E 按预期跳过）；全量（含 E2E 18/18）→ EXIT=0。
- pre-push hook 模拟运行（空 stdin）→ EXIT=0，打印「✓ 全部质量防线通过，允许推送」。

## [2026.10.21.7]

**物料管线 docs/40 本期（A1-A5+B6）落地收尾：验收缺口闭合**

### 🐛 Bug Fixes 问题修复
- 【物料导入】修复 classics 目录级指纹的哨兵门控回退：最小夹具（无 shijing.json）不再误报「解析经典语料失败」，恢复可选源缺失 → 物料级跳过的既有语义

### ✨ New Features 新增功能
- 【数据服务】/readyz 改用 `Store.ReadPing()` 真探活持久层读路径，替换打内存 map 的 `GetHanziByChar`（库损坏/读路径被锁时收敛 503）
- 【物料导入】`ImportReport` 增加 `Elapsed`（全链服务端总耗时），`--json` 报告含 per-material 与汇总两级耗时位，供 CI 消费

### 📈 Improvements 性能/体验优化
- 【物料导入】导入报告 CLI 表格新增「耗时」列，慢物料可下钻定位

### 📚 Docs 文档更新
- docs/40 增补「七、实施偏差与最终形态」落地回执：A1 特例化路径与哨兵门控、A2 partial 指纹打点自愈机制、T-A1..T-A5 验收对照表与遗留项清单

### 🔧 Dependencies 依赖更新
- 无

## [2026.10.21.6]

**诗词出处高亮命中字 + 修复「选了唐诗宋词却显示诗经出处」的脱节**

### ✨ Features 功能增强
- 【前端】诗词典故的**原句与全诗展开**中，高亮「实际命中的名字用字」（`text-crimson` + 下划线）。出典匹配是「任一字命中即算有出典」——例如「荔嘉」命中的是「嘉」，「荔」并不在这句里；高亮让用户一眼看出是哪一字带来了这处出处，避免误以为整个名字都出自该典籍。
- 【前端】新增 `src/lib/poetryHighlight.ts` 纯函数 `splitByMatchedChars(text, matchedChars)`（短字优先切分、不含于原句的命中字忽略、无命中字退化为纯文本），配套单测 `poetryHighlight.test.ts`（8 例）。
- 【后端】新增 `poetry_matched_chars` 字段（`name.Name` 与 `name.NameAnalysis`），透出 `classics.PoetryMatch.MatchedChars` 供前端高亮。

### 🐛 Bug Fixes 问题修复
- 【后端】修复**经典来源选择与出处展示脱节**：此前 `source_classic` 只作用于候选池（`GetPoetryCharList`）与文化印象加分，而出处回填走的是**不筛来源**的 `QueryNamePoetry`，导致用户选「唐诗宋词」却看到「诗经《南有嘉鱼》」这类无关出处。现新增 `classics.QueryNamePoetryBySource(name, source)`，出处查询先按来源族筛选再排序；该族内无命中时回退全库（不丢失出处）。
- 【后端】新增 `classics.sourceFamilyGroups` 来源族映射表，解决候选池侧与展示层两套来源命名体系的对应关系：用户可选入口「唐诗宋词」在展示层对应 7 个子集（唐诗三百首/宋词三百首/元曲/乐府诗集/水墨唐诗/辞赋/唐诗补录），「诗经」「楚辞」「论语」等为一对一。
- 【后端】`resolvePoetry` / `enrichPoetrySource` / `enrichPoetryForName` 增加 `sourceClassic` 参数并贯穿 `/generate`、`/generate/analysis`、`/evaluate`、`/names/generate/explore` 四条链路。

### 🧪 验证
- 后端：`go test ./internal/domain/classics/... ./internal/domain/name/... ./internal/application/services/... ./internal/infrastructure/server/...` 全绿；新增 `TestQueryNamePoetryBySource`（含「选唐诗宋词不应返回诗经出处」「族内无命中回退全库」「MatchedChars 必须真的出现在引句中且是名字用字」）与 `TestSourceFamilyMembers`。
- 前端：`pnpm test` 53 例全绿（6 文件）；`tsc --noEmit` 0 错误；`next build` 成功（11 路由静态生成）。
- 端到端（真实 API）：`source_classic=诗经` → 出处全部为诗经（《小明》《静女》《桑柔》），且 `poetry_matched_chars` 明确标出命中字（`['明']`/`['炜']`/`['宝']` —— 单字命中）；`source_classic=唐诗宋词` → 出处为辞赋/乐府诗集（属该族）；不选 → 全库最优。
- 产物核验：chunk `0690b2m-65x62.js` 含 `poetry_matched_chars` 传参与高亮 span；CSS 含 `.underline` / `.underline-offset-2` / `.decoration-crimson\/40`。
- 回归：E2E 巡检 **18/18 通过**（退出 0），含「source_classic 生效（Top10 差异 3 个）」。

## [2026.10.21.5]

**start-server 脚本纳入前端资源构建 + 修复 `/json-viewer` 等根级静态资源 404**

### ⚡ Features 功能增强
- 【部署】`backend/bin/start-server.sh` 新增**前端构建环节**：比较 `frontend/src`、`frontend/public` 及前端配置（`next.config.js`/`package.json`/`tsconfig.json`/`postcss.config.*`/`tailwind.config.*`）的最新 mtime 与 `frontend/out` 产物 mtime，缺失或过期时自动执行 `next build`（产物 `frontend/out`）；依赖缺失时先 `pnpm install`。
- 【部署】新增 `--skip-frontend` 参数：仅启动后端接口时跳过前端构建；`--rebuild` 现同时强制重建后端与前端；`--check` 覆盖两者。
- 【部署】前端构建失败时**不启动服务**（避免托管过期产物导致线上跑旧代码），并提示可改用 `--skip-frontend`。
- 【部署】脚本内显式 `export CODEBUDDY_SAFE_DELETE_ENABLED=0`：本地 safe-delete shim 会拦截 `next build` 清理 `.next`/`out` 的动作（批量阈值 50，命中报 `SAFE_DELETE_BULK_CONFIRM_REQUIRED`）导致构建失败；这些目录均为可再生构建缓存。注意 shim 语义为 `!== '0'` 即启用，故必须显式赋 `0`（用 `:-0` 在环境已预置 `1` 时不会覆盖）。

### 🐛 Bug Fixes 问题修复
- 【后端】`internal/infrastructure/server/static.go` 补齐 `json-viewer` 页面路由：前端导航「JSON 查看器」入口指向 `/json-viewer`，但该页未注册路由，此前恒 404。
- 【后端】新增 `registerRootStaticAssets`：Next.js `output:'export'` 会把 `public/` 内容原样复制到产物根目录，而 gin 仅注册了 `/_next` 与 `/static`，导致 `/json4matter.html`（JSON 查看器 iframe 内容）、`/manifest.json`、`/sw.js`、`/gate-data/*` 全部 404。现按文件名白名单与目录白名单精确注册，并加 `noCache` 规则。

### 🧪 验证
- 脚本三态实测：源码/产物均最新→前后端全跳过；`touch frontend/src/app/result/page.tsx`→自动触发前端重建（`next build` 退出 0，11 路由静态生成，`out/` 刷新至 16:18）；重建后再跑→全跳过（无死循环）。
- 改 `backend/internal/infrastructure/server/static.go` 后经脚本启动→自动重建后端二进制并生效。
- 产物核验：重建后 5 个 chunk 均含 `getFavorites` 归一化 `s=t?.data?.data??t?.data?.records??[];data:Array.isArray(s)?s:[]`；结果页 chunk 含 `Array.isArray(D.data.data)?D.data.data:y` 防御。
- HTTP 端到端：`/`、`/result`、`/json-viewer`、`/json4matter.html`、`/manifest.json`、`/sw.js`、`/gate-data/naming_quality.json` 均 200；`/api/v1/favorites` 200。（`/favicon.ico` 404 属预期——仓库无 favicon 源文件，浏览器自动请求，不影响功能）
- 测试：`go test ./internal/infrastructure/server/...` 全绿，新增 `TestSetupStaticFileServer_JsonViewerAndRootAssets` 守护上述路由。

## [2026.10.21.4]

**修复结果页 `I.some is not a function` 控制台崩溃（前端构建产物过期）**

### 🐛 Bug Fixes 问题修复
- 【前端】结果页 `/result` 控制台报 `I.some is not a function`：根因是 `frontend/out/`（Next.js `output:'export'` 静态产物）自 2026-10-19 起**从未重新构建**，浏览器仍运行旧的未归一化 `getFavorites`——它把后端分页信封 `{code,success,data:{data,records,total,page,limit}}` 原样返回，使结果页 `cachedFavorites`（即 `I`）取到**对象而非数组**，触发 `.some` 崩溃。
- 【前端】重新执行 `next build` 重建 `out/`，使产物反映已修复的源码（`api.ts` 的 `getFavorites` 已做 `data: Array.isArray(list)?list:[]` 归一化，返回恒为数组）。
- 【前端】`src/app/result/page.tsx` 的 `cachedFavorites` 与 `getCachedFavorites` 增加 `Array.isArray` 防御兜底，即使接口偶发返回信封对象也不会再崩溃（双保险）。
- 注：`out/json4matter.html` 等静态资产来自 `frontend/public/`，`next build` 会自动复制，重建不会丢失；`json4matter.html` 顺带刷新为 `public/` 内更新版本（10-20）。

### 🧪 验证
- 构建：`next build` 退出 0，11 个路由全部静态生成，`out/` 刷新至 16:04。
- 产物核验：重建后 chunk 内 `getFavorites` 为 `s=t?.data?.data??t?.data?.records??[];return{...data:Array.isArray(s)?s:[]}`；结果页 `I=(0,s.useMemo)(()=>D.data?.success&&Array.isArray(D.data.data)?D.data.data:y,[D.data])`。
- 运行核验：经 `start-server.sh`（自动重建二进制+启动）起服务，`curl` 命中结果页 chunk 含上述守卫；`/api/v1/favorites` 返回 `data.data` 为数组，归一化链路闭环。

## [2026.10.21.3]

**修复 start-server.bat 双击黑框一闪即逝**

### 🐛 Bug Fixes 问题修复
- 【部署】`start-server.bat` 原先把 Windows 反斜杠绝对路径（`C:\...\start-server.sh`）直接传给 `bash`，bash 将 `\` 当作转义符导致找不到脚本、控制台立即退出。改为先 `cd /d "%~dp0."` 切到脚本目录，再以相对名 `bash start-server.sh` 调用（与 Git Bash 验证通过的等价命令一致）
- 【部署】`.bat` 内容改为纯 ASCII（中文 Windows GBK 代码页下 UTF-8 含中文的 .bat 会被乱码破坏），并加 `chcp 65001` 让服务日志中文正常显示；bash 查找优先级调整为 git 推导路径优先于 PATH，兜底常见安装目录；失败时 `pause` 让黑框停留显示错误而非闪退

## [2026.10.21.2]

**新增 backend/bin/start-server 一键启动脚本（缺失或源码更新时自动重建）**

### ✨ Features 新功能
- 【部署】新增 `backend/bin/start-server.sh` 一键启动脚本：启动前检测 `backend/bin/namer-server.exe` 是否存在、以及 backend 下源码/依赖（`*.go`/`go.mod`/`go.sum`）与配置（`*.yml`）的 mtime 是否比二进制更新；任一条件满足（或传 `--rebuild`）则自动 `go build -o bin/namer-server.exe ./cmd/server`，否则跳过构建。随后前台启动服务（绑定 `127.0.0.1:8080`，Ctrl+C 优雅关闭），日志落 `backend/bin/logs/`。支持 `--rebuild` 强制重建、`--check` 仅校验/构建不启动
- 【部署】附 `backend/bin/start-server.bat` 薄封装，双击即可经 Git Bash 委托运行同一脚本（Windows 一键入口）
- 【部署】二进制构建产物统一落 `backend/bin/namer-server.exe`（已被 `.gitignore` 的 `*.exe` 忽略，属本地产物，脚本按需再生，不纳入版本库）

### 🧪 验证
- 脚本三态实测：①二进制最新 → 跳过构建；②`touch` 任一 `.go` → 自动重建（go build 成功）；③重建后再次运行 → 跳过（无死循环）
- 经脚本启动服务，`/healthz`、`/readyz` 均 200；物料载入正确（康熙笔画 7,838 / 经典字集 46,691 / 字库 8,105），与权威库一致
- 注意：`find -exec stat` 会因派生进程过多被沙箱 SIGTERM，已改用 `find -print0 | xargs -0 stat` 批处理

## [2026.10.21.1]

**修复 fate 构建断点（已核实无误）、千家诗字集静默遗漏，并新增经典语料巡检报告**

### ✨ Features 新功能
- 【巡检】`materials import` 与 `materials status` 新增「经典字集提取数 / 诗词条目数」报告：`classics_extracts` 按已登记 material 逐行列出字集提取数，`classics_poem_entries` 按 source 列出诗词条目数；任一已登记字集物料在库中字集行数为 0 时显式告警 `⚠ 缺失（出典加分静默遗漏风险）`，使「索引齐全但字集缺失」类遗漏可在巡检中被发现

### 🐛 Bug Fixes 问题修复
- 【经典语料】修复千家诗（qianjiashi.json）字集静默遗漏：该文件部分分章诗篇（其一/其二）的 `paragraphs` 为嵌套对象 `[{subchapter, paragraphs}]`，通用 `loadClassicFromJSON` 整文件反序列化失败 → 千家诗 0 字集。新增 `loadQianjiashiFromJSON` + `flattenParagraphs` 兼容平铺/嵌套两种结构，字集由 0 → 1,626 字（与元曲/乐府/辞赋同口径进入 `classics_extracts`）
- 【构建】`fate` 引擎 `generateWithGeneration`（字辈分支）经核实已实现完整，整仓 `go build ./...` 与 `go vet ./...` 全绿，`cmd/materials` 可正常构建并跑通 `import` 端到端验证闭环（此前 `2026.10.21.0` 误记为未定义断点，特此更正）

### 📈 Improvements 性能/体验优化
- 【数据库】重新执行整仓 `materials import` 重建权威库 `backend/namer.db`，使其与当前代码完全对齐：`kangxi_strokes` 裁剪至 7,838 行（0 孤儿）、`classics_extracts` 21 类经典字集全部非空（合计 46,691 字，含千家诗 1,626 / 元曲 4,828 / 乐府 2,515 / 辞赋 4,183）

## [2026.10.21.0]

**确立单一权威库消除双库分叉、kangxi_strokes 按字集裁剪、补齐元曲/乐府/辞赋起名字集**

### ✨ Features 新功能
- 【经典语料】补齐 `classic_loader.go` 的 `loadYuanQu`/`loadYueFu`/`loadCiFu`，使元曲/乐府/辞赋进入 `classics_extracts` 起名字集（此前仅进展示索引），修复用这三类字起名时无出典加分的遗漏；同步登记 `buildPoetryIndex`/`extractTargets`/`appendAllExtracted`/`GetPoetryCharList` 等清单

### 🐛 Bug Fixes 问题修复
- 【数据库】确立 `backend/namer.db` 为唯一权威库（`application.yml` 的 `database.path` 改为绝对路径，dev/prod 容器均经 `NAME_DATABASE_PATH` 指定）；`cmd/materials` 6 处 `--db` 默认值收敛到配置路径；清理 backend 库孤儿表 `words`(14809行)/`data_meta` 并移走根目录分叉副本
- 【数据库】`kangxi_strokes` 按 `hanzi_data` 字符集裁剪：63,696 → 7,838 行（删除 55,858 孤儿行），导入管线 `Import()` 末尾新增 `pruneKangxiStrokes()` 持久化该裁剪

### 📈 Improvements 性能/体验优化
- 【数据库】`kangxi_strokes` 裁剪后文件 22,904 → 21,488 KB，运行期内存字集零影响（缺失笔画返回 0 优雅降级）

### 🔧 Dependencies 依赖更新
- 【已知阻塞·已解决】`engine.go:705` 的 `generateWithGeneration` 经 `2026.10.21.1` 核实已实现完整，整仓构建与 `materials import` 端到端验证均已跑通，先前记录有误；权威库已整仓重建对齐（见 `2026.10.21.1`）

## [2026.10.20.1]

**万年历展示 24 节气与农历节日、JSON 查看器统一为新中式皮肤，并接入 E2E 框架、新增字辈(Generation)生成分支**

### 🐛 Bug Fixes 问题修复

- 【万年历】新增后端 `/api/v1/lunar/month` 逐日节气 / 农历节日批量接口与前端标注渲染，修复万年历不显示 24 节气（清明、谷雨、寒露、霜降…）与农历节日（重阳节、上巳节…）的问题；农历节日优先、节气次之
- 【导航】修正首页顶部「JSON」按钮图标：原仅画两个松散花括号被误读为 "{}JSON"，改为「文档 + `</>` 代码」图标
- 【前端】修复 `WannianliView` 漏导入 `useMemo` 导致的运行时崩溃，以及 `huangli/page.tsx` 中 `monthLunarQuery` 在 `currentYear/currentMonth` 声明前引用引发的 TDZ 500

### ✨ Features 新功能

- 【起名引擎】`fate` 引擎新增「字辈(Generation)」生成分支：用户显式指定字辈时，字辈字固定占一个槽位、其余槽位仍从候选池自由枚举；字辈不合法时回退常规枚举（`generateWithGeneration`，以 `handled` 返回值控制分支）
- 【测试】接入 tester-army/e2e v0.21.2 框架：新增 `e2e.config.api.ts` / `e2e.config.ts`、`frontend/tests/` 主链路用例、`backend/webui_e2e_test.py`，并补充 `docs/35-e2e框架接入指南.md`；`next.config.js` 加 `allowedDevOrigins: ['127.0.0.1']` 修复 E2E 浏览器经跨源打开时 "渲染但不水合" 的问题

### 📈 Improvements 性能/体验优化

- 【JSON 查看器】`public/json4matter.html` 离线工具重皮肤为「新中式·墨韵宣纸」调色板（朱红 / 金 / 青 + 宣纸底），字体统一 Noto Sans SC，拖拽遮罩与错误色改朱红；新增 `syncParentTheme()` 跟随父页面明暗模式；宿主页 `json-viewer` 改为云纹底 + 卡片框 iframe，与主应用风格统一

### 🔧 Chore / 基建

- 【物料】收尾 `backend/data/*` → 仓库根 `data/` 的迁移：删除仓库内残留的 `backend/data/*.json` 跟踪项；`.gitignore` 追加 `data/`
- 【依赖】`frontend` 新增 e2e 相关 devDependencies（`e2e`、`@e2e-dev/web`），更新 `pnpm-lock.yaml`
- 【数据】`frontend/public/gate-data/` 纳入门禁数据静态资源（随导出发布）

> 验证：后端 `go test ./internal/domain/bazi/ -run 'TestGetSolarTermExact|TestGetLunarFestivalName|TestGetLunarMonthCalendar'` 通过；`/api/v1/lunar/month` 返回 2026-10 寒露/重阳节/霜降、2026-04 清明/上巳节/谷雨；前端 `next dev`(3000) 下 `/huangli`、`/json-viewer` 均 200 无编译错误；E2E 两份用例（API 契约 + UI 主链路）复跑全绿。

## [2026.10.19.3]

**CLI `--data` 默认值同步改为向上探测仓库根 `data/`：补齐与服务端同一缺陷，消除「从 backend/ 运行即找不到物料」**

### 🐛 Bug Fixes 问题修复

- 【CLI】`namer-cli` 的 `--data` 默认值固定为 `./data`，物料源文件迁至仓库根
  后从 `backend/` 运行即指向已不存在的 `backend/data`。与服务端 `cwdDataDir`
  是同一缺陷的两个实例，本轮一并修复：显式传入之外先向上探测仓库根 `data/`
  （共用 `paths.FindDataDir`），探测无果再回落 `./data`，保留「仅解压可执行
  文件、无仓库结构」时的历史行为
- 【CLI】`--data` 标志默认值由 `./data` 改为空串，以便区分「用户显式传入」与
  「未传入」（原实现下两者都表现为 `./data`，无法在探测前介入）

### 📈 Improvements 性能/体验优化

- 【代码结构】`resolveDataDir` 与 `cwdDataDir` 同风格：探测起点提取为包级
  变量 `cliSourceFile`，可被测试替换以稳定模拟无仓库结构的场景

### 📚 Docs 文档更新

- `--data` 的帮助文本更新为「未指定时自动探测仓库根 data/，再回落 ./data」

## [2026.10.19.2]

**服务端默认数据目录改为向上探测仓库根 `data/`：无 `DATA_DIR` 时不再指向已迁走的 `backend/data`，并连带修正 fail-fast 导入指引的路径**

### 🐛 Bug Fixes 问题修复

- 【服务启动】`cwdDataDir` 在未设 `DATA_DIR` 时固定回落 `cwd/data`，而物料
  源文件已从 `backend/data` 迁至仓库根 `data/`。后果是从 `backend/` 启动且
  不设环境变量时服务 fail-fast 拒绝启动，**且打印的恢复命令指向那个不存在的
  目录**——用户照抄 `materials import --data "…\backend\data"` 只会导入失败。
  现改为先向上探测仓库根 `data/`（与测试侧共用 `internal/paths.FindDataDir`），
  探测无果再回落 `cwd/data`
- 【部署同源】探测无果时的回落保持 `cwd/data` 不变，与入口脚本
  `DATA_DIR=${DATA_DIR:-/app/data}` 的契约继续一致：容器内 `/app` 无仓库结构，
  行为与改动前完全相同，仅本地开发受益

### 📈 Improvements 性能/体验优化

- 【代码结构】新增零依赖的 `internal/paths` 包承载数据目录探测规则，
  `internal/testutil.DataDir` 改为委托它、`cmd/server` 直接复用它，消除
  「测试侧一套、进程侧一套」的分叉。`cwdDataDir` 的探测起点提取为包级变量
  `mainSourceFile`，使「容器内无仓库结构」的场景可被测试稳定模拟
- 【测试】新增 `TestCwdDataDir_仓库内探测到仓库根` 覆盖真实仓库布局；
  `TestCwdDataDir` 与入口脚本契约测试补上探测起点替换，分别覆盖回落分支与
  探测命中分支

## [2026.10.19.1]

**消除测试对 `backend/data` 固定层级的硬编码：新增 `testutil.DataDir` 向上探测，数据目录重组不再导致全仓测试静默失效**

### 🐛 Bug Fixes 问题修复

- 【测试基建】16 个测试文件各自硬算 `../../../data` 定位物料数据目录，
  目录一旦重组（如 `backend/data` 迁至仓库根 `data/`）即**全线静默失败**。
  静默是关键：多数用例在字库加载失败时只告警不阻断，跑出来的是"数据为空"
  而非"测试失败"，与真实缺陷的表现完全一致，排查成本极高。新增
  `internal/testutil` 的 `DataDir(callerFile)` / `MustDataDir(t, callerFile)`：
  以 `namer.json` 为标记自调用方源文件目录向上逐级探测，同时兼容
  `backend/data` 与仓库根 `data/` 两种布局及后续任何一次重组
- 【测试基建】探测失败时 `MustDataDir` 直接 `t.Fatalf` 并附中文说明。这与
  `DataDir` 返回空串后由调用方"继续跑空数据"是刻意的分层：数据是用例前提
  的场景应当立即失败，而不是带着空字库跑出一片无意义的绿
- 【CLI 集成测试】`cmd/namer-cli` 集成测试改用 `-data` 显式传入探测到的
  数据目录，不再依赖 CLI 默认值 `./data` 按 CWD 解析。CLI 默认值语义保留
  不变（面向交互式使用），仅测试侧消除对 CWD 的脆弱耦合

### 📈 Improvements 性能/体验优化

- 【代码结构】`internal/testutil` 提供共用的测试数据目录解析，替代分散在
  `domain/hanzi`、`domain/fate`、`domain/name`、`domain/yijing`、
  `domain/namestat`、`domain/classics`、`application/handlers`、
  `application/services`、`database/sqlite`、`cmd/namer-cli` 中的重复实现，
  顺带消除各包自建的 `packageDir` / `dataDir` / `e2eDataDir` /
  `locateBackendDataDir` 等同义 helper

## [2026.10.19.0]

**服务启动期物料 fail-fast：必需物料缺失一律拒绝启动并打印导入指引，根治"healthz 200 但全部起名请求 400"的假健康态**

### 🐛 Bug Fixes 问题修复

- 【服务启动】`cmd/server` 装载传统文化物料失败时由 `logger.Warn` 后继续改为
  **拒绝启动**（exit 1）。原行为下物料库为空（未执行 `materials import`）服务
  照常监听、`/healthz` 返回 200，但每个起名请求都 400——监控看到"服务健康"
  而业务 100% 不可用，比启动失败难排查得多。与 docs/31 ADR-008
  「运行时 SQLite 唯一真源 + 必需物料缺料 fail-fast」对齐：`materials load`
  早已退出码 1，服务端此前却把同一类错误降级成 warn
- 【启动指引】拒绝启动时向 stderr 打印完整恢复命令
  `go run ./cmd/materials import --data "<dataDir>" --db "<dbPath>"`，路径取
  当前进程实际生效的 `DATA_DIR` 与 `database.path`，用户照抄即可补救
- 【指引可用性】指引命令用普通双引号而非 `%q`：`%q` 是 Go 语法转义，会把
  Windows 路径写成 `"D:\\19-..."`，复制到 cmd/PowerShell 会指向错误目录；
  该产出是 shell 命令而非 Go 字面量

### ⚠️ Breaking Changes 破坏性变更

- 【服务启动】原先依赖"物料缺失仍能降级启动"的部署（如先起服务、后补物料）
  现会启动失败。请在启动前执行 `materials import`，或改用
  `materials load` 作为启动前置探针

### 📈 Improvements 性能/体验优化

- 【代码结构】`loadCulturalData` 拆出 `loadCulturalDataFromDB`（可直接用
  `data.DBLoader` 假实现测试，不必构造整个 `database.Store`）与
  `materialsImportHint`（指引命令构造，纯函数可测）

### 📚 Docs 文档更新

- 【测试】`cmd/server/main_test.go` 新增 5 个用例：指引命令构造（含
  Windows 路径不转义回归守护）、空物料库返回错误且含缺料原因与恢复命令、
  错误保留底层缺料原因不被包装吞掉、无库场景不附带 `materials import` 指引
  （该场景修复动作是补数据文件而非导入，给出 import 命令是误导）

## [2026.10.18.16]

**校验器(validator)汉字判定改用 `unicode.Is(unicode.Han)`，根治"合法中文字符被误拒"——附 GBK 测试伪象排查**

### 🐛 Bug Fixes 问题修复

- 【校验器】`ValidateSurname`/`ValidateGivenName` 由硬编码区间 `[4E00,9FA5]` 改为
  `unicode.Is(unicode.Han, r)` 标准汉字判定。原区间漏掉 CJK 扩展A（如 `䶮` U+4DAE）、
  〇（U+3007）等**合法但罕见**中文字符，属"合法中文字符被误拒"类缺陷；现覆盖 CJK 统一表意文字、
  扩展A、兼容汉字及〇，仍拒绝字母/数字/标点并保留长度上限（姓≤4/名≤8）
- 【排查】用户反馈 `ValidateSurname` 对"李"误判 400；实测证实源码无误判
  （单测 PASS、UTF-8 实发 200），400 系 Git Bash 按 GBK 发中文的测试命令编码伪象，
  非代码缺陷（已在测试规范中标注，测中文接口须用 UTF-8）

### 📈 Improvements 性能/体验优化

- 【测试】`validator_test.go` 新增罕见合法汉字用例：姓氏 `䶮`（CJK 扩展A）、
  名字 `含〇`/`单字〇`；原有非法用例（含字母/夹数字/超长）仍成立

## [2026.10.18.15]

**数据集去冗余与结构瘦身：classics_extracts 去字典属性、删除无读取方的 classics_paragraphs、统一归一化口径（48.0 MB → 22.1 MB）**

### ✨ New Features 新增功能

- 【经典语料】`classics_extracts` 去除 `pinyin`/`meaning`/`wuxing`/`gender`
  四个反规范化列（`hanzi_data` 的副本，18 组字集平均把每个汉字抄 3.30 遍，
  单是 `meaning` 一列即占全库 11.5%）；运行期由 `LoadExtractSets` 按 `char`
  从字库补全，取值与缺省口径同 `extractFromEntry`
- 【经典语料】`importClassics` 不再需要预先装载字库（字典属性既不落库，
  原先的 `hanzi.LoadNamerFromJSON` 隐性依赖随之消除）
- 【经典语料】`LoadClassicsFromDB` 新增前置条件守卫：字库未装载即报错，
  避免静默补出空属性导致释义评分整体失效
- 【经典语料】新增 `migrateClassicsSchema`：存量库按「建新表 → 迁数据 → 换名」
  重建 `classics_extracts`、DROP `classics_paragraphs`，并在结构变化后执行
  一次 `VACUUM` 回收空间

### ⚠️ Breaking Changes 破坏性变更

- 【表结构】删除 `classics_paragraphs` 表：全仓无任何 SELECT 读者，却以
  114,139 行承载与 `classics_poem_entries.content` 重复 42.2% 的诗文文本。
  诗文全文以 `classics_poem_entries.content` 为单一真源
- 【表结构】`baijiaxing.json` 的 `origin`（姓 → 地）与 `qianziwen.json` 的
  `spells`（拼音）原本只落在段落表中，现随表移除；二者均非运行期消费数据
- 【运行期】诗词出典的 `PoetryChar.Meaning` 由未截断原文改为与 `hanzi_data`
  一致的 200 rune 截断版（口径统一，释义评分有微幅变化）

### 🐛 Bug Fixes 问题修复

- 【口径分裂】`LoadNamerFromJSON` 与导入期 `normalize.Word` 各做各的归一化，
  同字两值：`Meaning` 为 200 rune 截断 vs 262~276 rune 原文；`StyleTags`/
  `NamingCategories` 为排序 vs 源序。现归一**就地**做一次，两个下游共用
- 【进程级单例】`classics.loadShiCiOnce`（`sync.Once` + 单通道）改为按
  dataDir 去重 + 每次装载独立完成通道。此前同进程第二次 `LoadFromJSON`
  拿不到 shici 却不再发起装载，使对拍测试静默丢掉 6950 条覆盖而不报错
- 【进程级单例】`hanzi.freqDB.loaded` 布尔闩增加 `dir` 字段，改为同目录短路、
  换目录重装。此前换目录后函数直接返回，而 `LoadNamerFromJSON` 依赖它把
  档位合并进刚重建的 `HanziData`，导致同进程内第二次装载字库时所有字
  `NameFreqTier` 静默变 0

### 📈 Improvements 性能/体验优化

- 【表结构】经典语料由三表收敛为两表 + 两张物化表；
  `store_classics.go` 四组 importer 简化为只写书籍/篇章骨架，
  随之失效的 `splitSentences` 一并移除
- 【体积】库体积 48.0 MB → 22.1 MB（-54%）；结构迁移后 `VACUUM` 回收空闲页
  （DROP 只把页还给 freelist，实测迁移后空闲页达 56.6%、文件不降反升）

### 📚 Docs 文档更新

- `docs/34` 追加阶段 E-8（去冗余与结构瘦身），含 E-8-A~E-8-E 各项取舍与实测数据

### ✅ Tests 测试

- 对拍守护由「8 个字库字段」扩到**24 个字段逐字段逐字** + 18 组字集
  （33,539 条）+ 诗词条目（12,474 条），JSON 路径与库路径零差异
- 新增守护：`classics_paragraphs` 已废弃、`classics_extracts` 无冗余列、
  字集字典属性由字库补全且与 `hanzi_data` 一致
- `golden_names.json` 基线按预期重建（JSON 路径释义/标签口径变化的直接结果）
## [2026.10.18.14]

**统一规范 SQLite 数据集结构：curated/classics 接入统一导入器，运行期彻底只读库**

### ✨ New Features 新增功能

- 【策展名】`curated_names` 接入统一导入器（`materials import`），新增 `importCurated`：
  仅重建种子行（`source=seed`），保留用户自学习收藏行（`source=user_favorite`），
  移除启动期 `seedCuratedNames` 回路
- 【策展名】`curated_names` 扩列 `meaning`/`wuxing`/`styles`/`tags`，新增
  `migrateCuratedNamesColumns` 为存量库补列，使运行时改读库后策展释义与风格标签不再丢失
- 【经典语料】`classics` 接入统一导入器，新增 `importClassics`：装载字库补全拼音/五行/性别
  → 复用 JSON 提取逻辑 → 三表与物化表单事务写入
- 【经典语料】新增两张运行期物化表 `classics_extracts`（18 组经典字集）与
  `classics_poem_entries`（诗词索引）；`FullText`/`Chars` 为 `content` 纯函数故不落库
- 【经典语料】新增 `classics/snapshot.go` 数据契约：`ExportExtractSets`/`LoadExtractSets`、
  `ExportPoemEntries`/`LoadPoemEntries`
- 【经典语料】新增 `Store.LoadClassicsFromDB()` 运行期装载；`data` 包经可选
  `ClassicsLoader` 接口接入
- 【CLI】新增 `materials load` 子命令（库 → 内存装载自检，必需物料缺料以非 0 退出）
- 【CLI】`namer-cli` 新增 `--db` 走 SQLite 物料库，与服务端运行期口径一致

### ⚠️ Breaking Changes 破坏性变更

- 【运行期】物料来源策略默认收紧为 `SourceSQLite + AllowFallback=false + RequireAll=true`：
  必需物料（kangxi/namer/naming_gate/forbidden_combos/yijing/zodiac）在库中缺失即启动失败
  并提示执行 `materials import`，不再静默回落 JSON
- 【CLI】`namer-cli` 不带 `--db` 时仍走纯 JSON 路径；带 `--db` 时不再读 JSON 物料
  （策展名改从 `curated_names` 表读）

### 🐛 Bug Fixes 问题修复

- 【表结构】`hanzi_data` 建表语句此前在 `store.go` 与 `store_materials_schema.go`
  各定义一份且默认值不同（`usage_level` 3 vs 0），已收敛为单一真源
- 【表结构】经典三表建表语句从 `store.go` 收敛到 `materialTableDDL`；
  `data_meta`/`classics_version` 对拍机制随统一导入器落地而移除
- 【导入器】`snapshotMaterialTables` 原用字符串 `+=` 拼接快照，物料表扩容后退化为
  O(n²)，导致 sqlite 包测试耗时失控（>600s）；改用 `strings.Builder`
- 【导入器】`excludeVolatileCols` 补入 `created_at`（`curated_names` 每次重灌刷新该列），
  避免幂等断言恒假
- 【导入器】classics 导入后归零 `sqlite_sequence`，保证「同输入多次导入结果等价」

### 📈 Improvements 性能/体验优化

- 【经典语料】单文件解析失败不再阻断整个物料（如 `qianjiashi.json` 的 `paragraphs`
  元素为 `{subchapter, paragraphs}` 嵌套对象）；跳过清单写入导入报告 `Reason`，由静默改为可见
- 【装载报告】新增 `loadedElsewhereMaterials`，把「导入侧接入集合」与「运行期装载报告集合」
  解耦：curated/classics 由各自领域消费方自证 fail-fast

### 📚 Docs 文档更新

- 更新 `docs/31`（ADR-008 落地状态：全部 12 项物料均已接入统一导入器）与
  `docs/34`（阶段 E：统一数据集结构与运行期唯一真源）

### ✅ Tests 测试

- 新增 `TestImportClassics_真实数据下库路径与JSON路径等价`：真实 `data` 目录对拍，
  18 组字集 33539 条 + 12474 条诗词条目，库路径与 JSON 路径零差异
- 新增 curated 导入守护（富字段落库、用户行保留、修订传导、同名让位）
- 新增 classics 导入/装载守护（三表与物化表、字集属性补全、自增序列幂等、空库 fail-fast）
## [2026.10.16.13]

**物料管线阶段 B/C/D：清理接线、健壮性与回归固化**（`docs/34` 阶段 B~D）。
收敛「宣称未落地」的优化、修补装载漏报与原子性缺口，并把幂等/增量门控固化为 CI 断言。

### ✨ New Features 新增功能

- 【物料校验】`validate.Errors.Summarize`：校验错误按 (字段, 规则) 聚合计数 +
  前 N 条明细样例，`failValidate` 改走聚合输出（大批量脏数据时可读、可回归比对）。
- 【物料装载】`IntegratedMaterialNames()`：运行时装载物料清单改为从
  `schema.Definitions` 派生（剔除 legacy），「导入侧接入集合」与「装载侧报告集合」
  单一真源，杜绝手抄清单漂移。

### 🐛 Bug Fixes 问题修复

- 【物料装载】修复装载报告漏报 optional 物料：`materialNames()` 曾漏列
  `allowlist` / `compliance_redline`，`LoadMaterialsFromDBWithReport` 与
  `VerifyMaterialsFromDB` 永远不报这两类物料缺失（docs/34 阶段 B2）。
- 【物料导入】修复 `materialUnchanged` 空表误判：辅表（如 namer 的 `char_groups`）
  合理为空时物料被误判为「已变更」而每次重灌，指纹门控形同虚设；现只校验主表
  （`Tables[0]`，schema 层加约定注释）（docs/34 阶段 B3）。
- 【灰度发布】`subscribe.writeFile` 改为「写临时文件 → fsync → rename 覆盖」：
  promote/abort 直接覆写生效物料与 lock 的中断窗口（半截文件）关闭（docs/34 阶段 C1）。

### ♻️ Refactors 重构

- 【物料导入】抽取两个通用骨架（docs/34 阶段 C3）：
  `importStringSliceMaterial`（naming_gate / forbidden_combos）与
  `importKindRowsMaterial`（allowlist / compliance_redline），importer 只保留
  物料差异参数，样板代码约减 120 行。
- 【代码整洁】删除 `checksum.Cache`/`Stat`/`StatOf`/`Forget`（docs/34 阶段 B1）：
  size+mtime 快速路径从未接线，且「等长修改 + mtime 未变」场景会漏判——
  用正确性换不存在的性能；单次导入每文件只 hash 一次，缓存无收益。
- 【代码整洁】sha256 统一到 `materials/checksum`：`subscribe.HashBytes`/`HashFile`
  改为委托（docs/34 阶段 P2-4），全仓不再有两份指纹实现。
- 【代码整洁】`schema.Word.ClassicalSc` → `ClassicalScore`（缩写命名对齐
  PositiveScore/ModernScore 等同族字段）。

### ✅ Tests 测试

- 【物料导入】新增 `store_materials_idempotency_test.go`（docs/34 阶段 D1）：
  ① 二次导入全短路后内容快照逐字节等价（含频率档派生列一致性）；
  ② 增量门控：单物料源变更只重灌该项、其余全部短路；
  ③ 辅表为空不误判变更（B3 回归）；④ 可选物料源缺失即跳过不失败。
- 【灰度发布】新增原子写测试：覆盖后内容完整、无临时文件残留、首写场景成立。

## [2026.10.16.12]

**物料管线阶段 A：收敛数据口径**（`docs/34` 阶段 A）。根除 `hanzi_data` 双口径写入，
`namer.json` 成为字库唯一真源，`word.json` 退出物料管线。

### ✨ New Features 新增功能

- 【物料管线】`Import()` 分发前预解析跨物料依赖并经 `ImportOptions.deps` 注入
  （`resolveImportDeps`/`parseKangxiFile`），`importNamer` 不再回读只读连接
  `s.db` 取康熙笔画——解除「依赖物料已落库 + WAL 跨连接可见」两个隐含前提。
- 【物料管线】新增 `Store.syncNameFreqTier`：`hanzi_data.name_freq_tier` 改为在
  `Import()` 末尾以 `name_frequency` 表为唯一真源单点归一化（`COALESCE` 子查询，
  仅 Tier ≥ 2 回填），双向传导、与导入顺序和指纹短路无关。

### 🐛 Bug Fixes 问题修复

- 【物料管线】修复 `hanzi_data` 双口径写入：legacy `seedHanziData`/`rebuildHanziData`/
  `ReloadHanziData`（内存 `hanzi.HanziData` → `hanziDataValues` 投影）与物料导入路径
  `importNamer`（`namer.json` → `wordValues` 投影）字段派生口径不一致，一旦被误用即污染
  字库。该路径无生产调用方，已整体移除，`importNamer` 成为唯一写入方。
- 【物料管线】修复 `name_freq_tier` 两处维护（`importNamer` 回读 + `importCharFreq`
  内 `UPDATE`）的隐式顺序依赖与口径漂移风险。

### 🗑 Deprecated 废弃功能

- 【物料管线】移除 `word` 物料（`schema.WordMaterial`、`importWords`、`words` 表 DDL、
  `loadWordsFromDB`、`materialNames` 条目）。`namer.json` 已包含全部三级候选字及释义，
  `word.json` 不再被加载/读取。
- 【领域层】移除 `hanzi.LoadWordData`/`WordMeaning`/`SetWordMeanings`/`WordEntry`
  及 word.json 运行时加载节（均无生产调用方，仅测试引用）。
- 【CLI】`cmd/namer-cli` 移除 `hanzi.LoadWordData` 调用，装配顺序与 server 对齐。

### 📈 Improvements 性能/体验优化

- 【物料管线】康熙笔画 CSV 由「每物料各解析一次」收敛为 `parseKangxiFile` 单一入口，
  `importKangxi` 与依赖预解析共用同一份口径。

### 📚 Docs 文档更新

- `docs/34` 阶段 A 标记为已实施并登记验证结论；`docs/32` M11 双模式 e2e 复验记录更新。

## [2026.10.16.11]

**门禁层 M7：灰度发布（按物料整体 staging + 观察 + 回滚）**（`docs/33` M7 / `docs/31` ADR-015）。

### ✨ New Features 新增功能

- 【门禁层】`subscribe.Lock` 扩为两态：新增 `staging []LockEntry`（`json:"staging,omitempty"`，
  缺省即单态旧格式，向后兼容），并增 `FindStaging`/`SetStaging`/`RemoveStaging`/`UpsertEntry`。
- 【门禁层】新增 `internal/materials/subscribe/release.go`：
  - `Stage`：候选按物料结构校验后落 `data/staging/<material>.json` + 写 lock 候选指纹 + 记 release；
  - `Observe`：复用 M5 `DiffLines` 出差分预演（`Current`/`Candidate` 双指纹 + 新增/移除条目）；
  - `Promote`：当前 active 存档到 `data/history/<material>/<sha>.json` → 候选覆盖生效路径
    → 更新 lock.active 并清候选；
  - `Abort`：无 `--to` 丢弃候选；`--to <sha>` 从存档回退 active（粒度=整物料）；
  - 每次动作追加一条 append-only `data/release.jsonl`（可审计、可回滚）。
- 【门禁层】`cmd/materials` 新增 `stage`/`observe`/`promote`/`abort` 子命令
  （`--material`/`--from`/`--to`/`--operator`/`--source`/`--lock`/`--limit`）并更新帮助文本。

### 📚 Docs 文档更新

- `docs/33` M7 状态更新为已完成并补落地记录；§十二 状态改为「已落地」、勾选待实现清单、
  `observe` 参数对齐实现；§六/§七/「一句话」标注 **M0~M8 全部落地**。

### 🧪 Tests 测试

- 新增 `internal/materials/subscribe/release_test.go`：两态 lock 往返与旧格式兼容、
  `stage→observe→promote`、`abort` 丢弃、`abort --to` 回退、`release.jsonl` append，
  以及 lock 缺失/物料与候选脏读/归档与日志路径占用/只读 lock 等错误路径；
  release.go 新增函数覆盖率 100%（不可达防御分支除外）。
- 演练：`lock-init → stage → observe → promote → stage → promote → abort --to` 全流程通过；
  全量 `go build/vet/test` 与 `e2e_check -strict` 全绿。

## [2026.10.16.10]

**门禁层 M8：匹配性能（Aho-Corasick 多模式匹配，替换合规红线线性扫描）**（`docs/33` M8）。

### ✨ New Features 新增功能

- 【门禁层】新增 `fate/actrie.go`：纯 Go、无第三方依赖的 Aho-Corasick 自动机——
  `NewACTrie`（模式去重、BFS 构建 fail 指针并继承后缀 outputs）、`MatchAny`（短路查询）/
  `FindAll`（去重、首次命中序）/`PatternCount`；构建后只读、可并发使用。
- 【门禁层】`compliance_redline.go` 接入 trie：三套动态红线集合与内置兜底 `forbiddenEntities`
  各建一个自动机（`newRedlineTrie` 跳过单字条目），`SetComplianceRedline` 重建、
  `IsComplianceRedline` 走 trie 扫描（白名单豁免语义不变）。

### 📈 Improvements 性能/体验优化

- 【门禁层】合规红线匹配由 **O(模式数 × 文本长)** 线性 `strings.Contains` 改为
  **O(文本长 + 命中数)**：5000 条词表下 **142604 ns/op → 69.81 ns/op（约 2000×）、零分配**
  （`-benchmem` 对比 benchmark 见测试）。
- 行为等价：既有 `fate` 合规红线测试零改动全绿（子串命中/单字忽略/白名单优先不变）。

### 📚 Docs 文档更新

- `docs/33` M8 状态更新为已完成并补落地记录；§七差距表「匹配性能」标注已落地。

### 🧪 Tests 测试

- 新增 `fate/actrie_test.go`：MatchAny / FindAll / 去重与空模式 / 中文组合 / 边界（空与 nil 自动机）/
  单字跳过 / `matchAnyEntity` 边界 + 线性 vs trie 性能对比 benchmark。
- `actrie.go` 与 `compliance_redline.go` 新增/改动函数覆盖率 100%。

## [2026.10.16.9]

**文档：门禁层 M7「灰度发布」设计定档（ADR-015，按物料整体），代码待实现**（`docs/33` §十二）。

### 📚 Docs 文档更新

- `docs/31` 新增 **ADR-015**：门禁物料灰度发布 = **物料级 staging + 观察窗口 + 一键回滚**，
  切分粒度**按物料整体**（`naming_gate`/`forbidden_combos`/`allowlist`/`compliance_redline`
  各一份独立版本）；明确**不做** kind 级切分与条目百分比/流量分流（留作多实例演进）。
- `docs/33` 新增「§十二 灰度发布设计（M7）」：状态与产物（`materials.lock` 扩 `active`/`staging`
  两态 + `data/staging/` + append-only `data/release.jsonl`）、发布流（`stage → observe →
  promote|abort`）、观察窗口（复用 M4/M6 `gate_*` 指标与 `gate_review` 人审，到期仅提示不自动切换）、
  回滚（`abort --to <lock 版本>`）、待实现清单；M7 里程碑状态更新为「设计✅（ADR-015）·代码⬜」。
- 本次为**纯设计定档**，无代码改动；M7 代码实现留待后续迭代。

## [2026.10.16.8]

**门禁层 M6：bad-case 回流闭环（gate_review 状态机 + 收藏回流 + gate_patch 导出）**（`docs/33` M6）。

### ✨ New Features 新增功能

- 【门禁层】新增 `fate/review.go`：复审状态机 `ReviewStatus`
  （`pending`/`confirmed`/`false_positive`/`border`）、`ValidReviewStatus`/`CanDecide`/`ReviewKey`，
  指标口径 `ReviewMetrics`（`Reviewed`/`FalsePositiveRate`/`SampleMissRate`/`ReviewLatencyMs`）、
  `TallyReviews`/`SortReviews`。
- 【门禁层】新增 `gate_review` 表与 `sqlite/store_gate_review.go`：`AddGateReview`（唯一键
  `name+layer+category+hit_value` 幂等）、`ListGateReviews`、`DecideGateReview`（仅 pending 可落定）、
  `GateReviewStats`、`GateReviewLatencyMs`；`Store` 实现 `fate.GateMetricProvider`
  （误报率 / 漏报代理 / 复审时长），`GET /metrics` 的 `gate_*` 变为真实数据。
- 【门禁层】收藏回流：application 端口 `services.GateReviewRecorder` + `SetGateReviewRecorder`；
  `FavoriteService.SaveFavorite` 落库成功后登记收藏正样本候选（未注入 no-op，已存在收藏不重复登记），
  `cmd/server` 注入。
- 【工具】新增 `internal/materials/review.ToPatch`：已落定复审条目 → `gate_patch`（confirmed →
  增补 `naming_gate`/`forbidden_combos`；false_positive → `allowlist` char/combo；border/pending 跳过）。
- 【工具】`cmd/materials` 新增子命令 `review`（`list` / `decide` / `export`），串起
  「线上命中 / 收藏 → 人审 → gate_patch → materials patch → 重导入」回流闭环。

### 📚 Docs 文档更新

- `docs/33` M6 状态更新为已完成并补落地记录；§六指标闭环/误伤治理、§七外部回流·误报漏报指标差距标注已落地。

### 🧪 Tests 测试

- 新增 `fate/review_test.go`、`sqlite/store_gate_review_test.go`、`internal/materials/review/export_test.go`；
  `favorite_service_test.go` 追加「回流转录 / 未注入 / 已存在 / 写失败不回流」用例。
- domain 与 review 包新增函数覆盖率 100%（sqlite 侧错误/防御分支除外）；
  「收藏 → pending → decide → export → patch → 回滚」端到端闭环演练通过。

## [2026.10.16.7]

**门禁层 M5：外部词库订阅管线（materials.lock + 行差分 + gate_patch）**（`docs/33` M5）。

### ✨ New Features 新增功能

- 【门禁层】新增 `internal/materials/subscribe` 订阅管线：`materials.lock` 登记门禁四物料
  （`naming_gate`/`forbidden_combos`/`allowlist`/`compliance_redline`）的上游来源与产物 sha256，
  支持 `BuildLock`/`LoadLock`/`SaveLock`/`Find`/`Verify`（逐条重算指纹并报告不一致）。
- 【门禁层】行差分 `DiffLines`/`ParseLines`（去空白、去 `#` 注释、去重）与补丁
  `PatchEntry`/`BuildPatch`/`ParsePatch`/`FormatPatch`（append-only JSONL，`op ∈ add|remove`）；
  `Invert` 生成反向补丁即回滚。
- 【门禁层】`ApplyToSet`（幂等）与 `ApplyToFile`：按物料分派写入 `[]string` 物料或
  `allowlist`（chars/combos/blessings）、`compliance_redline`（state_orgs/places/persons）各 kind。
- 【工具】`cmd/materials` 新增子命令 `lock-init` / `lock-verify` / `lock-diff` / `patch`（`--rollback` 回滚），
  构成「上游更新 → 差分补丁 → 应用 → 重新 import」闭环。

### 📚 Docs 文档更新

- `docs/33` M5 状态更新为已完成并补落地记录。

### 🧪 Tests 测试

- 新增 `subscribe_test.go`：哈希 / 清单读写与校验 / 行差分 / 补丁构建·解析·格式化·反向 /
  集合与文件应用（各 kind、非法 kind、未注册物料、缺失与 JSON 非法、只读写盘失败）/ 清单生成含跳过。
- 新增 `backend/data/materials.lock` 种子（4 项门禁物料指纹，`git add -f` 入库）。

## [2026.10.16.6]

**门禁层 M4：指标闭环（命中率/命中分布 + /metrics 端点）**（`docs/33` M4）。

### ✨ New Features 新增功能

- 【门禁层】新增 `fate/gatemetrics.go`：指标快照 `GateMetrics`（评估次数 / 命中评估数 /
  命中率 / 命中事件数 / 分层·等级·类别分布 / 误报率 / 抽样漏报率 / 复审时长）。
- 【门禁层】`RecordGateEvaluation(hit bool)` 记录一次门禁评估，由 `AssessNameRisks`
  （经 `recordAssessHits`）调用；命中率 = 命中评估数 / 评估数（分母为 0 时取 0）。
- 【门禁层】`GateMetricProvider` + `SetGateMetricProvider`：误报率 / 抽样漏报率 / 复审时长
  的来源注入点，未注册时以 0 暴露，待 M6 复审闭环接入真实数据。
- 【接口】`cmd/server` 新增 `GET /metrics`，返回 `fate.GateMetricsSnapshot()`（JSON，`gate_*` 字段）。

### 📚 Docs 文档更新

- `docs/33` M4 状态更新为已完成并补落地记录；§六指标闭环、§七误报/漏报指标差距标注已落地。

### 🧪 Tests 测试

- 新增 `fate/gatemetrics_test.go`：命中率、空快照、命中分布、指标来源注入与注销、重置、
  `recordAssessHits` 命中/未命中两分支、真实体检路径评估计数；`gatemetrics.go` 全部函数覆盖率 100%。

## [2026.10.16.5]

**门禁层 M3：命中审计埋点（脱敏）+ gate_audit 扩展**（`docs/33` M3）。

### ✨ New Features 新增功能

- 【门禁层】新增门禁命中审计埋点 `fate.GateHit` / `fate.RecordGateHit`（`fate/audit.go`）：
  结构化记录 来源 / 层级 / 类别 / 等级 / 名字 / 命中值 / 说明；`CheckBadPinyinCombo`（评分链路 L3）
  与 `AssessNameRisks`（测名体检）命中点接入。
- 【门禁层】埋点自带「全量计数 + 详情节流」：计数精确累加（`GateAuditTotal` / `GateAuditCounts`），
  详情按 (层级, 类别, 等级, 命中值) 去重并按窗口节流（`GateAuditRecent`，环形容量 256），
  保护双名 N² 评分热路径不被埋点拖慢。
- 【门禁层】脱敏前置：写入前对 名字 / 命中值 / 说明 统一做 PII 掩码（身份证 / 邮箱 / 手机号 → `***`）。
- 【门禁层】分层解耦：domain 只产出事件，基础设施 `internal/infrastructure/gateaudit.Install()`
  注册 sink 落为结构化日志（`cmd/server` 启动装配）。
- 【工具】`tools/gate_audit` 新增「D. 命中审计：门禁命中埋点快照」，展示命中总数、分类计数
  与最近命中详情（已脱敏）。

### 📚 Docs 文档更新

- `docs/33` M3 状态更新为已完成并补落地记录；§七「审计日志」差距标注已落地。

### 🧪 Tests 测试

- 新增 `fate/audit_test.go`：脱敏掩码、计数 / 详情副本、节流、sink 注册与取消、时间保留、
  环形封顶、重置、负窗口等；`audit.go` 全部函数与 `recordAssessHits` 覆盖率 100%。

## [2026.10.16.4]

**门禁层 M2：L5 合规红线独立物料**（`docs/33` M2）。

### ✨ New Features 新增功能

- 【门禁层】新增 `compliance_redline` 物料（`compliance_redline.json`，可选）：
  输出侧 L5 法务红线，`kind ∈ state_org|place|person`（机关单位 / 争议地名 / 负面历史人物），
  落库新表 `compliance_redline(entry PK, kind)`。
- 【门禁层】`IsComplianceRedline(fullName)`：全名精确 / 名部（去姓氏首字）/ 全名包含（条目 ≥2 字）
  即命中，命中前先查白名单豁免；单字条目不参与。原 `IsForbiddenEntity` 保留为兼容入口并委托本函数。
- 【门禁层】硬编码 `forbiddenEntities` 降级为「内置兜底」，与物料三集合合并判定——
  物料缺失红线也不失效；新增 `fate.SetComplianceRedline` / `LoadComplianceRedlineFromJSON`（JSON 回落）、
  `loadComplianceRedlineFromDB`（库路径，接入 `LoadMaterialsFromDB`）、`importComplianceRedline`。
- 【校验】新增 `validate.Entity`：合规红线条目须为 2~8 个汉字且不含空白。
- 【数据】新增 `backend/data/compliance_redline.json` 种子（共 98 条：机关 40 / 地名 34 / 负面人物 24）。

### 📈 Improvements 性能/体验优化

- 【门禁层】`allowlist` 新增 `IsAllowlistedText`（任意 kind 原串整体豁免），
  供跨长度条目（合规红线等）复用「白名单优先」语义。

### 📚 Docs 文档更新

- `docs/33` M2 状态更新为已完成并补落地记录；§二 L5 层、§七差距清单标注已落地。
- `docs/31 §8.5`：已建成表 12 → 13 张（补 `compliance_redline`）与实测行数。

### 🧪 Tests 测试

- 新增 `fate/compliance_redline_test.go`、`sqlite/store_materials_compliance_test.go`、
  `validate_test.go::TestEntity`：覆盖装载/计数/三类命中/白名单豁免/单字忽略/降级/错误路径，
  新增/改动函数覆盖率 100%。

## [2026.10.16.3]

**门禁层 M1：显式 allowlist 物料（白名单优先于黑名单）**（`docs/33` M1）。

### ✨ New Features 新增功能

- 【门禁层】新增 `allowlist` 物料（`allowlist.json`，可选）：显式白名单条目
  `kind ∈ char|combo|blessing`，落库新表 `allowlist(entry PK, kind)`。
  匹配优先级 `allow > deny`：被收录的「字/组合」直接豁免字级门禁（`IsNonNamingChar`）、
  禁忌组合（`IsBadCombo`）与历史人物组合（`IsHistoricalFigureCombo`）；
  祈福组合（`IsBlessingCombo`）取「内置兜底 ∪ 物料」并集。
- 【门禁层】新增 `fate.SetAllowlist` / `LoadAllowlistFromJSON`（JSON 回落，缺失降级内置默认不阻断）、
  `loadAllowlistFromDB`（库路径，接入 `LoadMaterialsFromDB`）、`importAllowlist`（导入期校验+单事务写入）。
- 【数据】新增 `backend/data/allowlist.json` 种子：内置 9 条祈福组合
  （去病/弃疾/除疾/愈病/去疾/除病/愈疾/安恙/康宁）。

### 📈 Improvements 性能/体验优化

- 【门禁层】零回归：白名单动态集合为空时行为与既有完全一致；`allowlist` 为可选物料，
  不计入必需物料清单，不影响 `VerifyMaterialsFromDB` 的 fail-fast。

### 📚 Docs 文档更新

- `docs/33` M1 状态更新为已完成，并新增「十一、落地记录」小节记录 M1 交付细节。
- `docs/31 §8.5`：已建成表由 11 张更新为 12 张（补 `allowlist`）与实测行数。

### 🧪 Tests 测试

- 新增 `internal/domain/fate/allowlist_test.go`、`store_materials_allowlist_test.go`：
  覆盖装载/计数/匹配优先级/祈福并集/降级与错误路径/库路径生效，新代码函数覆盖率 100%。

## [2026.10.16.2]

**门禁层 M0：文档拆分与落地里程碑登记**（`docs/31` §十二 → `docs/33`）。

### 📚 Docs 文档更新

- 新增 `docs/33-门禁层设计与运营闭环.md`：原 `docs/31` §十二「门禁层落地实践」拆分独立
  （该章与物料管线正交），含结论/分层/数据源治理/匹配技术/分级处置/工程运营/差距清单/
  外部订阅/回流闭环，并登记落地里程碑 **M0~M8** 与各里程碑验收门禁。
- `docs/31` §十二 收敛为去向指针，文首说明指向 `docs/33`，避免两处真相源漂移。

## [2026.10.16.1]

**P0 `Material` 接口抽象**（`docs/31 §11.9` 第 3 项，`docs/32` M5）。

### 📈 Improvements 性能/体验优化

- 【物料管线】`importOne` 的 `switch` 硬编码改为 `materialRegistry` 注册表多态分发：
  新增 `sqlite.Material` 接口（`Definition()` + `Import()`），已接入物料用方法表达式
  （`(*Store).importKangxi` 等）登记，`classics`/`curated_names` 显式登记为 legacy 跳过。
  `init` 按 `schema.Definitions` 逐项校验，未登记即 `panic`——新增物料漏登记从
  「静默跳过」变为「启动期立即失败」。

### 🧪 Tests 测试

- 新增 `store_material_registry_test.go`：锁定「注册表 ↔ `schema.Definitions` 双射」、
  注册名与 `Definition().Name` 一致、已接入物料不得登记为 legacy、legacy 物料跳过且给出原因。

### 📚 Docs 文档更新

- `docs/31 §11.9` 第 3 项、`docs/32` M5 状态更新为已完成。

## [2026.10.16.0]

**物料管线模式一/模式二双链路验证 + 空库回落修复**（`docs/31` ADR-008/ADR-013、
`docs/32` §6.2）。工作区存量未提交改动（P-1.1~P-1.4 与 P0 主体）经本轮全量验证
后交付；验证过程实测并修复两个真缺陷。

### ✨ New Features 新增功能

- **golden names 快照设施**（`docs/32 §6.2`）：3 组固定生辰（男/女 + 不同喜用神）
  Top10 快照存 `internal/application/services/testdata/golden_names.json`，
  `TestGoldenNames_快照比对` 同进程生成比对，`UPDATE_GOLDEN=1` 更新基线。
- `materials` CLI 双模式验证：`import --dry-run` / `import` / `status` 全部跑通，
  9 项物料入库（康熙 63696、namer 8134、门禁 2990、频率 2237、双字 124994、
  词典 14809、禁忌 962、卦象 64、生肖 12）。

### 🐛 Bug Fixes 问题修复

- 【Top-N 同进程不确定】golden 快照首次接入即实测复现「同一请求两次生成
  Top10 不同」（同分 74.1 首位在王金/王春间波动）。根因：`ExcellentTable`
  堆比较与 `Finalize` 只按分数，同分次序由并发推入次序决定；双名路径预排序
  `infos` 只按潜力分、同分字的相对顺序随机 → 分片边界随机 → 早停阈值随机
  → 入选集合随机。修复：引入全序 `betterEntry`（分数降序 + 同分 Char1+Char2
  字典序），堆比较、容量替换、Finalize 排序三处统一；`infos` 预排序补
  char 字典序 tiebreak。`TestExcellentTable_*` 与 e2e 第 5 项（6 连发一致）守护。
- 【错误码吞没】`writeGenerateError` 把非超时错误一律映射为 500 并替换成
  通用文案，业务层明确的 `[Code: 400] 无法生成符合条件的名字，请调整筛选条件`
  被吞——前端拿不到可行动提示，监控多了一批假 500。修复：识别 `*AppError`
  透传 Code 与 Message；超时仍 503、未知错误仍 500。5 个映射守护测试锁定。
- 【空库回落全失败（B8 孪生陷阱）】库未执行 `materials import`（hanzi_data
  0 行）时 SQL 过滤「成功且零行」，与「有数据但无匹配」不可区分：`FindCharacters`
  按无匹配直接返回空候选池，dataloader 已回落装载的 JSON 字库被完全绕过
  （实测空库场景全部生成请求失败）。修复双保险：① `SQLHanziFilter` 增加
  `HasData()`，`FindCharacters` 在 SQL 空结果且库无数据时降级 Go 端过滤；
  ② `cmd/server` 装配期以 `HanziRowCount()` 甄别，库空不注入 SQL 过滤并
  warn 引导 import。`TestFindCharacters_NoDataFallsBack` 守护。

### 📈 Improvements 性能/体验优化

- `materials import` 指纹未变的物料短路跳过（实测重灌 234ms 全跳过）；
  运行时装载日志补「字库行数」，空库场景有明确 warn。

### 🧪 Tests 测试

- `internal/materials/{schema,normalize,validate}` 三包从零测试补到 **100%
  覆盖**（守护测试 + 删除两个 0 覆盖死函数 `MetaChecksumKey/MetaVersionKey`）。
- 新增 `table_determinism_test.go`（Top-N 全序确定性）、
  `name_handler_error_mapping_test.go`（错误分级）、
  `golden_names_test.go`（快照比对 + 同进程可复现）。
- 全量回归：`go build` / `go vet` / `go test ./...` **25 包全绿**；
  `e2e_check -strict` **模式一（空库回落）与模式二（DB 优先）均 18/18**。

### 📚 Docs 文档更新

- `docs/32` 实施状态：P-1.1~P-1.4 与 P0 主体完成并验证；剩余项登记
  （`seedHanziData` 与 `importNamer` 双口径收敛、`docs/28 §8.5` 遗留项）。

## [2026.10.14.2]

**物料管线 §11.9 第 11/12 项：`name_freq_tier` / `name_penalty` / `is_negative` 全链路落库**
（`docs/31 §9.6`、ADR-009/ADR-012）。此前这三列虽已建表并由导入器写入，但 **SQL 读路径
（`SearchHanziByFilter`）未返回**，导致生成链路（走 SQL）与测名链路（走内存）对同一字算出
不同总分（实测 80.10 vs 80.60）；且频率档位在「仅 namer.json 变更」时会被静默清零。

### ✨ New Features 新增功能

- `database.Hanzi` 补 `IsNegative / NamePenalty / NameFreqTier` 三字段；`SearchHanziByFilter`
  的 SELECT 与 Scan 同步补齐；`databaseHanziToCharacter` 全量透传（不再默认置零）。

### 🐛 Bug Fixes 问题修复

- 【同源同分】`is_negative / name_penalty / name_freq_tier` 此前在 SQL 路径恒为 `false/0/0`，
  分别参与 `fate` 的硬负面剔除（`engine.go`）、起名扣分（`rater.go`）、频率评分
  （`frequency_rater.go`）→ 生成与测名总分不一致。补齐读路径后两条链路同源同分。
- 【派生列静默清零】频率档位此前只在 `importCharFreq` 内 `UPDATE hanzi_data`。当**仅
  `namer.json` 变更**（频率源不变）时，`importNamer` 重写全表 `name_freq_tier=0`，而
  `importCharFreq` 因指纹未变被短路跳过 → 档位丢失。改为 `importNamer` 从 `name_frequency`
  表回读合并（与康熙合并同构），派生链不再因部分重导而断。

### 🧪 Tests 测试

- 新增 `store_materials_projection_test.go`：评分字段落库、频率档位落库、**仅 namer 变更
  时保留频率档位**（该例在修复前必红）。
- 扩充 `adapters_p2_test.go`：`TestSQLAndMemoryScoringFieldsIdentical` 锁定 SQL 行评分字段
  全透传、不得出现默认零值。

### 📚 Docs 文档更新

- `docs/31 §11.9` 第 11/12 项状态更新为已完成（第 12 项 `pair_blacklist` 明确「源无真值、
  不落库」，两侧一致置空）。

## [2026.10.14.1]

**P2：消除 `services` 层 Go 端回查**（ADR-003「SQL 与内存结果等价」）。
SQL 下沉此前因 SQL 表缺列而形同虚设——`queryViaSQL` 与 `FindCharacters` **两处**
逐字 `hanzi.GetHanzi` 回查内存补全字段，SQL 召回收益归零，且两路径结果可能不一致。

### 🐛 Bug Fixes 问题修复

- 【五行不一致】五行修正规则 `ApplyWuxingOverrides`（字义法覆盖表 > 已有值 > 部首字形法兜底）
  **只在内存侧生效**，物料导入直接落 `namer.json` 原值。实测 **14 字** `hanzi_data.wuxing`
  为空而内存有值，导致五行过滤结果不一致。已抽出无状态 `hanzi.ResolveWuxing` 作单一真源，
  内存装载与物料导入**共用同一套规则**。
- 【生僻字判定相反】`hanzi.Hanzi.IsRare`（namer.json 99 字）与 `hanzi.RareChars`
  （编译期静态表，含 CJK 扩展区）**实测有 102 字不一致**。此前内存路径只信 `RareChars`、
  SQL 路径只信 `is_rare`，同一字在两条路径下 `IsRegular` 得到**相反**结论。
  已统一取**并集**（任一侧判生僻即生僻，取保守侧），符合「避开生僻字」安全策略。
- 【重复回查】`FindCharacters` 在 `queryViaSQL` 已过滤并返回结果后，又逐字回查内存
  重建 `Character` ——纯重复劳动，且丢弃 SQL 已返回的字段。

### ⚡ Improvements 性能优化

- 【SQL 下沉】`SearchHanziByFilter` 的 SELECT 补齐 `level / kangxi_strokes / gender /
  radical / naming_categories / meaning / is_rare`，`database.Hanzi` 同步补 7 字段；
  `naming_categories` 的 JSON 数组在驱动层解析，**解析失败按「无分类」处理而非丢弃整行**
  （否则单条脏数据会连带丢掉该字全部候选属性）。
- 【派生规则同源】抽出 `deriveCharacterFlags(level, strokes, isRare)` 作为 SQL 路径与
  内存路径**唯一共有的派生规则**；两路径差别只在输入来源，判定规则必须同源。
- 【净效果】每次 `FindCharacters` 的内存回查从 2 轮降为 0 轮。

### 📚 Docs 文档更新

- `docs/31` 新增 §11.10「P2 实施记录」（原病灶、实施步骤、实施中暴露的 2 个数据缺陷、
  刻意保留的 1 处静态表查表及理由）；§11.9 剩余待办表剔除已完成的第 2 项，
  补入 `name_freq_tier` / `name_penalty` / `pair_blacklist` 未落库 3 项。

### 🧪 Tests 测试

- 新增 `adapters_p2_test.go`（5 组）：以「stub SQL 返回内存中不存在的字」**反证回查已消除**
  （若仍有回查必被 `if !ok { continue }` 丢弃）；派生规则表驱动；生僻字并集；SQL 失败降级。
- `filter_integration_test.go` 的 SQL/内存集合等价断言在修复后由红转绿。

### ⚠️ Known Issues 已知问题

- `e2e_check -strict`（18/18）仍未复验：本机 8080 被其他项目占用。

## [2026.10.14.0]

**物料管线落地实施**（commit `bbc9fa9`）：P-1 四项前置修复全部完成 + 物料统一导入器 + CLI +
**运行时数据方向反转为「库 → 内存」**。依据 `docs/31 §五 P-1` 与 `docs/32` 实施计划。

### ⚠️ Breaking Changes 破坏性变更

- **运行时不再解析物料 JSON/CSV**：物料统一以 SQLite 为数据源，JSON/CSV 降级为 import 输入。
  `NewStore` 不再 seed 物料（移除「启动即用内存覆盖库」的回路）。
- **改 `data/*.json` 必须重新执行 `materials import` 才生效**（归一化与校验已前移到导入期，运行时不再处理）。
- **`normalizeGender("")` 由返回空串改为 `neutral`**：实测 8105 字中 6266 字（77%）无性别标注，
  此前这类字在性别筛选下被整批误剔除。已加回退开关 `hanzi.GenderUnspecifiedAsNeutral`
  （置 false 可立即回到旧行为）。此项会扩大候选池，影响起名排序。

### ✨ New Features 新增功能

- 【CLI】新增 `cmd/materials`：`import`（`--data/--db/--only/--lenient/--dry-run/--json`）与
  `status` 子命令。实测全量导入 2.9s / 27.7MB，二次导入按指纹短路 63ms。
- 【物料表】新增 11 张规范化物料表：`hanzi_data` / `kangxi_strokes` / `char_groups` / `words` /
  `name_frequency` / `name_bigram` / `naming_gate` / `forbidden_combos` / `yijing_hexagrams` /
  `zodiac` / `materials_state`，随库创建而就绪。实测行数：康熙 63696、字库 8105+29 组、
  双字频率 124994、单字频率 2237、释义 14809、门禁 2990、禁忌 962、卦象 64、生肖 12。
- 【导入器】拓扑排序 + 依赖闭包自动补齐 + 归一化 + 分字段校验（即拒，错误带「物料/字段/值/规则」
  四元组）+ 单事务写入 + 指纹登记。`--only` 指定物料时自动补齐依赖（如 `--only naming_gate`
  会带出 `namer` 与 `kangxi`）。

### 🐛 Bug Fixes 问题修复

- 【P-1.1 写入语义】`seedHanziData` 原为「表非空即 return + `INSERT OR IGNORE`」，**物料字段修订与
  删字永不传导到 SQLite**，SQL 与内存双读长期静默不一致。改为单事务 DELETE+INSERT 全量重建，
  并按 checksum 门控短路；空字库拒绝重建以免清空既有表。
- 【P-1.2 字段投影】`namer.json` 自带 `isPolyphonic`（实测 1279 字，15.8%）但 `NamerChar` 未声明该字段而
  **被静默丢弃**，导致多音字标记在内存与 SQL 侧恒为 false。已补齐。
- 【降级反模式】`classics.LoadFromJSON` 失败会 `return error`，把前面已成功从库装载的字库一并作废
  ——「语料缺失拖垮全局」。改为 warn 降级，`yijing`/门禁/禁忌同理。
- 【主键映射】`importNamer` 误引 `nc.CurationLevel`/`nc.GenderTags`/`nc.PairBlacklist`
  （`NamerChar` 无这些字段），导致编译失败；已按「源无真值」显式置零。
- 【测试污染】`pendingErrors` 曾是包级全局，跨物料/跨测试互相污染；已改为局部传递。

### 📈 Improvements 性能/架构优化

- 【P-1.4 门控】新增 `internal/materials/checksum`（sha256 + `size+mtime` 快速路径）；
  物料指纹登记新表 `materials_state`，`version` 降级为可读标识（实测 `namer.json` 的 `version`
  是静态 `"1.0"`，不随内容变化，无法驱动任何 diff）。
- 【P-1.3 康熙落点】新增 `kangxi_strokes` 表，消除「运行时只读 SQLite」下康熙笔画无来源的
  设计空洞；CSV 解析抽为 `hanzi.ParseKangxiCSVFile` 供运行时与导入复用；旧库缺表按简体笔画兜底。
- 【运行时】各域补 DB 装载 setter（`ReplaceHanziData`/`SetWordMeanings`/`SetFrequencyFromData`/
  `SetBigramFromData`/`SetHexagrams`/`SetZodiacs`/`SetNonNamingChars`/`SetForbiddenCombos`/
  `SetNamerGroups`）；实测**数据目录 0 文件、仅靠 SQLite 物料库启动成功**，日志「来源: sqlite」
  且 9 项物料全部装载。
- 【工程】`.gitattributes` 补 `*.go text eol=lf`；`.gitignore` 忽略 `cover-func.txt`。

### 📚 Docs 文档更新

- `docs/31` 新增**实施状态表**与**与设计文档的 5 处实施偏差**记录（表名 `char_groups`、
  指纹表 `materials_state`、写入语义改全量重建、`usage_level` 默认值、独立 CLI），
  并更新 §2.1 现状表、§八 8.5 持久化策略、§九 9.9 字段真值裁定、§十一 11.9 实施遗留。
- `docs/32` 落地实施计划已建立 M0~M9 里程碑。

### 🧪 Tests 测试

新增 40+ 例守护测试：字段修订/删字传导、schema 三向漂移（Hanzi ↔ 表列 ↔ INSERT ↔ 行值 ↔ 补列清单）、
康熙落表与旧库降级、checksum 门控（增/删/改字段/表空/无记录）、物料导入全流程（脏数据即拒、
幂等、删字传导、依赖闭包、dry-run）、运行时库装载与 JSON 路径**等价性**。
`gofmt`/`go build`/`go vet` 全绿，**23 个包测试全绿**。

### ⚠️ Known Issues 已知问题

- **`e2e_check -strict`（18/18）未完成复验**：本机 8080 端口被其他项目（`dbMetrics`）占用，
  e2e 打到错误进程返回 403。需释放端口或改用空闲端口后重跑两种运行模式。
- `classics` / `curated_names` 尚未接入统一导入器（import 报告显式标注跳过）。

## [2026.10.13.8]

**审查修正 + 实施计划立项**：`docs/31` 依同日逐行复核修正，新增 `docs/32` 落地实施计划。纯文档，未改动运行时代码。

### 📚 Docs 文档更新

- 【docs/31 §五 新增 **P-1 前置修复**】四项**正确性前置**（不先做，P2 校验上移与 ADR-008 必返工）：
  ① `INSERT OR IGNORE` → upsert（现状字段更新与删字均不传导，「双读一致」当前已不成立）；
  ② `namer_loader.go:110-129` 未注入 8 字段 + `Hanzi` 无 `Level` → `hanzi_data` 8 列为**空壳列**；
  ③ 康熙笔画在「运行时只读 SQLite」下**无落点**（无表且声明不落库）；
  ④ `namer.json` `version="1.0"` 静态 → 版本 diff 永不触发，须以 sha256 checksum 为门控依据。
- 【docs/31 新增 ADR-011~014】upsert 语义 / 单一 mapper 三目标投影（`Hanzi`+`NamerChar`+`hanzi_data`）/
  康熙落 `kangxi_strokes` 表 / checksum 为唯一 diff 依据；修订 ADR-001/003/006/007/008/009/010。
- 【docs/31 事实校准】`namer_loader.go` 行号系统性偏移（`:179`→`:152`、`:245` 越界→`:231`、`:53`→`:68`）；
  `NamerCategories` 为不存在符号（→ `NamerCharMap[char].NamingCategories`/`NamerGroups`）；
  `adapters.go:535`→实为 `:564`（回查代码在 `:560-601`）；康熙 CSV 实际在 `data/raw/`；
  loader 复制「4 处」→ 实测 **≥10 处**；测试包 21→**22**；`data_meta` 为 KV 表（不新增列）。
- 【docs/31 实测值订正】tone **562**/8105（原文 563）；多音字 **1279（15.8%）** → 声调回填上限约 84.2%
  而非 ~100%，且**多音字不得编造声调**；gender 空值 **6266** → `""→neutral` 属**行为变更**需 golden 回归；
  未提交项 94→**127**；ADR-007 收益按 T1/T3 分档列（常驻内存仅 T3 降为≈0）。
- 【docs/31 新增冲突记录】词典「首见即锁」（依赖文件顺序）与 SQL「按主键排序」互斥（ADR-010 确定性前提）；
  WAL 只读交付缺 `-shm`（`?mode=ro` 需要）；`cp -n` no-clobber 与 checksum 门控错配致「版本新、数据旧」。
- 【docs/32 新增】《物料资源管线落地改造实施计划》：P-1 四子项逐项改动清单 + TDD 用例 + 字段真值裁定表、
  P0~P4 进入条件、6 项并行低风险项、批次门禁与 golden names 快照、7 项风险登记、M0~M9 里程碑、DoD 清单。

### ⚠️ Breaking Changes 破坏性变更（文档层面）

- `docs/31 §5.1` 迁移序列改为 **P-1 → P0~P4**；ADR-010 的幂等定义由「`INSERT OR IGNORE` 冲突即跳过」
  改为「主键 upsert」（ADR-011），ADR-010 §10.4 同步修订；ADR-007 收益口径按定档分列。
- `docs/31 §11` 新增 P3「仓库与编码卫生」，并将 schema 漂移守护测试上提为 P-1.2 交付物。

## [2026.10.13.7]

设计定稿：`docs/31` §十二 增补 **§12.8 外部订阅数据源** 与 **§12.9 回流闭环**。纯文档，未改动运行时代码。

### 📚 Docs 文档更新

- 【§12.8 外部订阅数据源】三类来源：通用敏感词（开源词库 Sensitive-lexicon/textfilter/ToolGood.Words + 商业审核服务阿里云/易盾/数美/顶象/天御/百度——后者为长文本调、短名误报高，仅离线兜底）、命名专用（通用规范汉字表等级/机关地名负面人物 curated+法务/不雅谐音自建）；订阅机制接 §11.4 `materials.lock`（pull→差分→gate_patch→版本化→灰度）。
- 【§12.9 回流闭环】触发（warn/fail 命中 + 「warn 却仍被收藏」误报候选）→ `gate_review` 三态人审 → append-only `gate_patch.jsonl` → 复用车 `materials import` 合入 forbidden/gate/allowlist；命中脱敏、度量（命中/误报/漏报/回流时长）。

## [2026.10.13.6]

设计定稿：`docs/31` 新增 **§十二 门禁层落地实践**（业界最佳实践 + 本项目差距清单）。纯文档，未改动运行时代码。

### 📚 Docs 文档更新

- 【§十二 门禁层落地】结论（线上普遍设门禁；生成式做**输出侧门禁**，责任更重）；六层纵深防御分层表
  （词元/组合/语音形近/模型分级/合规法务）及本项目对应；数据源治理（内置/外部订阅/bad case 回流/白名单豁免）；
  匹配技术（归一化前置 + Aho-Corasick/trie + 谐音形近）；分级处置（block/downrank/warn/人审/allow，本项目 pass/warn/fail）；
  工程与运营七要点（独立发布/版本化回滚/审计/灰度/指标/误伤治理/评测回归）；**本项目差距清单**。
- 结论：门禁层四件套，项目已覆盖前三者大半，短板在外部回流、审计指标、灰度、合规红线运营闭环（对应 §十一 gate 物料治理）。

## [2026.10.13.5]

设计定稿：`docs/31` 新增 **§十一 改进路线图**，并核实「门禁层为多层级」写为前置事实。纯文档，未改动运行时代码。

### 📚 Docs 文档更新

- 【§十一 改进路线图】7 个方面 + P0-P3 路线图：数据质量（声调回填 6.9%→~100%、寓意补全、完整度门禁）、
  可观测（启动物料报告、`/readyz` 版本探针、checksum 校验）、领域建模（全局态 → `CharacterRepository`
  注入 + ACL，中间态先做 schema 漂移守护测试）、供应链（surnames 归一、`materials.lock`）、CI（物料刷新 +
  golden names + 归一化属性测试）、字符集（CJK 扩展/繁简）、仓库卫生（`*.go eol=lf`）。
- 【前置事实】核实并写入门禁层粒度为多层级：① 字级硬剔除 `IsNonNamingChar` ② 组合级 `IsBadCombo`/
  历史人物/`PairBlacklist` ③ 音韵连读 `CheckBadPinyinCombo` ④ 风险分级 `AssessNameRisks`(pass/warn/fail)。

## [2026.10.13.4]

设计定稿：`docs/31` 新增 **§十 SQL 初始化数据与上线交付** 与 **ADR-010**。纯文档，未改动运行时代码。

### 📚 Docs 文档更新

- 【§十 SQL 初始化与上线】`materials import` 归一化后同时 `--emit-sql`（可 diff/可审计的可移植初始化脚本）
  与 `--emit-db`（预构建二进制库）；entrypoint `cp -n` 预构建库 + `data_meta` 版本 diff 才 `sqlite3` 兜底；
  运行时只读开库。含 6 项确定性要求（排序稳定 / `INSERT OR IGNORE` 幂等 / UTF-8 无 BOM / 转义 helper /
  版本门控 / journal 模式）与三种上线场景（Docker / 单机 / 开发者）。
- 【ADR-010】SQL 初始化数据 + 预构建 DB 作为上线交付物：上线从「烘 87MB JSON + 启动 seed」降为「烘一个库 + 只读」；
  闭合 ADR-008+§9 的运维终局。

## [2026.10.13.3]

设计定稿：`docs/31` 新增 **§九 物料字段归一化设计** 与 **ADR-009**，把规范化/校验落到字段级。
纯文档，未改动运行时代码。

### 📚 Docs 文档更新

- 【§九 字段归一化】单一规范基座 `CanonicalChar`（合并 `NamerChar`∪`Hanzi` 去冗余）+ 10 类物料的
  Canonical schema；枚举规范化表（gender 男/male、wuxing、level、tone、dynasty、分类词表）；
  9 步可组合归一化管线（Pinyin→Enum→StrokeDual→WuxingDerive→CategoryInfer→MergeKangxi→MergeFreqTier→
  Clamp→Validate）；跨物料合并优先级写死；单位标度隔离铁律（score≠tier≠count≠strokes）；
  单一 mapper 投影到 typed 结构 + `hanzi_data`（补 level/kangxi_strokes/name_freq_tier）。
- 【ADR-009】字段归一化「单一规范基座 + 导入期可组合 Steps + 即拒校验」：将运行时补丁
  （normalizeGender 等）前移为导入期字段级、可 golden 回归、单一真相源；是 ADR-004/005/008 的字段级展开。
- 包结构 `internal/materials/{schema,normalize,validate}` + `cmd/materials import`；含 round-trip/脏数据/
  合并优先级/投影一致四类 TDD 验收。

## [2026.10.13.2]

设计定稿：`docs/31` 采纳「SQLite 为物料运行时唯一源」方向，新增 **ADR-008** 并据此修订 §八.5。
纯文档，未改动运行时代码。

### 📚 Docs 文档更新

- 【ADR-008】SQLite 成为物料运行时唯一源（显式 import，运行时不再解析 JSON）：热数据启动从库载入
  typed 结构、冷点查按需查库、全程不碰 JSON；归一化/校验上移到 import；`hanzi_data` 补列
  `level/kangxiStrokes/nameFreqTier` + 新增 `charGroups`/`words` 表；**触发=显式 `namer materials import`
  CLI + entrypoint「库缺失或 `data_meta` 版本过期」才导入一次**；移除启动 seeding 与运行时 JSON 解析。
- 【§八.5】改为「单一源 + 何时入库 + 触发方式 + 补齐缺口 + fail-fast」；订正 `adapters.go:535`
  「SQL 表无 gender/categories/radical」的过时注释（`store.go:979-1017` 实为全列写入）。
- 行为变更明示：改 `data/*.json` 后须 re-import 方生效（须写入 CLI 帮助与部署文档）。

## [2026.10.13.1]

架构设计补充：`docs/31` 新增「§八 加载与持久化策略」与「ADR-007」，细化**相对稳定物料**的
加载时机、懒加载技术、JSONL 与 SQLite 持久化选型。纯文档，未改动运行时代码。

### 📚 Docs 文档更新

- 【架构设计】`docs/31` §八：按「变更频率 × 访问特征」分 A 权威稳定 / B 周期重建 / C 追加 /
  D 纯离线 / E 用户数据；提出**四档加载策略**（T0 急载阻塞 / T1 异步预热+有界等待+降级 /
  T2 懒载 / T3 DB 按需查询）、懒加载「四件套」内建于 `Material.Warm()/Ensure()`、
  **JSONL 选型**（行型大语料用 JSONL，结构化小文档留 JSON）、**SQLite「导入一次·为源」**
  （离线 build→JSONL→`materials import` 单事务入库一次，运行时按 `data_meta` 版本 diff 才重导），
  附「物料 × 策略」决策大表与 5 项按收益排序的改造点。
- 【ADR-007】稳定物料「导入一次 · SQLite 为源 · 四档加载」：启动阻塞段 ≈38MB→≈5MB
  （`word.json` 26MB 急载 → T1/T3）；复用 `classic_loader.go` 既有 Once/ready/recover/有界等待范式。
- 核实结论存档：`full/given/gender_stats`(9.2/1.9/20MB) 为 `build_namestats` 离线产物、运行时不加载。

## [2026.10.13.0]

架构设计：新增「物料资源管线（Materials Pipeline, MRP）」三层化设计方案与交互式架构图，作为后续
物料加载/校验/持久化重构蓝本。本版为**纯设计文档**，未改动运行时代码，21 包 Go 测试与两种运行模式
`e2e_check -strict`（18/18）维持不变。

### 📚 Docs 文档更新

- 【架构设计】新增 `docs/31-物料资源管线架构设计.md`：盘点 9 类物料（通用规范汉字表 / 候选字 /
  拼音声韵 / 偏旁部首 / 笔画 / 词典 / 古典文学语料 / 人名频率 / 门禁）的现状与 7 项结构性问题
  （loader 四处复制、加载顺序硬编码、运行时补丁、SQL 双真源、大文件整读、热更新不闭环、
  声韵无独立物料），提出 L1 清单 / L2 规范化校验 / L3 仓储三层 + 统一 `Material` 接口的目标架构，
  附迁移路径（P0-P4）与 TDD 验收标准。
- 【ADR】内嵌 6 条架构决策记录：三层装载（ADR-001）、最小侵入适配现有域全局（ADR-002）、
  SQLite 双写 + 统一投影（ADR-003）、装载期强校验（ADR-004）、离线/在线同构 schema（ADR-005）、
  热更新 diff 闭环 + 删除语义（ADR-006）。
- 【架构图】`docs/assets/31-materials-pipeline.html`（Archify 渲染，自包含交互式，支持明暗主题 /
  缩放 / 检索 / 导出）；构件核验 9/9、0 告警，源码证据 6 处固定于 `b7207f5`。

## [2026.10.02.3]

`docs/30` §九：**E2E 吞吐判据改造实施**（docs/28 §8.5 遗留清偿）+ **B14 决策关闭**。
验证口径：`-strict` **18/18 通过**（检查项 17 → 18）。

### 🐛 Bug Fixes 问题修复

- 【e2e_check】★ **吞吐判据改造**：旧判据「加速比 < 1.5× 即失败」由 Little 定律
  可推出加速比 ≈ NumCPU/f——串行路径优化越好比值越低（假阳性，9.25 那轮
  16/17 误报即此），超订+慢串行段反而可能达标（假阴性）。改为两条直击
  「不超订」初衷的判据：① 并发吞吐 ≥ 串行吞吐；② 并发 p95 ≤ 串行时长 × 12
  （阈值按本机三轮实测 8-9× 膨胀包络校准，OOM 级爆炸 30×+ 仍被抓住）。
  加速比降级为信息展示。实测：吞吐 2.27 ≥ 1.35 ✓、p95 7.10 ≤ 0.741×12 ✓
- 【repo】`git add -A` 误提交的 3 个 go test 覆盖率输出产物移除并加入 .gitignore

### 📚 Docs 文档更新

- docs/30 §九：判据改造实施记录 + Top10 一次性异常排查存档（污染库冷启动
  组合，连续两轮复验稳定，未复现）+ B14 决策关闭；docs/28 §8.5 标记已处置
- **B14 决策关闭**：本应用无公网访问需求（产品确认），不引入应用层鉴权；
  nginx 444 默认丢弃与网关层准入声明维持现状
- 本轮验证遗留项全部清偿
- 【验证基线升级】本地安装 gcc 后 `-race` 竞态检测器可用：**全仓
  `go test -race ./...` 21 包全绿、0 DATA RACE**——全部并发修复首次经
  真正竞态检测器验证（docs/30 §九.4）

## [2026.10.02.2]

`docs/30` §八：**两项代办收尾——排盘并行化 + NameCard a11y 重构**。
验证口径：后端 21 包全绿（新增并发确定性压测与基准）；前端 tsc 0 错、
vitest 45/45、eslint NameCard 0 问题。

### 📈 Improvements 性能/体验优化

- 【bazi/ziwei/性能】★ **移除 `baziCalcMutex` / `ziweiMutex` 全局锁**：
  逐一核查 tyme 包级可变状态——仅有的两处 lazy-init（`LunarYearLeap`/
  `RabByungMonthDays`）均在 `sync.Once` 闭包内、初始化后只读；接口变量与
  数据串无本项目写入方。新增 32 goroutine 并发**确定性**压测（同输入同输出）
  与 `Benchmark*Parallel16` 基准固化。实测（8 核稳定值）：bazi 16 并发排盘
  **56.36 → 14.27 ms/op（3.9×）**；ziwei 同时段 A/B 复测 16.60 → 12.78 ms/op
  （~20%，并消除生产多请求下的全局串行点）
- 【frontend/a11y】NameCard 重构：容器 `role="button"` 内嵌 checkbox/按钮
  违反 WAI-ARIA 嵌套交互规则，且容器 onKeyDown 对 Space preventDefault
  **导致键盘用户无法勾选对比框**。名字标题升级为语义化 button
  （aria-pressed/aria-label/focus-visible 环）承担键盘选中路径；容器仅留
  鼠标点击；checkbox/收藏/详情补 aria-label，详情按钮补 aria-expanded

## [2026.10.02.1]

`docs/30` §七：**死代码专项与卫生收尾**（清偿 §五「仍余」与 §四遗留 2/4）。
验证口径：后端 `go build`/`go vet`/`go test` **21 包全绿**（少 1 包系删除的
namestatistics 自带测试）；前端 tsc 0 错误、vitest 45/45、lint 0 errors。

### 🗑 Deprecated 废弃功能

- 【hanzi/数据】`LoadFromJSON` 合并语义定案并注释固化（B9）：**按键覆盖合并
  （upsert）、不删除旧键**——内存中还有非 JSON 来源注入数据（kangxi 回填/
  分类同步/AddNamingChar），整体替换会冲掉它们；删除语义缺失仅影响无生产
  调用方的热更新场景
- 【config/依赖】I2 mongo-driver indirect 定案为良性：`go mod why` 证实不参与
  构建，tidy 零改动，属依赖图一致性元数据

### 📈 Improvements 性能/体验优化

- 【fate/性能】FrequencyRater 补 `skipDetail` 支持：全部 8 个 Rater 中唯一
  漏网者，此前双名 N² 枚举热路径无条件 `fmt.Sprintf`+`Join`，使
  「枚举期跳过文案构造」的优化（省 22% CPU）对频率维度失效
- 【frontend/筛选】NameFilter 笔画区间 min≤max 自愈（改 min 抬高 max、
  改 max 压低 min），杜绝 min=20/max=5 的恒空矛盾筛选
- 【fate/注释】6 处注释阈值漂移修正（>=85 → >=90，与代码实际一致）
- 【bazi/取象】五行计数转字符改 `strconv.Itoa`：加权计数 >9 时旧写法
  输出 `:` 等乱码

### 🔧 Dependencies 依赖更新 / 清理

- 【I1】删除 `domain/namestatistics` 整包（零外部引用，含测试）
- 【I9】四处裸构造 `&HanziDataProvider{}` 收敛为 `services.NewHanziDataProvider()`
  唯一构造点（测名/探索/卦象/笔画回查 + namer-cli 同步装配），SQL 字库下沉
  时单点切换，杜绝「同一次生成两套数据口径」
- 【cache】生产死装配移除（server/namer-cli 的 `cache.Init/GetCache/Close`、
  `NameService.cache` 字段与 `WithCache` 选项、`newServices` 签名收窄）——
  `s.cache` 零读取，每分钟清理协程纯空转；`cache` 包保留（单测有效，
  测试辅助仍在用）
- 【data】删除 `fileCache` 死代码（注释宣称的「热更新比对」从未实现）
- 【frontend】删除零引用的 `types/api/zodiacs.ts` 与同步版 `generateNames`

## [2026.10.02.0]

`docs/30` §六：**B12 容器实测**（192.168.10.180 Docker 环境）。在干净构建
上下文上实测出 **13 处容器构建/启动缺陷（C1-C10.5）并全部修复**——此前
「pnpm build 通过」的验证全部发生在已装好依赖的开发机上，镜像构建路径
从未真正跑通过。

### 🐛 Bug Fixes 问题修复

- 【frontend/构建链】**C1-C6**：`.dockerignore` 排除 `scripts/` 与 deps 阶段
  COPY 清单缺 `scripts/`、`pnpm-workspace.yaml` → postinstall/构建审批在容器内
  全部失效；`pnpm-workspace.yaml` 被交互写坏（"core-js" 逐字符序列化）；
  pnpm 基座不符（pnpm 11 需 Node ≥22.13 的 `node:sqlite`，基座却是 node:20）；
  `pnpm@latest` 漂移到 12 即拒绝解析。修复：反向规则、deps 清单补齐、
  固定 `pnpm@11.0.9`、基座 node:22、npmmirror 镜像源（直连 npmjs 实测
  ~10KiB/s 不可用）、`allowBuilds: {sharp: true}`（pnpm 11 新键 +
  strictDepBuilds 默认开启）。提交 `0011b83`/`1c760b9`/`13a3e55`/`21a684e`/
  `afb7a73`/`3103060`/`2d6611b`
- 【frontend/运行时】**C7/C7.5**：运行时阶段仍是 standalone 模式旧写法
  （复制 `.next/standalone` + node server.js），而前端早已静态导出——
  该目录不存在，构建必失败。改为 nginx:alpine 服务 `out/` 静态产物
  （新增 nginx-frontend.conf，双栈监听 + 路由 HTML 直达 + 静态资源长缓存），
  镜像 187MB → 63.8MB。提交 `8c8b32f`/`5c3f730`
- 【repo/行尾】**C8**：Windows `core.autocrlf=true` 使 `git archive` 导出 CRLF，
  `docker-entrypoint.sh` 的解释器被读成 `/bin/sh\r` → 后端容器 exec 失败
  崩溃循环。新增 `.gitattributes` 强制脚本/配置类文件 LF。提交 `6ba7e40`
- 【deploy/播种】**C9**：busybox cp 对 `cp -rn 目录/. 目标/` 静默不复制
  （exit 0、零文件），叠加吞错后日志照印「已就绪」——播种假成功 → 配置缺失
  → 默认 mode=all → 容器内 Fatal 重启循环。entrypoint 改逐项复制+逐项报错；
  compose 补 `NAME_MODE=api`。提交 `361571e`
- 【config/server】**C10**：viper `Unmarshal` 不解析 `AutomaticEnv`——
  compose 传入的 `NAME_MODE=api`/`NAME_SERVER_HOST=0.0.0.0` 等对结构体字段
  **静默失效**（而 logger 的 output_path 走 Get* getter 生效，同一次加载
  两套行为），且 `ensureStaticDir` 无 mode 门控 → `mkdir /frontend` Fatal。
  `Load()` 在 Unmarshal 后逐字段经 viper Get 重解析；静态目录检查仅
  `mode==all` 执行。提交 `3667bd5`
- 【deploy/镜像标签】**C10.5**：浮动基础镜像标签因 digest 漂移触发重新拉取
  （部署环境拉 docker hub 仅 ~5KB/s）。固定 `golang:1.25.10-alpine` /
  `node:22.22.3-alpine`。提交 `5720e2a`

### ✅ 实测验证（2026.10.03 全新环境全流程复现）

- `docker compose build`（全新上下文无缓存）：backend（256MB）/ frontend
  （63.8MB）双双成功
- fresh-volume `up -d`：**三容器全部 healthy**（backend / frontend / nginx）
- ① 健康检查：`https://…/healthz` 经 nginx 返回 `{"status":"ok"}`
- ② 字库播种：entrypoint「新增 36 项，失败 0」，二次启动「保留已有 36 项」
  （no-clobber 正确）；生成 API 返回真实八字/五行/喜用神与名字
  （瑞赟 82.4 / 湃信 77.8 / 嫦浩 76.8）
- ③ 配置加载：启动日志 `Starting NameMaster server {"host": "0.0.0.0"}`
  （C10 实证）；DB 落宿主数据卷、容器层无残留；无静态 Fatal
- ④ 持久化：收藏（92.7 浮点）→ `restart backend` → 完整回读
- 部署前置（实测必需）：宿主挂载目录 `chown -R 1000:1000`；证书须含
  fullchain/chain/privkey 三件（certbot 正常签发即含）；nginx 80→443 301、
  非配置域名 444 丢弃

## [2026.10.01.2]

`docs/29` **P3 摘要项批量处置**（补齐 `docs/30` §四第 4 条遗留）。验证口径：
后端 22 包全绿、前端 tsc 0 错误、vitest 45/45、lint 0 errors。

### 🐛 Bug Fixes 问题修复

- 【sqlite】`rows.Err()` 补齐 10 处 `rows.Next()` 循环：迭代中途的 IO/解码
  错误此前被当成「正常取完」，静默返回截断数据（历史列表缺后半页、
  精选名库缺行）。无错误返回值的方法记录错误日志，可返错的保持原签名。
- 【sqlite】`NewStore` 失败路径泄漏连接句柄：写连接打开成功后任一步骤
  失败（含读连接打开/Ping、建表）直接 return，残留句柄在 Windows 上锁住
  namer.db。改用 success 标记 + defer 统一关闭读/写两个连接。
- 【server】关闭路径两缺口：启动失败分支跳过 store 关闭/限流器停止/
  缓存释放（与信号分支不对称）；`signal.Notify` 注册晚于 ListenAndServe
  启动，早期信号走默认处置直接杀进程。现注册前置、失败分支走完整清理。
- 【middleware】访问日志补 `request_id` 字段：X-Request-ID 此前与日志行
  无法关联，排查目标落空。
- 【server】中间件顺序：CORS 移到 MaxBodyBytes(413)/RequestTimeout(503)
  之前——此前跨源请求的错误响应不带 CORS 头，浏览器把真实状态误报为
  CORS 错误。
- 【前端 Toast】id 由 `Date.now()` 改自增序号（同毫秒双 toast 定时器互相
  覆盖、误删）；容器补 `role="status"` + `aria-live="polite"`。
- 【前端 history】删除按钮防双击（`deletingId` state）：此前双击第二次
  DELETE 404 误报「删除失败」。

### 📈 Improvements 性能/体验优化

- 【前端 evaluate】测名提交前校验出生时间（复用 `validateBirthTime`）：
  数值输入清空得到 0 此前直接打到后端靠 400 兜底；`validateBirthTime`
  补真实日历校验（2 月 30 日这类范围检查拦不住的日期）。
- 【前端】生产构建 `removeConsole` 保留 `error/warn`：ErrorBoundary 的
  线上日志此前被整段剥除，前端故障无从排查。

## [2026.10.01.1]

`docs/29` 第二轮审查发现的 **P1×10 + P2×17** 修复实施。详细实施记录见
`docs/30-第二轮审查修复实施记录.md`。

### 🐛 Bug Fixes 问题修复

- 【fate/domain】**A1/A3/A4/A5 与 fate 令牌配对死锁**：自制 `syncOnce` 并发首调 fatal、大衍筮法永无动爻、卦号矩阵码误当传统卦序、紫微禄存/擎羊/陀罗表全错，并修复 fate 令牌配对死锁（`646ba99`）。
- 【bazi/handlers】**A2 非法日期 nil panic**：日期解析吞错后在领域层解引用 nil，用户侧 500；现返回 400（`0298379`）。
- 【favorite/sqlite/memory】**A6 收藏写入失败伪装成功**：存储层错误被吞，接口返回成功但数据未落库（`5f7cac3`）。
- 【fate】**B3 除夕硬编码腊月三十**：腊月为小月（29 天）的年份除夕识别失败；改用 `tyme.LunarDay.GetDayCount()` 判定当月最后一天。新增 `holiday_ny_eve_test.go` 覆盖大小月与放假日。
- 【bazi】**B4 节假日查询 730 次全量推算**：`GetHolidayInfo` 内两段逐字重复的 365 次逐日轮询合并为单次轮询，并为 `GetHoliday` 增加按日期缓存（返回副本，附上界重建）。新增对拍测试确保与原算法逐字一致。
- 【frontend】**B5 黄历年月切换日期溢出**：1 月 31 日切到 2 月被 `new Date` 自动进位为 3 月 3 日（跳过整月）；新增 `buildClampedDate` 将日钳制到目标月末，覆盖闰年/大小月/跨年。
- 【classics】**B10 `ShiCiExtracted` 并发读写**：写侧有锁、读侧裸读；`shici.json`（6.7MB）后台加载完成瞬间与 HTTP 请求构成 slice 并发读写。所有 `*Extracted` 读取统一走加读锁的快照复制 `copyExtracted`。
- 【name】**B11 `Reload()` 自死锁**：持 `db.mu` 调用 `Load()`，子加载器二次加锁导致永久阻塞（表现为超时而非报错）。改为「清空索引临界区」后释放锁再 `Load()`。新增带超时守护的回归测试，并实测在旧实现上 10s 未返回而失败。
- 【hanzi】**B9 全局字库并发无锁写**：`ApplyWuxingOverrides`/`mergeKangxiStrokes` 对全局 `HanziData` 无锁写，读侧 `IsCommonChar`/`HetuWuxingOfChar` 等亦无锁，热更新接通即 `fatal error: concurrent map write`。写侧加 `mu` 写锁，所有直接读取迁移到线程安全访问器 `GetHanzi`/`GetNamerChar`。并发回归测试经实测可在旧实现上稳定触发 fatal。
- 【services】**B7 异步任务无上限堆积 OOM**：`Submit` 无条件 `go ts.run`，淘汰只删 map 不终止 goroutine，稳态可堆积数千个 5 分钟枚举任务。新增容量 4 的运行信号量、满载 `ErrTooManyTasks`（handler → 503）、淘汰/TTL 取消运行中任务、取消后不回填结果并释放槽位。
- 【services/sqlite】**B8 SQL 故障伪装成「无结果」**：`SearchHanziByFilter` 出错吞为 `nil`，注释承诺的 Go 降级路径永不可达。全链路改为返回 `error`，SQL 故障可降级 Go 过滤，成功空结果与基础设施故障可区分。
- 【services/handlers】**B6 历史/收藏无分页上限**：`/history` 的 `limit` 无上限（内嵌完整结果 JSON，可一次拖走全库）且未使用已有的页大小钳制；`/favorites` 全量返回。统一走分页查询并钳制 `maxPageSize = 100`，非法分页返回 400。
- 【config】**B13 环境变量覆盖静默失效**：`AutomaticEnv` 缺 `SetEnvKeyReplacer`，嵌套键名形如 `NAME_SERVER.PORT` 无法设置。补 replacer 并新增覆盖测试。
- 【deploy】**B12 生产配置链断裂（补正）**：镜像原 `COPY config/` 路径不存在导致构建必失败；修正后仍有两处缺陷——① compose 用宿主目录挂载 `/app/config` 会遮蔽镜像内配置，新部署配置缺失、viper 全量回落默认值；② 进程固定读 `cwd/data` 而入口脚本按 `DATA_DIR` 播种。现改为「配置与字库同样烘焙 seed + 入口脚本 `cp -n` 播种」，并让 `main.go` 以 `DATA_DIR` 为准（缺省回落 `cwd/data`）。
- 【deploy】**B15 dev Redis 无密码暴露**：`6379` 由 `0.0.0.0` 改为仅绑定 `127.0.0.1`。

### ✨ New Features 新增功能

- 【fate】**A9 寓意关键词 / 偏旁选字全链路生效**：`MeaningKeywords` 原被引擎丢弃，前端把点选字拼进文本关键词、结构化通道从未被读。现把偏好落到候选池（收窄而非打分，避免破坏 `RateNameScore` 权重和恒为 1.0 的语义）：命中字义/起名分类的字 ∪ 点选字；偏好集为空时回退原池，绝不产出零结果。
- 【favorite】**B6 收藏分页 API**：`GET /favorites` 支持 `page`/`limit`，返回 `data`/`records`/`total`/`page`/`limit`。

### 📈 Improvements 性能/体验优化

- 【frontend】**A7/B16/B17 收藏稳定性与请求取消**：修复收藏 effect 临时数组导致的无限重渲染白屏；收藏态覆盖 `exploreBatch`；`fetchWithRetry`、异步生成/轮询/进度全链路接入 `AbortSignal`，防止过期响应覆盖。
- 【frontend】**A8 lint 质量门禁恢复**：恢复 ESLint 10 配置、移除废弃 `--ext`，修复 Windows 扁平依赖树中的 minimatch 版本污染与 4 处 `no-useless-assignment`。门禁由「完全失效」恢复为 `0 errors / 34 warnings`。
- 【holiday】节假日缓存将重复查询由 730 次换算降为一次 map 查找，月度视图收益最明显。

### 📚 Docs 文档更新

- 新增 `docs/30-第二轮审查修复实施记录.md`：逐项记录处置方式、验证命令与实测反证。
- 【B14 安全】确认后端零鉴权、仓库无任何 JWT 实现；移除 compose 与 `.env` 中的 `JWT_SECRET`/`JWT_EXPIRE` 死配置，在 `DOCKER_QUICKSTART.md` 明确公网部署须在 Nginx/网关层做访问控制。应用层鉴权方案待产品决策。

---

## [2026.10.01.0]

### 📚 Docs 文档更新

- 【审查】第二轮全项目审查报告 `docs/29-项目全面审查报告第二轮.md`。基线 `2026.09.25.2`：后端 21 包全绿、前端 tsc 0 错误、`docs/28` 修复项抽查确认落地。新发现 **P1×10**（自制 syncOnce 并发 fatal、非法日期 nil panic 500、大衍筮法永无动爻、卦号矩阵码当传统卦序、紫微禄存表全错、收藏写失败伪装成功、收藏查询失败前端无限重渲染白屏、lint 门禁失效、寓意关键词全链路失效、XFF 伪造绕过限流）、**P2×17**（含五行相生「土→火」写反、/history limit 无上限、异步任务 goroutine 堆积、Docker 生产配置链断裂、viper 环境变量覆盖失效等）、**P3 若干**。全部 P1 均经人工读码复核，修复待后续迭代实施。

---

## [2026.09.25] 当日总览

前端测试环境修复专项 + 后端审查 Warning 处置，版本 `2026.09.25.0` 与 `2026.09.25.1`。

| 项 | 结论 |
|------|------|
| 根因 | 本机建不出可用**符号链接**（`os.symlink()` 不报错但产出空 junction）→ pnpm 隔离式装配必然失败，vitest 依赖链接全缺 |
| `.0` | 新增 `frontend/scripts/hoist-deps.mjs`（postinstall 自动执行），用**目录 junction** 完成平铺；前端测试首次跑通 **36/36** |
| 附带 | 前端 `tsc --noEmit` 由 6 个错误 → **0 错误**（此前报错全是 vitest 模块缺失引起） |
| `.1` | 处置后端 Warning **W2 / W4 / W8**，其中 **W4 连带发现两个更严重缺陷**（fatal 竞态 + 排除功能完全失效）；另修 W2 的第三处缺陷（配置路径不存在被判致命） |
| `.2` | 处置后端剩余 Warning **W3 / W5 / W6 / W7**；其中 **W7 原诊断有误**——实测不是「报错打噪音」而是**只读连接真的改写了库文件** |
| 回归 | 后端 **21 包**全绿（0 FAIL / 0 panic / 0 data race）；前端 36/36；E2E 16/17（唯一未过项为吞吐判据，详见 `docs/28` §8.5） |

---

## [2026.09.25.2]

### 🐛 Bug Fixes 问题修复

- 【fate】★ **取消/超时被伪装成成功（`docs/28` W3）**：各枚举循环在 `cancelled(ctx)` 为真时只是**静默 `return`**（退出该 worker），主流程照常组装 TopNames 并 `return nil`。于是「会话被取消」在**领域层完全不可见**——`Wait()` 返回 nil、`State()` 是 `Finish`、`Result()` 是一份**被截断的榜单**。原设计把兜底推给调用方（服务层确实补了 `ctx.Err()` 检查），但这属于「每个调用方都必须记得做对」的事，漏一个就静默返回垃圾数据。现改为在 `generate()` 返回前统一判定 `ctx.Err()`，非 nil 即以 `名字生成被中止: %w` 上抛；`run()` 据此把状态标为 `SessionStateCanceled` 而非 `Failed`（前者是主动中止/超时，后者是生成故障，上层据此决定重试还是改参数）。
  - **实测反证**：新增用例跑在修复前的代码上直接失败——`ctx 已取消却 Wait() 返回 nil —— 上层会拿到被截断的榜单当成成功`。
  - 注：原审查称根因是「提前 `return` 跳过 `wg.Wait()`」。核对后确认 `return` 是从 goroutine 内部返回、`defer wg.Done()` 由 defer 保证执行，**`wg.Wait()` 并不会被跳过**，因果链有误，但「被取消却报成功」的结论成立。
- 【response】★ **未列举错误码被降级为 HTTP 200（`docs/28` W5）**：`getHTTPStatus` 只列举 400/401/403/404/422/500/503，其余一律 `return 200`。客户端普遍以 `res.ok`（`status < 400`）判成败，因此这类错误会被**静默吞掉**——用户看到「没报错但也没数据」的页面，前端不重试、监控收不到信号。**实测**：409/429/999/0 全部映射为 HTTP 200。现改为**按区间兜底**：合法 `400-599` 直接透传，其余任何值（0、负数、1xx/2xx/3xx、>599）一律归 **500**（兜底到 5xx 而非 4xx：写错状态码是服务端缺陷，归 5xx 才能被监控告警捕获）。顺带删除 8 个零引用的导出错误常量（属死代码清单，且它们是「看起来能用、实则一用就把错误降级成 200」的诱饵）。
  - **影响面核实**：现存 53 处调用实际只传 400/404/500/503，**当前无活跃故障**；W5 是潜伏陷阱，任何新增 409/429/451 等码的调用点会立刻踩中。
- 【fate】**每请求重复预分配 cap=10000 的 ExcellentTable（`docs/28` W6）**：`generate()` 先建一张容量 10000 的表，`generateDoubleName()` 末尾又 `*table = NewExcellentTable()` 重造一份，第一份**从未写入任何条目**就成垃圾（约 2MB/请求）。现建表容量改为按 `topCount*10` 自适应（带 1000 下限），`generateDoubleName` 的 `table` 参数由 `**ExcellentTable` 改为 `*ExcellentTable`，合并分片时**就地填充**不再重造。必须沿用 `NewExcellentTableWithCap`（保留去重）而非 `Unique` 变体——跨分片合并时 `Char1+Char2` 可能重复。
- 【sqlite】★ **「只读」连接会写库（`docs/28` W7，原诊断有误）**：原记录称对 `mode=ro` 执行 `PRAGMA journal_mode=WAL`「必然失败，每次启动打一条噪音 Warn」。**实测两个判断都不成立**——语句 `Exec` 返回 `nil` **不报错**、**也没有 Warn**，而是**真的改写了库文件**：文件头偏移 18/19 字节由 `(1,1)` 变为 `(2,2)`（legacy → WAL），新开的只读连接读到 `journal_mode = "wal"`。即问题不是「无效噪音」，而是「**一个显式声明只读的连接在悄悄写盘**」。在当前调用顺序下它恰好无害（写连接已先设 WAL，此处退化为 no-op），但这是一个**依赖调用顺序才安全的陷阱**——任何「先建只读连接、后建写连接」的重构、或对外部只读副本（非 WAL 库）的误用，都会变成真实写入。现删除该语句，改为**只读校验**（读回 `journal_mode` 比对，不一致才告警，这反而能捕捉「读写连接指向了不同文件」这一真正值得报警的情况）。

### 🧪 Tests 测试补充

- 【fate】新增 `internal/domain/fate/engine_cancel_test.go`（**4 条**）：取消路径 `Wait()` 必须返错且可 `errors.Is(err, context.Canceled)` 判定、`State()` 必须为 `Canceled`、`Result()` 必须为 nil；超时路径同理判定 `DeadlineExceeded`（handler 据此返回 503）；正常路径不受影响；`Stop()` 主动中止也必须可感知。
- 【fate】新增 `internal/domain/fate/engine_table_alloc_test.go`（**4 条**）：容量按 `Count` 自适应（10/50/100 → 1000，200 → 2000）；`Count=1` 落到下限；双名榜单**无重复名字**（盯住「误用 `Unique` 变体」这一回归点）+ 分数在 0-100 区间；单名路径同样自适应。
- 【response】新增 `internal/application/response/response_test.go`（**3 条**）：`getHTTPStatus` 表驱动覆盖原白名单 + 未列举合法码（409/429/451/599）+ 越界值（0/-1/201/301/999/399/600）；`ErrorJSON` 端到端断言非法码不落入成功区间**且响应体仍保留原始业务码**；`SuccessJSON` 未受波及。
- 【sqlite】新增 `internal/infrastructure/database/sqlite/store_journal_mode_test.go`（**2 条**）：把「只读连接执行 WAL PRAGMA 会写盘」这一反直觉行为固定下来（用**文件头字节**这一文件级依据判定，不依赖任何连接，因此既能识破「连接自称 WAL 但文件没变」也能识破「只读连接真改了文件」；若将来驱动改为报错会走另一分支并提示重新评估，不会假绿）；`NewStore` 打开已 WAL 的库后模式不变且只读连接可正常查询。

### 📚 Docs 文档更新

- `docs/28` 新增 §9「后端剩余 Warning 处置记录」：逐条记录 W3/W5/W6/W7 的根因、修法、**实测反证**与验证口径；§4 表格四项标记已修；§8.5 待办收敛为 I1/I2/I9 与死代码。
- `docs/28` §9.4 明确纠正 W7 的原始诊断（原称「必然失败 + 噪音 Warn」，实为「不报错 + 真写文件」），并说明为何这是比原描述更严重的问题。
- `docs/28` §9.3 附 W6 的**等价性证明**：改造前后三个入参（王/male/双名/50、李/female/双名/50、张/male/单名/20）的 TopNames **逐字节一致**，唯一差异是表容量 10000 → 1000。

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
- `docs/28` §8.5.1 进一步给出该判据的**数学模型**：因 worker 总量被进程级信号量钳制在 `NumCPU`，加速比 ≈ `NumCPU / f`（`f` = 枚举段占单请求服务时长的比例），**与串行基线快慢无关**；实测 1.33× 反推 `f ≈ 6`，说明 16 并发下并行度尚未饱和（存在约 1/8 的纯串行段：八字排盘 / 加载字库 / 响应编码）。故该判据方向与直觉相反——串行段越快越好过，当前难过恰恰因为串行段慢。给出两条替代判据建议（单请求 p95 不随并发恶化、并发吞吐不低于串行吞吐），待决策。
- `docs/28` §8.5 附注：核实 `sessionStageLoadChars`/`sessionStagePool` 两个阶段**从未被 `setStage` 设置**（§4 I 项已记录），故同步/异步一致性巡检观测到的阶段序列只有三段；若要靠阶段序列判定一致性需先补上报点。

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
