# 报告 PDF 生成实现说明

> 版本：`2026.09.15.3`
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
| `font.go` | 系统字体发现与校验（`NAMER_PDF_FONT` → 平台常见路径） |

输出：PDF 1.7，字体为 **Type0 + CIDFontType2 + Identity-H**，
CID 直接等于子集内的新字形号，因此 `/CIDToGIDMap /Identity`，无需额外映射流；
同时写入 ToUnicode CMap，保证文本**可复制、可检索**。

### 字体子集化

一次报告只用到几百个字形，而 `simhei.ttf` 有 9.7 MB。子集化后的报告约 **53 KB**，
`pdffonts` 校验为 `CID TrueType / Identity-H / emb yes / sub yes / uni yes`。

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

- 查找顺序：`NAMER_PDF_FONT` → 平台常见路径（Windows/Linux/macOS，见 `font.go`）。
- 每个候选都会被真实解析并**校验含「中」字字形**：只检查"文件能解析"是不够的，
  macOS 的 `PingFang.ttc` 首个字体是西文，Linux 上也存在只有拉丁字形的 `.ttf`。
- 支持 `.ttc`（TrueType 集合，取第 0 个字体）；**不支持 `.otf`/CFF**，会明确报错。
- 字体缺字时的降级：先尝试**去变音符号**（`ā → a`，走 `golang.org/x/text/unicode/norm`），
  仍无字形才用 `?` 占位；缺字清单由 `Document.MissingRunes()` 返回并写日志。
  实践中发现 simhei 缺部分拉丁扩展字符，但**有** `ā/ī`（U+0101/U+012B），
  真正缺字的是少数其他字体。

## 5. 报告内容

`ReportService` 改为「解析请求 → 统一报表模型 → 分别渲染 PDF / HTML」：

- 模型字段沿用既有请求体（`surname/gender/birth_date/birth_time/bazi/wuxing/xiyongshen/names[]`），
  与 `/report/html` 的既有契约保持兼容。
- PDF：标题 + 生成时间 + 「一、基本信息 / 二、八字与五行 / 三、推荐名字」三节，
  名字表格含序号/姓名/拼音/五行/评分，寓意在行内折行；页脚带「第 X / Y 页」。
- HTML：同一份模型渲染，保留原有样式并补齐表格与转义（`html.EscapeString` 只在渲染处做一次，
  避免 PDF 里出现 `&amp;` 实体）。
- 空数据/非对象输入：产出一份「未提供有效报告数据」的可读报告，而不是 500。

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
| `TestGenerateHTMLSharesModel` / `TestGenerateHTMLEscapesUserInput` | 同模型渲染、XSS 转义 |
| `TestGeneratePDFEmptyData` / `TestGeneratePDFContextCancelled` | 空数据可读、ctx 取消即返回 |
| `TestReportHandlerGeneratePDF` | 接口层 200 + `application/pdf` + `%PDF-` 头 |

系统无中文字体的环境下，涉及渲染的用例会 `t.Skip` 而不是失败。
