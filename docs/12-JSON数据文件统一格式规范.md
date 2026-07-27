# JSON 数据文件统一格式规范

## 概述

`backend/data/` 下所有 JSON 数据文件按数据性质分为 **6 大类别**，每类采用统一数据结构。

---

## 类别一：汉字库 (character dictionary)

**用途**：汉字基础数据，提供拼音、笔画、释义、五行等核心属性。

### hanzi.json

```json
[
  {
    "char": "一",
    "pinyin": "yī",
    "strokes": 1,
    "radical": "一",
    "meaning": "第一",
    "wuxing": "土",
    "gender": "通用",
    "tone": null,
    "genderTags": null,
    "styleTags": null,
    "usageLevel": null,
    "positiveScore": null,
    "phoneticScore": null,
    "modernScore": null,
    "classicalScore": null,
    "isPolyphonic": false,
    "isRare": false,
    "isNegative": false,
    "pairBlacklist": null,
    "curationLevel": null,
    "namePenalty": null
  }
]
```

### word.json（待统一）

```json
[
  {
    "char": "啊",
    "pinyin": "ā,á,ǎ,à,a",
    "strokes": 10,
    "radical": "口",
    "meaning": "叹词，表示惊叹或赞美的语气",
    "oldWord": "阿",
    "more": "详细释义..."
  }
]
```

> **说明**：word.json 提供更详细的笔画和释义，与 hanzi.json 互补。后续可通过程序合并字段。

---

## 类别二：篇章典籍 (classic text)

**用途**：经典书籍的原文段落，保留章节结构和完整正文。

**统一规范**：

```json
{
  "title": "论语",
  "author": "孔子及弟子",
  "dynasty": "春秋",
  "book": "蒙学",
  "abstract": "简介文字",
  "tags": ["儒家", "四书"],
  "content": [
    {
      "chapter": "学而",
      "paragraphs": [
        "子曰：学而时习之，不亦说乎？",
        "有朋自远方来，不亦乐乎？"
      ]
    }
  ]
}
```

### 适用文件

| 文件 | 当前格式 | 转换动作 |
|---|---|---|
| `daxue.json` | dict {chapter, paragraphs} | 套入统一模板 |
| `zhongyong.json` | dict {chapter, paragraphs} | 套入统一模板 |
| `lunyu.json` | dict[] {chapter, paragraphs}[] | 套入统一模板 |
| `mengzi.json` | dict[] {chapter, paragraphs}[] | 套入统一模板 |
| `蒙学/baijiaxing.json` | dict {paragraphs}[71] | 部分已有 tags |
| `蒙学/sanzijing-new.json` | dict {paragraphs}[131] | 合并两版 |
| `蒙学/sanzijing-traditional.json` | dict {paragraphs}[96] | 合并两版 |
| `蒙学/zhuzijiaxun.json` | dict {paragraphs}[51] | content -> paragraphs |
| `蒙学/qianziwen.json` | dict {paragraphs, spells}[250] | 保留 spells |
| `蒙学/dizigui.json` | dict {content}[8章] | 已有统一结构 |
| `蒙学/guwenguanzhi.json` | dict {abstract, content}[12卷] | 扁平化嵌套 |
| `蒙学/youxueqionglin.json` | dict {abstract, content}[4卷] | 扁平化嵌套 |
| `蒙学/wenzimengqiu.json` | dict {abstract, preface, content}[4卷] | 扁平化嵌套 |
| `蒙学/shenglvqimeng.json` | dict {abstract, content}[2卷] | 扁平化嵌套 |
| `蒙学/zengguangxianwen.json` | dict {abstract, content}[2章] | 已有统一结构 |
| `蒙学/qianjiashi.json` | dict {content}[4类] | 扁平化嵌套 |
| `蒙学/tangshisanbaishou.json` | dict {content}[7类] | 扁平化，拆分到 shici.json |

---

## 类别三：诗词曲 (poetry)

**用途**：完整诗文数据，用于「诗词典故」功能。

**统一规范**：

```json
[
  {
    "title": "静夜思",
    "author": "李白",
    "dynasty": "唐",
    "type": "五言绝句",
    "book": "唐诗三百首",
    "source": "唐诗",
    "tags": ["思乡", "月亮"],
    "paragraphs": [
      "床前明月光，疑是地上霜。",
      "举头望明月，低头思故乡。"
    ]
  }
]
```

### 适用文件

| 文件 | 条目数 | 当前格式 | 转换动作 |
|---|---|---|---|
| `shici.json` | 822 | 已有统一格式 | 无需转换 |
| `cifu.json` | 47 | {content, title, author, book, dynasty} | content→paragraphs，book→source |
| `yuefu.json` | 203 | {content, title, author, book, dynasty} | content→paragraphs，book→source |
| `yuanqu.json` | 11,057 | {dynasty, author, paragraphs, title} | 补充 type="元曲" |
| `蒙学/tangshisanbaishou.json` | 7类x多首 | 嵌套三层 | 扁平化为 dict[]，合并到 shici.json |

---

## 类别四：经典用字 (classic chars)

**用途**：从经典文献中提取的单个汉字，用于起名时关联出处。

**统一规范**：

```json
[
  {
    "char": "关",
    "pinyin": "guān",
    "meaning": "关隘、涉及",
    "source": "诗经·国风",
    "work": "诗经",
    "chapter": "关雎",
    "wuxing": "木",
    "gender": "通用"
  }
]
```

### 适用文件

| 文件 | 条目数 | 当前格式 |
|---|---|---|
| `shijing.json` | 16,372 | 已有统一格式 |
| `chuci.json` | 10,722 | 已有统一格式 |
| 蒙学文件 | - | 待从 content 提取汉字生成 |

---

## 类别五：候选名库 (curated names)

**用途**：预置的名字推荐，用于名字生成时的候选池。

**统一规范**：

```json
[
  {
    "name": "子轩",
    "pinyin": "zǐ xuān",
    "gender": "male",
    "source": "诗云精选",
    "meaning": "气宇轩昂",
    "wuxing": "木",
    "yinyunScore": 85,
    "styles": ["古典", "文雅"],
    "tags": ["诗云", "唐诗"]
  }
]
```

### 适用文件

| 文件 | 条目数 | 当前格式 |
|---|---|---|
| `curated_names.json` | 5,457 | 已有（`yinyun_score` 转为 `yinyunScore`） |

---

## 类别六：专项数据 (specialized)

### 易经 yijing.json

```json
[
  {
    "id": 1,
    "name": "乾",
    "number": 1,
    "symbol": "䷀",
    "upperTrigram": 1,
    "lowerTrigram": 1,
    "guaCi": "元亨利贞",
    "xiangCi": "天行健君子以自强不息",
    "yaoCi": ["潜龙勿用", "见龙在田"],
    "interpretation": "乾卦象征着天..."
  }
]
```

> 当前字段使用小驼峰 → 统一为小驼峰。

### 生肖 zodiac.json

```json
[
  {
    "id": 1,
    "name": "鼠",
    "wuxing": "水",
    "compatible": ["牛", "龙", "猴"],
    "conflicting": ["马", "羊", "兔"],
    "avoidChars": "午马未羊卯兔",
    "luckyNumber": [1, 6],
    "luckyColor": "蓝、金、绿",
    "luckyDirection": "北、西",
    "goodPianpang": "宀、口、王",
    "badPianpang": "午、未、卯"
  }
]
```

### 标准用字 standard_chars.json

```json
[
  {
    "radical": "氵",
    "name": "三点水",
    "meaning": "与水有关",
    "chars": ["江", "河", "海", "湖", "波"]
  }
]
```

---

## 类别总表

| 类别 | 数据性质 | 顶层结构 | 代表字段 | 文件数 |
|---|---|---|---|---|
| 一 | 汉字库 | `dict[]` | char, pinyin, strokes, wuxing | 2 |
| 二 | 篇章典籍 | `dict` | title, author, content[{chapter, paragraphs}] | 17 |
| 三 | 诗词曲 | `dict[]` | title, author, paragraphs[], tags | 4 |
| 四 | 经典用字 | `dict[]` | char, pinyin, source, work | 2 |
| 五 | 候选名库 | `dict[]` | name, pinyin, gender, source | 1 |
| 六 | 专项数据 | `dict[]` | 各领域特有字段 | 3 |

---

## 转换优先级

1. **典籍类**（类别二）→ 统一为 `{title, author, content[{chapter, paragraphs}]}` 模板
2. **诗词曲**（类别三）→ 统一为 `dict[]` 格式，嵌套扁平化
3. **蒙学** → 按典籍类统一，嵌套扁平化
4. **用字类**（类别四）→ 典籍转换后提取汉字自动生成
5. **汉字库**（类别一）→ word.json 字段合并到 hanzi.json

---

## 字段命名规则

- 全部使用 **小驼峰**（camelCase）：`yinyunScore` √，`yinyun_score` ✗
- 汉字相关字段用 `char` 而非 `word`
- 源出用 `source` 而非 `book`
- 性别用 `gender`（male/female/通用）
- 五行用 `wuxing`
- 诗文内容统一用 `paragraphs[]`
