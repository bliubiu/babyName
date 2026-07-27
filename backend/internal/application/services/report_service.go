package services

import (
	"bytes"
	"context"
	"fmt"
	"html"
)

// ReportService 报告服务
type ReportService struct{}

// NewReportService 创建报告服务
func NewReportService() *ReportService {
	return &ReportService{}
}

// GeneratePDF 生成PDF报告
// 占位实现：动态构建 PDF 结构并计算 xref 偏移量，避免硬编码失效
// 实际项目中应替换为专业 PDF 库
func (s *ReportService) GeneratePDF(ctx context.Context, data interface{}) ([]byte, error) {
	const header = "%PDF-1.4\n"
	// 每个 obj 的内容（不含 "N 0 obj\n" 前缀和 "\nendobj\n" 后缀）
	objects := []string{
		`<< /Type /Catalog
   /Pages 2 0 R
>>`,
		`<< /Type /Pages
   /Kids [3 0 R]
   /Count 1
>>`,
		`<< /Type /Page
   /Parent 2 0 R
   /MediaBox [0 0 612 792]
   /Contents 4 0 R
   /Resources << /Font << /F1 5 0 R >> >>
>>`,
		`<< /Length 72 >>
stream
BT
/F1 24 Tf
100 700 Td
(宝宝起名报告) Tj
ET
BT
/F1 12 Tf
100 650 Td
(这是一个PDF报告示例) Tj
ET
endstream`,
		`<< /Type /Font
   /Subtype /Type1
   /Name /F1
   /BaseFont /Helvetica
   /Encoding /WinAnsiEncoding
>>`,
	}

	var buf bytes.Buffer
	buf.WriteString(header)

	// 记录每个对象的起始字节偏移量
	offsets := make([]int, len(objects))
	for i, obj := range objects {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}

	// 写入 xref 表
	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}

	// 写入 trailer
	fmt.Fprintf(&buf, "trailer\n<< /Size %d\n   /Root 1 0 R\n>>\nstartxref\n%d\n%%%%EOF", len(objects)+1, xrefOffset)

	return buf.Bytes(), nil
}

// GenerateHTML 生成HTML报告
func (s *ReportService) GenerateHTML(ctx context.Context, data interface{}) (string, error) {
	// 检查数据类型
	if generateData, ok := data.(map[string]interface{}); ok {
		// 构建HTML报告
		htmlContent := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>宝宝起名报告</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			line-height: 1.6;
			margin: 0;
			padding: 20px;
			background-color: #f5f5f5;
		}
		.container {
			max-width: 800px;
			margin: 0 auto;
			background-color: white;
			padding: 40px;
			border-radius: 10px;
			box-shadow: 0 0 10px rgba(0,0,0,0.1);
		}
		h1 {
			color: #333;
			text-align: center;
			margin-bottom: 30px;
		}
		.section {
			margin-bottom: 30px;
		}
		.section h2 {
			color: #555;
			border-bottom: 2px solid #e0e0e0;
			padding-bottom: 10px;
			margin-bottom: 20px;
		}
		.info-item {
			margin-bottom: 10px;
		}
		.info-label {
			font-weight: bold;
			display: inline-block;
			width: 120px;
		}
		.name-list {
			list-style: none;
			padding: 0;
		}
		.name-item {
			padding: 10px;
			border-bottom: 1px solid #f0f0f0;
		}
		.name-item:hover {
			background-color: #f9f9f9;
		}
		.name-score {
			font-weight: bold;
			color: #4CAF50;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>宝宝起名报告</h1>

		<div class="section">
			<h2>基本信息</h2>
			<div class="info-item">
				<span class="info-label">姓氏：</span>
				<span>` + getStringValue(generateData, "surname") + `</span>
			</div>
			<div class="info-item">
				<span class="info-label">性别：</span>
				<span>` + getStringValue(generateData, "gender") + `</span>
			</div>
			<div class="info-item">
				<span class="info-label">出生日期：</span>
				<span>` + getStringValue(generateData, "birth_date") + `</span>
			</div>
			<div class="info-item">
				<span class="info-label">出生时间：</span>
				<span>` + getStringValue(generateData, "birth_time") + `</span>
			</div>
		</div>

		<div class="section">
			<h2>八字分析</h2>
			<div class="info-item">
				<span class="info-label">八字：</span>
				<span>` + getStringValue(generateData, "bazi") + `</span>
			</div>
			<div class="info-item">
				<span class="info-label">五行：</span>
				<span>` + getStringValue(generateData, "wuxing") + `</span>
			</div>
			<div class="info-item">
				<span class="info-label">喜用神：</span>
				<span>` + getStringValue(generateData, "xiyongshen") + `</span>
			</div>
		</div>

		<div class="section">
			<h2>推荐名字</h2>
			<ul class="name-list">`

		// 添加推荐名字列表
		if names, ok := generateData["names"].([]interface{}); ok {
			for _, name := range names {
				if nameMap, ok := name.(map[string]interface{}); ok {
					htmlContent += `
					<li class="name-item">
						<div><strong>` + getStringValue(nameMap, "full_name") + `</strong> <span class="name-score">` + getStringValue(nameMap, "score") + `分</span></div>
						<div>拼音：` + getStringValue(nameMap, "pinyin") + `</div>
						<div>寓意：` + getStringValue(nameMap, "meaning") + `</div>
						<div>五行：` + getStringValue(nameMap, "wuxing") + `</div>
					</li>`
				}
			}
		}

		htmlContent += `
			</ul>
		</div>
	</div>
</body>
</html>`
		return htmlContent, nil
	}
	return "<html><body><h1>报告生成失败</h1></body></html>", nil
}

// getStringValue 获取字符串值并转义HTML（防止XSS攻击）
func getStringValue(data map[string]interface{}, key string) string {
	if value, ok := data[key]; ok {
		var str string
		if s, ok := value.(string); ok {
			str = s
		} else {
			str = fmt.Sprintf("%v", value)
		}
		return html.EscapeString(str)
	}
	return ""
}
