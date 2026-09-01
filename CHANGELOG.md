# 更新日志 (CHANGELOG)

本项目遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/) 格式，
版本号采用 **CalVer（日历版本）**：`YYYY.MM.DD.MICRO`，并在稳定分支以同号打 Tag。

> 注：自 `2026.08.24.0` 起建立统一变更日志；此前迭代未留档。

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

[2026.08.31.0]: https://github.com/bliubiao/name/releases/tag/2026.08.31.0
[2026.08.25.0]: https://github.com/bliubiao/name/releases/tag/2026.08.25.0
[2026.08.24.0]: https://github.com/bliubiao/name/releases/tag/2026.08.24.0
