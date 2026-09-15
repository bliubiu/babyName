// 命令 pdf_preview 生成一份样例起名报告 PDF，用于人工核对排版与中文渲染。
//
// 它走的是与线上一致的链路：构造 /report/pdf 的请求体 → ReportService.GeneratePDF
// → 写文件。因此也能顺带验证「本机是否能找到可用的中文字体」。
//
// 用法（在 backend 目录下）：
//
//	go run ./tools/pdf_preview                     # 输出到 .workbuddy/tmp/report_sample.pdf
//	go run ./tools/pdf_preview -out report.pdf
//
// 渲染结果可用 poppler 工具核对：
//
//	pdftotext -layout report.pdf -        # 文本是否完整可抽取
//	pdftoppm -png -r 80 report.pdf out    # 逐页转 PNG 以检查字形
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"name/internal/application/services"
	"name/internal/infrastructure/pdf"
)

func main() {
	out := flag.String("out", filepath.Join("..", ".workbuddy", "tmp", "report_sample.pdf"), "输出 PDF 路径")
	flag.Parse()

	if fontPath, err := pdf.FindSystemCJKFont(); err != nil {
		fmt.Fprintf(os.Stderr, "未找到可用中文字体：%v\n", err)
		os.Exit(2)
	} else {
		fmt.Printf("使用字体：%s\n", fontPath)
	}

	payload := map[string]interface{}{
		"surname":    "张",
		"gender":     "male",
		"birth_date": "2024年5月20日",
		"birth_time": "10:00（巳时）",
		"bazi":       "甲辰 己巳 丙申 癸巳",
		"wuxing":     "木土 土火 火金 水火",
		"xiyongshen": "水（喜神：金）",
		"names": []interface{}{
			map[string]interface{}{
				"full_name": "张珀熙", "pinyin": "zhāng pò xī", "wuxing": "水金", "score": 92.4,
				"meaning": "「珀」为琥珀，温润有光；「熙」取光明兴盛、和乐之意，《诗经·周颂》有「维清缉熙」句，寓前途明朗、气度从容。",
			},
			map[string]interface{}{
				"full_name": "张慧茗", "pinyin": "zhāng huì míng", "wuxing": "水木", "score": 91.8,
				"meaning": "「慧」为聪慧明达，「茗」指上等茶芽，合起来取「聪敏而清雅」之意，音韵上声调 1-4-2 起伏有致。",
			},
			map[string]interface{}{
				"full_name": "张茗浩", "pinyin": "zhāng míng hào", "wuxing": "木水", "score": 90.6,
				"meaning": "「茗」清雅，「浩」取其浩大宽广，《孟子》有「浩然之气」之说，寓胸怀开阔、正气充盈。",
			},
			map[string]interface{}{
				"full_name": "张珀渊", "pinyin": "zhāng pò yuān", "wuxing": "水水", "score": 89.9,
				"meaning": "「渊」为深水、渊博，《庄子·秋水》有「渊乎其不可测也」，与「珀」相配取温润而深沉的意象。",
			},
			map[string]interface{}{
				"full_name": "张赟慧", "pinyin": "zhāng yūn huì", "wuxing": "火水", "score": 88.7,
				"meaning": "「赟」为美好、文采焕发之意，与「慧」同为美德字，组合典重而不生僻。",
			},
			map[string]interface{}{
				"full_name": "张珀诗", "pinyin": "zhāng pò shī", "wuxing": "水金", "score": 88.1,
				"meaning": "「诗」取《诗经》之诗，寓文雅有教养；与「珀」搭配音韵清亮。",
			},
		},
	}

	svc := services.NewReportService()
	b, err := svc.GeneratePDF(context.Background(), payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成 PDF 失败：%v\n", err)
		os.Exit(1)
	}

	if dir := filepath.Dir(*out); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "创建输出目录失败：%v\n", err)
			os.Exit(1)
		}
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "写入 %s 失败：%v\n", *out, err)
		os.Exit(1)
	}

	abs, _ := filepath.Abs(*out)
	fmt.Printf("已生成：%s（%.1f KB）\n", abs, float64(len(b))/1024)
	fmt.Println("核对建议（务必带 -enc UTF-8，否则 pdftotext 默认按 Latin-1 输出，中文会被丢掉）：")
	fmt.Println("  pdftotext -enc UTF-8 -layout <文件> -")
	fmt.Println("  pdftoppm -png -r 90 <文件> page   # 转 PNG 目视检查字形")
}
