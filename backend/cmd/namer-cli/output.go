// namer-cli 命令行起名工具的输出格式化模块
//
// 提供两种输出格式：
//   - text：人类可读的终端文本（默认）
//   - json：与 Web API /api/v1/names/generate/analysis 响应结构一致，
//     便于用脚本对比 CLI 与 Web 链路的生成结果
package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"name/internal/application/services"
)

// RenderJSON 渲染 JSON 输出（结构与 API 响应 data 字段一致）
func RenderJSON(resp *services.GenerateWithAnalysisResponse) (string, error) {
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}
	return string(data), nil
}

// RenderText 渲染人类可读的终端文本
// count 为展示名字数量上限
func RenderText(resp *services.GenerateWithAnalysisResponse, count int) string {
	var b strings.Builder

	// --- 八字命理信息 ---
	b.WriteString("════════════════ 起名结果（CLI） ══════════════\n\n")
	b.WriteString(fmt.Sprintf("【八字】%s %s %s %s\n",
		resp.Bazi.Bazi.Year, resp.Bazi.Bazi.Month,
		resp.Bazi.Bazi.Day, resp.Bazi.Bazi.Hour))
	if len(resp.Bazi.Xiyongshen) > 0 {
		b.WriteString(fmt.Sprintf("【喜用神】%s\n", strings.Join(resp.Bazi.Xiyongshen, "、")))
	}
	if resp.Nayin != "" {
		b.WriteString(fmt.Sprintf("【纳音】%s\n", resp.Nayin))
	}
	if resp.Zodiac != "" {
		b.WriteString(fmt.Sprintf("【生肖】%s\n", resp.Zodiac))
	}
	b.WriteString("\n")

	// --- 名字列表 ---
	if len(resp.Names) == 0 {
		b.WriteString("未生成符合条件的名字，请调整筛选条件后重试。\n")
		return b.String()
	}

	shown := 0
	for _, n := range resp.Names {
		if shown >= count {
			break
		}
		shown++

		if n.Pinyin != "" {
			b.WriteString(fmt.Sprintf("%2d. %s（%s）\n", shown, n.FullName, n.Pinyin))
		} else {
			// 引擎未返回拼音时省略括号，避免空括号干扰阅读
			b.WriteString(fmt.Sprintf("%2d. %s\n", shown, n.FullName))
		}
		b.WriteString(fmt.Sprintf("    五行:%s  笔画:%d  综合评分:%.1f\n",
			n.Wuxing, n.Strokes, n.TotalScore))

		// 分维度评分（仅展示非零项）
		var dims []string
		if n.WuxingScore > 0 {
			dims = append(dims, fmt.Sprintf("五行 %.0f", n.WuxingScore))
		}
		if n.YinyunScore > 0 {
			dims = append(dims, fmt.Sprintf("音韵 %.0f", n.YinyunScore))
		}
		if n.MeaningScore > 0 {
			dims = append(dims, fmt.Sprintf("字义 %.0f", n.MeaningScore))
		}
		if n.ZodiacScore > 0 {
			dims = append(dims, fmt.Sprintf("生肖 %.0f", n.ZodiacScore))
		}
		if n.NoveltyScore > 0 {
			dims = append(dims, fmt.Sprintf("新颖 %.0f", n.NoveltyScore))
		}
		if n.BigramScore > 0 {
			dims = append(dims, fmt.Sprintf("共现 %.0f", n.BigramScore))
		}
		if len(dims) > 0 {
			b.WriteString(fmt.Sprintf("    维度: %s\n", strings.Join(dims, " | ")))
		}
		b.WriteString("\n")
	}

	if shown < len(resp.Names) {
		b.WriteString(fmt.Sprintf("（共 %d 个候选，仅展示前 %d 个，可用 count 参数调整）\n",
			len(resp.Names), shown))
	}

	return b.String()
}
