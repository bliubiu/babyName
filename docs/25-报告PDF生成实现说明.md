# 报告 PDF 生成实现说明

> 版本：`2026.11.24.1`
> 代码：`backend/internal/infrastructure/pdf/`、`backend/internal/application/services/report_service.go`
> 上游问题：`docs/24-遗留问题清单.md` 的 P1-3（`/report/pdf` 是硬编码占位实现）

## 0. 背景

`POST /api/v1/report/pdf` 此前输出的是**手写的占位 PDF**：正文硬编码
「宝宝起名报告」「这是一个PDF报告示例」，不含任何请求数据；`ReportService.GeneratePDF`
自述「占位实现…实际项目中应替换为专业 PDF 库」。同时 HTML 侧自成一摊：两种格式各自
解析一遍请求体、字段口径不一致。

本次把它做成真正的报告，并且**不引入任何第三方依赖**。

## 1. 为什么自己写而不是用 PDF 库

主流 Go PDF 库（gofpdf / fpdf / gopdf）输出中文都需要在运行时加载一个 TTF 并内嵌。
本实现做同一件事，但做了两个约束：

1. **依赖面收敛到标准库**（外加项目已有的 `golang.org/x/text`，仅用于字符折音）。
2. **把「字体缺失」变成显式错误**：宁可接口返回 500 + 排查指引，也不输出一份
   正文全是空白/方框的"报告"——后者用户无从判断是数据问题还是渲染问题。

## 2. 实现结构

| 文件 | 职责 |
| ---- | ---- |
| `ttf.go` | TrueType 解析：表目录、`cmap`（format 4/12）、`loca`、`glyf`、`hmtx`、复合字形依赖；`resolveGlyph` 字形解析（精确 → 折音降级） |
| `subset.go` | 字形子集化：按需收集 + 重编号 + 复合字形子号重写 + 表校验和 |
| `document.go` | PDF 对象模型、页面与内容流、顶层排版 API（Text/Paragraph/FillRect/Rule/Footer） |
| `font.go` | 系统字体发现与校验（`NAMER_PDF_FONT` → 平台常见路径），仅作最后兜底 |
| `embedded_font.go` | 内置开源中文字体（Noto Sans SC / SIL OFL 1.1）的 `go:embed` 与单例解析——**默认字体来源** |

输出：PDF 1.7，字体为 **Type0 + CIDFontType2 + Identity-H**，
CID 直接等于子集内的新字形号，因此 `/CIDToGIDMap /Identity`，无需额外映射流；
同时写入 ToUnicode CMap，保证文本**可复制、可检索**。

### 字体子集化

一次报告只用到几百个字形，而内置字体 `NotoSansSC.ttf` 有 17 MB（约 3.1 万字形）。
子集化后的报告约 **16 KB**，`pdffonts` 校验为 `CID TrueType / Identity-H / emb yes / sub yes / uni yes`。
内置字体是**可变字体**，但子集化只固化默认轮廓到静态 `glyf`，PDF 渲染器无需处理可变轴。

子集里刻意**不携带 `cmap`**（PDF 侧通过 CIDToGIDMap 定位字形，不需要它），
因此解析器允许字体缺 `cmap`。

## 3. ★ 踩过的坑：loca 错位一个字形

第一版子集把 `loca` 的结束偏移写到了 `loca[i+1]`，而正确位置是 `loca[i+2]`
（0 号字形是空的 `.notdef`，占掉 `[loca[0], loca[1])`）。

后果极其隐蔽：**PDF 能正常打开，字也确实是「汉字的样子」，但全是别的字**——
实测把「张氏宝宝起名报告」渲染成「德气容容辰告拼周」。结构校验、`pdfinfo`、
甚至「文件能解析」都不会报错，只有把渲染结果**画出来看**才会发现。

修复后新增 `TestSubsetGlyphBytesMatchOriginal`：逐字形比对「子集内 CID 的字形字节」
与「原字体对应字形字节」必须完全一致，并校验 `loca` 单调不减、首尾与 `glyf` 长度吻合、
`.notdef` 为空区间。

## 4. 字体发现与降级

- 查找顺序（见 `report_service.go:loadFont`）：
  1. `NAMER_PDF_FONT` 指定的外部字体（显式覆盖，兼容自定义场景）；
  2. **内置开源字体 Noto Sans SC（默认）**——随二进制分发，任何机器开箱可用；
  3. 平台系统字体（仅内置字体异常的兜底，走 `font.go` 的 `FindSystemCJKFont`）。
- 内置字体采用 SIL OFL 1.1 授权，可随应用嵌入分发；许可证全文见
  `backend/internal/infrastructure/pdf/fonts/OFL.txt`。
- 每个系统字体候选都会被真实解析并**校验含「中」字字形**：只检查"文件能解析"是不够的，
  macOS 的 `PingFang.ttc` 首个字体是西文，Linux 上也存在只有拉丁字形的 `.ttf`。
- 支持 `.ttc`（TrueType 集合，取第 0 个字体）；**不支持 `.otf`/CFF**，会明确报错。
- 字体缺字时的降级：先尝试**去变音符号**（`ā → a`，走 `golang.org/x/text/unicode/norm`），
  仍无字形才用 `?` 占位；缺字清单由 `Document.MissingRunes()` 返回并写日志。
  实践中发现 simhei 缺部分拉丁扩展字符，但**有** `ā/ī`（U+0101/U+012B），
  真正缺字的是少数其他字体。

## 5. 报告内容

`ReportService` 采用「解析请求 → 统一报表模型 → 渲染 PDF」两段式：

- 模型字段沿用既有请求体（`surname/gender/birth_date/birth_time/bazi/wuxing/xiyongshen/names[]`），
  详批字段包括 `rishou/nayin/zodiac/analysis/wuxing_dist/options.detail`。
- **头部**：朱砂色带 + 标题 + 右侧生成时间；下接摘要条（推荐数量、最高评分、喜用神）。
- **章节编号动态生成**：`sectionCounter` 从「一」连续编号；某节缺数据时不会出现断号
  （例如无五行分布时，命理依据仍紧接上一节编号）。
- **基本信息 / 八字与五行**：双列 label-value 网格，短字段每行两组，避免单列右侧留白；
  值过长时在单元格内折行。
- **五行分布**：横向条形图，按金木水火土固定顺序，保留浅色底槽与数值。
- **命理依据解读**：按前端提供的段落逐段折行/分页。
- **推荐名字**：表头为序号/姓名/拼音/五行/评分；前 3 名左侧色条标记（朱砂/赭金/青灰），
  评分 ≥90 朱砂、≥85 品牌蓝；寓意行以「寓意：」前缀折行，详批模式再追加维度评分。
  换页自动重画表头，避免续页列标题丢失。
- **页脚**：品牌行 + 「第 X / Y 页」。
- 空数据/非对象输入：产出一份「未提供有效报告数据」的可读报告，而不是 500。

> `/report/html` 已在接口整合中下线；当前仅 PDF 出口。

## 6. 验证方式

`tools/pdf_preview` 生成样例报告，走的是与线上一致的链路：

```bash
cd backend && go run ./tools/pdf_preview
# 输出到 ../.workbuddy/tmp/report_sample.pdf

# 1) 文本是否完整可抽取 —— 必须带 -enc UTF-8
pdftotext -enc UTF-8 -layout ../.workbuddy/tmp/report_sample.pdf -

# 2) 字体是否真内嵌且子集化
pdffonts ../.workbuddy/tmp/report_sample.pdf
# 期望：CID TrueType / Identity-H / emb yes / sub yes / uni yes

# 3) 目视检查字形（本项目实测就是靠这一步发现 loca 错位的）
pdftoppm -png -r 100 ../.workbuddy/tmp/report_sample.pdf page
```

> **`pdftotext` 默认按 Latin-1 输出**，中文会被静默丢弃、表现为"提取不到内容"。
> 排查时务必加 `-enc UTF-8`，否则会把正常的 PDF 误判成渲染失败。

## 7. 测试

| 用例 | 覆盖点 |
| ---- | ------ |
| `TestParseTTFBasic` / `TestParseTTFRejectsCFF` | 字体解析、明确拒绝 CFF |
| `TestSubsetForKeepsCompositeDeps` | 复合字形依赖被收录、子集可重新解析 |
| `TestSubsetGlyphBytesMatchOriginal` | **loca 错位回归**（逐字形字节比对 + loca 单调性） |
| `TestToUnicodeMapsEveryCID` | ToUnicode 覆盖与结构 |
| `TestDocumentProducesValidPDF` | 端到端结构、分页、子集体积占比 |
| `TestDocumentMissingGlyphFallback` | 缺字兜底与 `MissingRunes` |
| `TestWrapRespectsWidth` / `TestMeasureIsMonotonic` | 折行与测量 |
| `TestGeneratePDFContainsRealData` | **占位文案消失** + 请求数据进入 ToUnicode CMap |
| `TestReportLayoutHelpers` | 章节编号连续、最高分与分数格式化 |
| `TestParseReportModelScoreValue` | 字符串/数值评分统一转数值，供摘要与高亮 |
| `TestGeneratePDFEmptyData` / `TestGeneratePDFContextCancelled` | 空数据可读、ctx 取消即返回 |
| `TestReportHandlerGeneratePDF` | 接口层 200 + `application/pdf` + `%PDF-` 头 |

系统无中文字体的环境下，涉及渲染的用例会 `t.Skip` 而不是失败。
