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

	// 与线上一致：ReportService 默认用内置开源字体，无需系统安装中文字体。
	if _, err := pdf.LoadEmbeddedFont(); err != nil {
		fmt.Fprintf(os.Stderr, "内置中文字体不可用：%v\n", err)
		os.Exit(2)
	} else {
		fmt.Printf("使用字体：%s\n", pdf.EmbeddedFontName)
	}

	payload := map[string]interface{}{
		"surname":    "张",
		"gender":     "male",
		"birth_date": "2024年5月20日",
		"birth_time": "10:00（巳时）",
		"bazi":       "甲辰 己巳 丙申 癸巳",
		"wuxing":     "木土 土火 火金 水火",
		"xiyongshen": "水（喜神：金）",
		"rishou":     "丙",
		"nayin":      "山下火",
		"zodiac":     "龙",
		"analysis": []interface{}{
			"【八字格局】日主为丙火，生于巳月，火势得令；年柱甲辰、时柱癸巳，木火相生而水火既济，格局清透。",
			"【五行平衡】四柱五行分布为木1、火3、土1、金1、水2，火略偏旺，宜以水润之、以金泄之，避免再见木火。",
			"【喜用神补益】喜用神取水、金：水可调候润局，金能生水并有泄火之功；取名宜选属水属金之字，忌再补木火。",
		},
		"wuxing_dist": map[string]interface{}{"金": 1, "木": 1, "水": 2, "火": 3, "土": 1},
		"hexagram": map[string]interface{}{
			"number":         19,
			"name":           "地泽临",
			"symbol":         "䷒",
			"upper":          "坤",
			"lower":          "兑",
			"gua_ci":         "临：元亨利贞。至于八月有凶。",
			"xiang_ci":       "泽上有地，临；君子以教思无穷，容保民无疆。",
			"tuan_ci":        "临，刚浸而长。说而顺，刚中而应，大亨以正，天之道也。",
			"gua_de":         "上坤下兑，悦而顺",
			"interpretation": "临卦象征居上临下、以德感人；取名宜取温润守正、能容能教之意，忌过刚躁进。",
		},
		"ziwei": map[string]interface{}{
			"ming_gong":     "丙午",
			"shen_gong":     "丁未",
			"five_elements": "火六局",
			"soul":          "廉贞",
			"body":          "天相",
			"nian_zhu":      "甲辰",
			"yue_zhu":       "己巳",
			"ri_zhu":        "丙申",
			"shi_zhu":       "癸巳",
			"si_hua":        map[string]interface{}{"禄": "廉贞", "权": "破军", "科": "武曲", "忌": "太阳"},
			"zhu_xing": map[string]interface{}{
				"命宫": []interface{}{"紫微", "天府"},
				"夫妻": []interface{}{"太阳", "巨门"},
				"财帛": []interface{}{"武曲", "天相"},
				"官禄": []interface{}{"廉贞", "七杀"},
				"迁移": []interface{}{"破军"},
				"福德": []interface{}{"天机", "天梁"},
			},
			"brightness": map[string]interface{}{
				"紫微": "庙", "天府": "得", "太阳": "旺", "巨门": "平",
				"武曲": "旺", "天相": "庙", "廉贞": "利", "七杀": "旺",
			},
			"analysis": "命宫紫微天府同守，主稳重有度、格局端方；财帛武曲天相，宜守正理财；官禄廉贞七杀，事业宜攻坚但需防刚锐过甚。",
		},
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
