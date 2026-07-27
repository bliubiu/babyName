# data 目录 JSON 文件整合方案

## 一、现状分析

### 1.1 数据加载架构碎片化

当前 JSON 数据加载分散在多个域包的独立 loader 中：

| 加载器 | 所属包 | 加载文件 | 加载时机 |
|--------|--------|---------|---------|
| `classics/loader.go` | classics | `shijing.json`, `chuci.json` | 应用启动时 |
| `yijing/loader.go` | yijing | `yijing.json` | 应用启动时 |
| `zodiac/loader.go` | zodiac | `zodiac.json` | 应用启动时 |
| `hanzi/loader.go` | hanzi | `hanzi.json` | 应用启动时 |
| `name/name_db.go` | name | `curated_names.json`, `standard_chars.json` | 应用启动时 |
| `infrastructure/data/dataloader.go` | dataloader | 通用加载器（未接入） | 未使用 |

### 1.2 数据重复问题

`classics/classics.go` 和 `classics/poetry.go` 中的硬编码数据与 JSON 文件内容重复：

- `shijing.json` 包含完整诗经数据
- `classics.go` 额外硬编码了 ~80 条 `ClassicName{}`（诗经＋楚辞单字）
- `poetry.go` 额外硬编码了 ~300 条 `PoetryChar{}`（诗经＋楚辞＋唐诗）
- `ci.db` SQLite 数据库已包含完整汉字数据，但 `hanzi.json` 同时也在加载

### 1.3 未使用数据一览

data/ 目录当前约 1300+ 个文件，其中大量文件未被任何代码使用：

| 文件类别 | 数量 | 来源 | 是否使用 |
|---------|------|------|---------|
| `shijing.json` | 1 | chinese-poetry | ✅ 已使用 |
| `chuci.json` | 1 | chinese-poetry | ✅ 已使用 |
| `yijing.json` | 1 | 易经 | ✅ 已使用 |
| `zodiac.json` | 1 | 生肖 | ✅ 已使用 |
| `curated_names.json` | 1 | 精选名字 | ✅ 已使用 |
| `standard_chars.json` | 1 | 标准用字 | ✅ 已使用 |
| `hanzi.json` | 1 | 汉字数据 | ✅ 已使用 |
| `ci.db` | 1 | SQLite汉字库 | ✅ 已使用 |
| `poet.tang.*.json` | 58 | 全唐诗 | ❌ 未使用 |
| `poet.song.*.json` | 255 | 全宋诗 | ❌ 未使用 |
| `ci.song.*.json` | 22 | 全宋词 | ❌ 未使用 |
| `001.json`~`900.json` | 900 | chinese-poetry编号诗 | ❌ 未使用 |
| `authors.*.json` | 3 | 诗人传记 | ❌ 未使用 |
| 四书（daxue/lunyu/mengzi/zhongyong）| 4 | 四书 | ❌ 未使用 |
| `shuimotangshi.json` | 1 | 水墨唐诗 | ❌ 未使用 |
| `youmengying.json` | 1 | 幽梦影 | ❌ 未使用 |
| `yuanqu.json` | 1 | 元曲 | ❌ 未使用 |
| `hanzi_data.json` | 1 | 汉字数据 | ❌ 未使用 |
| 乱码文件名的JSON | 10+ | 编码损坏 | ❌ 未使用 |
| `dictionary.txt`/`UpdateCi.py` | 2 | 辅助文件 | ❌ 未使用 |
| `README.md` | 1 | 全唐诗说明 | ❌ 未使用 |

## 二、整合方案

### 2.1 总体原则

1. **数据与代码分离** — JSON 数据仅存放在 data/ 目录，Go 代码中不硬编码数据
2. **统一加载入口** — 所有 JSON 通过 `infrastructure/data/dataloader.go` 统一加载，各域包通过 DataLoader 获取数据
3. **合并重复数据** — 删除 `classics.go` 和 `poetry.go` 中的硬编码结构体，统一从 JSON 加载
4. **精简目录** — 未使用的 JSON 文件移出 data/ 目录，减少项目体积

### 2.2 实施步骤

#### 第一阶段：清理未用数据

将以下未使用的文件移出 data/ 目录（归档到 `data/archive/`）：

```
poet.tang.*.json      → archive/poet.tang/
poet.song.*.json      → archive/poet.song/
ci.song.*.json        → archive/ci.song/
001.json~900.json     → archive/numbers/
authors.*.json        → archive/authors/
daxue.json            → archive/classics/
lunyu.json            → archive/classics/
mengzi.json           → archive/classics/
zhongyong.json        → archive/classics/
shuimotangshi.json    → archive/other/
youmengying.json      → archive/other/
yuanqu.json           → archive/other/
hanzi_data.json       → archive/other/
dictionary.txt        → archive/other/
UpdateCi.py           → archive/other/
README.md             → archive/other/
乱码文件名JSON        → archive/garbled/
```

清理后 data/ 保留文件：

```
data/
├── shijing.json         # 诗经
├── chuci.json           # 楚辞
├── yijing.json          # 易经
├── zodiac.json          # 生肖
├── curated_names.json   # 精选名字
├── standard_chars.json  # 标准用字
├── hanzi.json           # 汉字数据
└── ci.db                # SQLite汉字库
```

#### 第二阶段：统一 DataLoader

1. 改造 `infrastructure/data/dataloader.go`：
   - 添加 `Init(dataDir string)` 启动时一次性加载所有 JSON
   - 各域包通过 `GetJSON(filename string)` 获取解析后的数据
   - 热更新支持：文件变更时自动重载

2. 改造各域 loader：
   - `classics/loader.go` — 改为通过 DataLoader 获取数据
   - `yijing/loader.go` — 同上
   - `zodiac/loader.go` — 同上
   - `name/name_db.go` — 保留自加载（因需要处理名字入库逻辑）
   - `hanzi/loader.go` — 同上

#### 第三阶段：去硬编码

1. `classics/classics.go` — 删除 `ShijingNames` / `ChuciNames` 的硬编码列表，改用 DataLoader 从 `shijing.json` / `chuci.json` 加载
2. `classics/poetry.go` — 删除 `PoetrySources` 的所有硬编码数据，改用 DataLoader 加载
3. 确保 `FindPoetryByChars()` / `GetPoetryCharList()` 等查询函数仍能正常工作

#### 第四阶段：扩展诗词出典

后续可接入 real 唐诗宋词数据用于名字出典扩展（当前 phase 暂不做）：

1. 编写 `poet.tang.parser.go` 处理全唐诗 JSON 格式（每个诗条目含 author/paragraphs/title）
2. 从诗句中提取双音节词，建立名字→出处索引
3. 接入 `classics.FindPoetryByChars()` 出典查询接口

### 2.3 影响分析

| 变更 | 影响 |
|------|------|
| 移出未用JSON | 无功能影响，仅减少 ~500MB 项目体积 |
| 删除硬编码数据 | `GetPoetryCharList()` 等查询函数依赖的数据改为文件加载，启动时自动填充 |
| DataLoader 统一 | 各包加载路径不变，仅内部实现改为走 DataLoader |

## 三、启动流程

整合后的应用启动流程：

```
main()
 └─ InitConfig()
 └─ InitLogger()
 └─ InitDatabase()
 └─ InitDataLoader("data/")          ← 新增：统一加载所有 JSON
     ├─ load shijing.json → classics 域
     ├─ load chuci.json   → classics 域
     ├─ load yijing.json  → yijing 域
     ├─ load zodiac.json  → zodiac 域
     └─ load hanzi.json   → hanzi 域
 └─ InitNameDB("data/")              ← name 域保留自加载
 └─ InitRouter()
 └─ StartServer()
```

## 四、近期执行清单

1. 创建 `data/archive/` 子目录
2. 将未用 JSON 按分类移入 `data/archive/`
3. 删除 `classics/classics.go` 硬编码列表，改为从 DataLoader 加载
4. 删除 `classics/poetry.go` 硬编码数据，改为从 DataLoader 加载
5. 改造 `infrastructure/data/dataloader.go` 为统一数据源
6. 逐一改造 domain loader 接入 DataLoader
7. 编译验证 + 运行测试
