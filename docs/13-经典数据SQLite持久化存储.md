# 经典数据 SQLite 持久化存储

## 一、概述

将 `backend/data/` 目录下的经典文化 JSON 数据（诗词、四书、蒙学、易经等）导入 SQLite 数据库，实现统一存储和持久化查询。

### 1.1 背景

此前仅有 `hanzi.json` 的数据写入了 SQLite 的 `hanzi` 表，其余 20+ 个经典 JSON 文件仅在进程启动时加载到内存，存在以下问题：

- **数据随进程丢失**：重启后需要重新从 JSON 加载到内存
- **查询能力受限**：无法使用 SQL 进行条件过滤、全文搜索、关联查询
- **数据结构不统一**：各 domain 包各自定义数据结构，维护成本高

### 1.2 设计目标

- **统一三表设计**：用 `classics_books → classics_sections → classics_paragraphs` 三级结构归一化所有经典数据
- **自动导入**：集成到 SQLite Store 初始化流程，首次启动自动导入，后续幂等跳过
- **覆盖全部经典文件**：排除 hanzi/word/curated_names/standard_chars/zodiac 等非经典数据文件，其余 23 个 JSON 文件全部导入

---

## 二、数据库表结构

### 2.1 classics_books（经典书籍表）

书籍级别元数据，一条记录对应一个 JSON 文件中的整部作品（如《论语》《诗经》《唐诗三百首合集》）。

```sql
CREATE TABLE classics_books (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    file_name  TEXT NOT NULL,       -- 对应 JSON 文件名（如 lunyu.json）
    category   TEXT NOT NULL,       -- 分类标签（诗经/诗词/四书/蒙学/易经/楚辞/元曲/乐府/辞赋）
    title      TEXT NOT NULL,       -- 书籍标题（如"论语""诗经"）
    author     TEXT,                -- 作者
    dynasty    TEXT,                -- 朝代
    book       TEXT,                -- 所属文集
    abstract   TEXT,                -- 简介摘要
    tags       TEXT,                -- 标签（逗号分隔）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_classics_books_file ON classics_books(file_name);
```

**字段说明**：
- `file_name` — 关联原始 JSON 文件，用于溯源和数据更新
- `category` — 分类标签，用于按类别检索（如查询所有蒙学类数据）
- `tags` — 逗号分隔的标签字符串，由 JSON 中的 tags 数组合并而来

### 2.2 classics_sections（篇章分段表）

篇章/章节级别，一条记录对应一部作品中的一个独立篇章（如"国风·周南""学而篇""离骚"）。

```sql
CREATE TABLE classics_sections (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id     INTEGER NOT NULL REFERENCES classics_books(id),
    title       TEXT,                -- 篇章标题
    chapter     TEXT,                -- 章/卷
    section     TEXT,                -- 节/部分
    author      TEXT,                -- 独立作者（某些篇章有独立作者）
    source      TEXT,                -- 来源引用
    extra       TEXT,                -- 额外元数据（JSON，预留扩展）
    sort_order  INTEGER DEFAULT 0    -- 排序序号
);
CREATE INDEX idx_classics_sections_book ON classics_sections(book_id);
```

**字段说明**：
- `title` — 篇章的主标题（如"关雎""岳阳楼记"）
- `chapter` — 上级分类（如诗经的"国风"，论语的"学而篇"）
- `section` — 更细的分类（如诗经的"周南""召南"）
- 对于数组型 JSON（诗词类），每个数组元素对应一条 section；对于对象型 JSON（典籍类），`content[]` 中的每个 chapter 对应一条 section

### 2.3 classics_paragraphs（段落内容表）

具体段落/句子级别，一条记录对应一句诗或一段正文。

```sql
CREATE TABLE classics_paragraphs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    section_id      INTEGER NOT NULL REFERENCES classics_sections(id),
    content         TEXT NOT NULL,       -- 段落内容
    paragraph_index INTEGER DEFAULT 0,   -- 段落序号
    char_count      INTEGER DEFAULT 0    -- 字符数（可用于搜索排序）
);
CREATE INDEX idx_classics_paragraphs_section ON classics_paragraphs(section_id);
```

**字段说明**：
- `content` — 正文内容，保持原始格式（含标点和空格）
- `paragraph_index` — 在篇章中的顺序号，从 0 开始
- `char_count` — 中文字符数（通过 `len([]rune(content))` 计算），可用于搜索结果的相关度排序

### 2.4 三表关系图

```
classics_books        classics_sections       classics_paragraphs
┌──────────────┐     ┌──────────────────┐    ┌───────────────────┐
│ id (PK)      │←───→│ book_id (FK)     │    │ section_id (FK)   │
│ file_name    │     │ id (PK)          │←──→│ id (PK)           │
│ category     │     │ title            │    │ content           │
│ title        │     │ chapter          │    │ paragraph_index   │
│ author       │     │ section          │    │ char_count        │
│ dynasty      │     │ author           │    └───────────────────┘
│ book         │     │ source           │
│ abstract     │     │ sort_order       │
│ tags         │     └──────────────────┘
└──────────────┘
     1 : N              1 : N
```

---

## 三、JSON 文件映射

### 3.1 文件分类与覆盖

除以下 5 个文件外，其余 JSON 文件全部导入：

| 文件 | 排除原因 |
|------|---------|
| `hanzi.json` | 已有单独的 hanzi 表导入 |
| `word.json` | 词汇数据，非经典文本 |
| `curated_names.json` | 精选名字库，单独管理 |
| `standard_chars.json` | 标准用字，偏旁部首数据 |
| `zodiac.json` | 已有内存加载 + SQLite Store 引用 |

### 3.2 文件结构与解析策略

| JSON 文件 | 分类 | 顶层结构 | 解析模式 | 预计数据量 |
|-----------|------|---------|---------|-----------|
| `shijing.json` | 诗经 | `[{title, chapter, section, content[]}]` | 数组型 | 305 篇 |
| `shici.json` | 诗词 | `[{title, author, dynasty, paragraphs[], ...}]` | 数组型 | ~800 首 |
| `chuci.json` | 楚辞 | `[{title, section, author, content[]}]` | 数组型 | ~300 篇 |
| `yuanqu.json` | 元曲 | `[{dynasty, author, paragraphs[], title}]` | 数组型 | ~11,000 首 |
| `yuefu.json` | 乐府 | `[{content, title, author, book, dynasty}]` | 数组型（长文本分句） | ~200 首 |
| `cifu.json` | 辞赋 | `[{content, title, author, book, dynasty}]` | 数组型（长文本分句） | ~50 篇 |
| `yijing.json` | 易经 | `[{id, name, number, gua_ci, xiang_ci, ...}]` | 数组型 | 64 卦 |
| `lunyu.json` | 四书 | `{title, ..., content: [{chapter, paragraphs[]}]}` | 对象型 | 20 篇 |
| `mengzi.json` | 四书 | `{title, ..., content: [{chapter, paragraphs[]}]}` | 对象型 | 14 卷 |
| `daxue.json` | 四书 | `{title, ..., content: [{chapter, paragraphs[]}]}` | 对象型 | 1 篇 |
| `zhongyong.json` | 四书 | `{title, ..., content: [{chapter, paragraphs[]}]}` | 对象型 | 1 篇 |
| `sanzijing-new.json` | 蒙学 | `{title, ..., content: [{chapter, paragraphs[]}]}` | 对象型 | ~130 段 |
| `sanzijing-traditional.json` | 蒙学 | `{title, ..., content: [{chapter, paragraphs[]}]}` | 对象型 | ~100 段 |
| `baijiaxing.json` | 蒙学 | `{origin: [{surname, place}]}` | 特殊（百家姓） | ~500 姓氏 |
| `qianziwen.json` | 蒙学 | `{spells[], content: [{chapter, paragraphs[]}]}` | 特殊（千字文） | ~250 句 |
| `qianjiashi.json` | 蒙学 | `{title, ..., content: [{chapter, author, paragraphs[]}]}` | 对象型 | ~200 首 |
| `dizigui.json` | 蒙学 | `{title, ..., content: [{chapter, paragraphs[]}]}` | 对象型 | ~100 段 |
| `shenglvqimeng.json` | 蒙学 | `{abstract, content: [{chapter, paragraphs[]}]}` | 对象型 | ~90 段 |
| `youxueqionglin.json` | 蒙学 | `{abstract, content: [{chapter, paragraphs[]}]}` | 对象型 | ~50 段 |
| `zengguangxianwen.json` | 蒙学 | `{abstract, content: [{chapter, paragraphs[]}]}` | 对象型 | ~200 段 |
| `guwenguanzhi.json` | 蒙学 | `{abstract, content: [{chapter, source, author, paragraphs[]}]}` | 对象型 | 222 篇 |
| `zhuzijiaxun.json` | 蒙学 | `{title, ..., content: [{chapter, paragraphs[]}]}` | 对象型 | ~50 段 |
| `wenzimengqiu.json` | 蒙学 | `{abstract, content: [{title, paragraphs[]}]}` | 对象型 | ~400 段 |

### 3.3 解析模式详解

#### 数组型（Array Pattern）

适用于 JSON 顶层为数组的文件，如 `shijing.json`、`shici.json` 等。

```json
[
  {
    "title": "关雎",
    "chapter": "国风",
    "section": "周南",
    "content": ["关关雎鸠，在河之洲。", "窈窕淑女，君子好逑。"]
  }
]
```

映射关系：
- 整部作品 → `classics_books` × 1（分类 + 文件名）
- 每个数组元素 → `classics_sections` × N（title/chapter/section 映射到对应字段）
- 元素中的 content/paragraphs → `classics_paragraphs` × N

**特殊处理 — 长文本分句**：`cifu.json` 和 `yuefu.json` 的 `content` 字段是单段长文本（全文），自动按句末标点（。？！；）切分为多段。

#### 对象型（Object Pattern）

适用于 JSON 顶层为对象的文件，如 `lunyu.json`、`sanzijing.json` 等。

```json
{
  "title": "论语",
  "author": "孔子及弟子",
  "dynasty": "春秋",
  "book": "四书",
  "content": [
    { "chapter": "学而篇", "paragraphs": ["子曰：学而时习之...", "有子曰：..."] },
    { "chapter": "为政篇", "paragraphs": ["子曰：为政以德...", "子曰：诗三百..."] }
  ]
}
```

映射关系：
- 顶层对象 → `classics_books` × 1（metadata 字段映射）
- content[] 中的每个 chapter → `classics_sections` × N
- 每个 chapter 中的 paragraphs → `classics_paragraphs` × N

#### 特殊结构

**百家姓**（`baijiaxing.json`）：`origin` 字段是 `[{surname, place}]` 数组。按每 20 个姓氏一组分批写入 sections，每条内容格式为 `张（清河）`。

**千字文**（`qianziwen.json`）：同时有 `spells[]`（拼音）和 `content[]`（正文）。拼音作为独立 section 写入，正文按章节写入。

---

## 四、导入流程

### 4.1 启动时序

```text
main()
  ├─ loadCulturalData(dataDir)       ← JSON → 内存（domain 原有行为不变）
  │   └─ data.Init(dataDir)
  │       ├─ hanzi.LoadFromJSON()
  │       ├─ yijing.LoadFromJSON()
  │       ├─ classics.LoadFromJSON()
  │       └─ zodiac.LoadFromJSON()
  │
  └─ initStore(dbPath, dataDir)      ← 新增：传递 dataDir
      └─ sqlite.NewStore(dbPath, dataDir)
          ├─ createTables()           ← 建表（含 classics_books/sections/paragraphs）
          ├─ seedHanziData()          ← 汉字数据（已有）
          └─ seedClassicsData(dataDir) ← 新增：经典数据导入
              ├─ 检查 classics_books 是否已有数据 → 有则跳过
              ├─ 遍历 data/ 目录下所有 JSON 文件
              │   └─ 跳过排除列表中的 5 个文件
              ├─ 按分类映射确定 category
              └─ 按解析策略导入每个文件（事务保护）
```

### 4.2 幂等机制

首次启动时导入全部数据，后续启动检查 `classics_books` 表是否有记录：

```go
var count int
s.db.QueryRow("SELECT COUNT(*) FROM classics_books").Scan(&count)
if count > 0 {
    return nil // 已有数据，跳过
}
```

如需重新导入，删除 `classics_books` 表数据即可：
```sql
DELETE FROM classics_books;
DELETE FROM classics_sections;
DELETE FROM classics_paragraphs;
```

### 4.3 事务策略

每个 JSON 文件的导入在一个独立事务中完成。大文件（如 `shici.json` ~6.8MB、`yuanqu.json` ~4.1MB）的单文件事务可能包含数万条 INSERT，SQLite 的 WAL 模式可保证写性能和大事务的稳定性。

---

## 五、代码结构

### 5.1 新增文件

```
backend/internal/infrastructure/database/sqlite/
├── store.go              ← 修改：createTables() 新增 3 张表，NewStore 调用 seedClassicsData()
└── store_classics.go     ← 新增：经典数据导入逻辑（603 行）
```

### 5.2 store_classics.go 核心结构

| 函数/类型 | 说明 |
|-----------|------|
| `seedClassicsData(dataDir)` | 入口函数，遍历 JSON 目录，分派到具体导入函数 |
| `importClassicsFile(data, fileName, category)` | 路由函数，根据文件名选择解析策略 |
| `importGeneric(data, fileName, category)` | 通用解析：先尝试数组型，后尝试对象型 |
| `importArrayWorks(works, fileName, category)` | 数组型 JSON 导入 |
| `importBookObject(book, fileName, category)` | 对象型 JSON 导入 |
| `importBaijiaxing(data, fileName, category)` | 百家姓特殊导入 |
| `importQianziwen(data, fileName, category)` | 千字文特殊导入 |
| `splitSentences(text)` | 长文本按句末标点分句工具函数 |
| `arrayWork` | 数组型 JSON 的中间结构体（含自定义 UnmarshalJSON 兼容 string/[]string） |
| `bookObject` / `bookChapter` | 对象型 JSON 的中间结构体 |
| `baijiaxing` / `surnameOrigin` | 百家姓 JSON 结构体 |
| `qianziwen` | 千字文 JSON 结构体 |

### 5.3 分类映射表

`classicsFileCategory` map 定义了每个 JSON 文件的分类归属：

```go
var classicsFileCategory = map[string]string{
    "shijing.json":           "诗经",
    "shici.json":             "诗词",
    "chuci.json":             "楚辞",
    "yuanqu.json":            "元曲",
    "yuefu.json":             "乐府",
    "cifu.json":              "辞赋",
    "yijing.json":            "易经",
    "lunyu.json":             "四书",
    "mengzi.json":            "四书",
    "daxue.json":             "四书",
    "zhongyong.json":         "四书",
    "sanzijing-new.json":     "蒙学",
    "sanzijing-traditional.json": "蒙学",
    "baijiaxing.json":        "蒙学",
    "qianziwen.json":         "蒙学",
    "qianjiashi.json":        "蒙学",
    "dizigui.json":           "蒙学",
    "shenglvqimeng.json":     "蒙学",
    "youxueqionglin.json":    "蒙学",
    "zengguangxianwen.json":  "蒙学",
    "guwenguanzhi.json":      "蒙学",
    "zhuzijiaxun.json":       "蒙学",
    "wenzimengqiu.json":      "蒙学",
}
```

---

## 六、查询示例

### 6.1 查询某部作品的所有篇章

```sql
-- 查询论语的所有篇章
SELECT s.id, s.title, s.chapter, COUNT(p.id) AS paragraph_count
FROM classics_books b
JOIN classics_sections s ON s.book_id = b.id
LEFT JOIN classics_paragraphs p ON p.section_id = s.id
WHERE b.file_name = 'lunyu.json'
GROUP BY s.id
ORDER BY s.sort_order;
```

### 6.2 按分类搜索

```sql
-- 查询所有蒙学类书籍
SELECT b.title, b.author, b.dynasty
FROM classics_books b
WHERE b.category = '蒙学'
ORDER BY b.title;
```

### 6.3 关键词搜索（LIKE）

```sql
-- 在段落内容中搜索包含"学而"的句子
SELECT b.title AS book, s.title AS section, p.content, p.char_count
FROM classics_paragraphs p
JOIN classics_sections s ON s.id = p.section_id
JOIN classics_books b ON b.id = s.book_id
WHERE p.content LIKE '%学而%'
ORDER BY p.char_count DESC;
```

### 6.4 统计各类别数据量

```sql
SELECT b.category,
       COUNT(DISTINCT b.id) AS books,
       COUNT(DISTINCT s.id) AS sections,
       COUNT(p.id) AS paragraphs
FROM classics_books b
LEFT JOIN classics_sections s ON s.book_id = b.id
LEFT JOIN classics_paragraphs p ON p.section_id = s.id
GROUP BY b.category
ORDER BY b.category;
```

---

## 七、注意事项

1. **首次启动耗时**：导入 23 个 JSON 文件（含大文件 shici.json ~6.8MB、yuanqu.json ~4.1MB）可能需要数秒，后续启动秒过。
2. **内存数据与 SQLite 共存**：原有 domain 包的内存加载不变（`dataloader.Init()`），SQLite 导入是额外持久化层，两者数据源相同。
3. **未覆盖的文件**：`word.json` 等非经典文件未纳入，如有需要可后续扩展。
4. **中文全文搜索**：当前使用 `LIKE` 模糊匹配，如需高效的全文检索（分词、相关性排序），后续可添加 SQLite FTS5 虚拟表。
