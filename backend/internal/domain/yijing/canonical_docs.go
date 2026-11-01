package yijing

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ============================================================================
// 易经经传文档解析器（docs/玄学/易经/*.md → 结构化语料）
//
// 单一真源：canonical_full_test.go 的金标准对拍、cmd/export_data 的
// yijing.json 生成、materials import 的库表装载，均消费本文件的解析结果，
// 避免多处解析口径漂移（docs/42 P1-② 语料结构化）。
//
// 文档格式两种：
//   - 常规式（62 卦）：彖曰/象曰成段，逐爻「象曰：…」独立行（如蒙、坤）
//   - 合体式（乾，Number==1）：彖曰独立行，大象与小象合在一行，按句号分段
// ============================================================================

// CanonicalDoc 单篇卦文档的结构化语料
type CanonicalDoc struct {
	Number    int      `json:"number"`     // 卦序 1-64
	Name      string   `json:"name"`       // 卦名
	GuaCi     string   `json:"gua_ci"`     // 卦辞全文
	DaXiang   string   `json:"da_xiang"`   // 大象传
	TuanCi    string   `json:"tuan_ci"`    // 彖传
	XiaoXiang []string `json:"xiao_xiang"` // 小象传（逐爻，与爻辞对齐）
	YaoCi     []string `json:"yao_ci"`     // 爻辞全文
}

// yaoPrefixRe 爻位前缀（用于大象行在「初六：」等爻辞起始处截断）
var yaoPrefixRe = regexp.MustCompile(`初[六九]|九[二三四五]|六[二三四五]|上[六九]|用[六九]`)

// cnDigit 单个中文数字字符 → 0-9
func cnDigit(r rune) int {
	switch r {
	case '一':
		return 1
	case '二':
		return 2
	case '三':
		return 3
	case '四':
		return 4
	case '五':
		return 5
	case '六':
		return 6
	case '七':
		return 7
	case '八':
		return 8
	case '九':
		return 9
	}
	return 0
}

// parseChineseNum 解析 1-64 的中文数字（如 二十三 → 23，六十四 → 64）
func parseChineseNum(s string) int {
	s = strings.TrimSpace(s)
	if s == "十" {
		return 10
	}
	ten := strings.Index(s, "十")
	if ten < 0 {
		return cnDigit([]rune(s)[0])
	}
	left := s[:ten]
	right := s[ten+len("十"):]
	tens := 1
	if left != "" {
		tens = cnDigit([]rune(left)[0])
	}
	ones := 0
	if right != "" {
		ones = cnDigit([]rune(right)[0])
	}
	return tens*10 + ones
}

// parseHexagramDocFull 解析单篇卦文档，返回全部结构化字段
func parseHexagramDocFull(content string) (number int, name, guaCiFull, daXiang, tuanCi string, xiaoXiang, yaoCi []string) {
	titleRe := regexp.MustCompile(`《易经》第(.+?)卦\s+(\S+?)\s`)
	if m := titleRe.FindStringSubmatch(content); m != nil {
		number = parseChineseNum(m[1])
		name = m[2]
	}

	// 卦辞全文：名：之后，至首个 彖曰 / 象曰 / 文言 之前（含后续续句）
	headerRe := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + `：(.+)$`)
	if m := headerRe.FindStringSubmatch(content); m != nil {
		rest := m[1]
		if idx := strings.Index(rest, "。彖曰"); idx >= 0 {
			guaCiFull = rest[:idx]
		} else if idx := strings.Index(rest, "。象曰"); idx >= 0 {
			guaCiFull = rest[:idx]
		} else if idx := strings.Index(rest, "。文言"); idx >= 0 {
			guaCiFull = rest[:idx]
		} else {
			guaCiFull = rest
		}
	}

	// 彖传：首个「彖曰：」之后，至其后首个「象曰：」之前（可跨段落，去空白）
	tuanStart := strings.Index(content, "彖曰：")
	if tuanStart < 0 {
		tuanStart = strings.Index(content, "彖曰")
	}
	if tuanStart >= 0 {
		rest := content[tuanStart+len("彖曰："):]
		end := strings.Index(rest, "象曰")
		if end < 0 {
			end = len(rest)
		}
		tuanCi = removeSpaces(rest[:end])
	}

	// 大象：彖曰之后的第一个「象曰：」起，至行尾
	daXiangStart := -1
	if tuanStart >= 0 {
		if idx := strings.Index(content[tuanStart:], "象曰："); idx >= 0 {
			daXiangStart = tuanStart + idx
		}
	}
	if daXiangStart < 0 {
		if idx := strings.Index(content, "象曰："); idx >= 0 {
			daXiangStart = idx
		}
	}
	daXiangLine := ""
	if daXiangStart >= 0 {
		raw := content[daXiangStart+len("象曰："):]
		if nl := strings.IndexAny(raw, "\r\n"); nl >= 0 {
			raw = raw[:nl]
		}
		daXiangLine = raw
	}

	if number == 1 {
		// 合体式（乾）：大象行按句号拆段，段[0]=大象，其余=小象
		parts := strings.Split(daXiangLine, "。")
		for i, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if i == 0 {
				daXiang = p
			} else {
				xiaoXiang = append(xiaoXiang, p+"。")
			}
		}
	} else {
		// 常规式：大象=象曰行在爻位前缀处截断；小象=其余行首「象曰：」独立行
		daXiang = truncateAtYaoPrefix(daXiangLine)
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "象曰：") {
				continue
			}
			body := strings.TrimPrefix(line, "象曰：")
			if body == daXiangLine {
				continue // 大象行本身
			}
			xiaoXiang = append(xiaoXiang, body)
		}
	}

	// 爻辞：以爻位前缀起的行
	// 排除文言「九四曰：」类行（爻位后接「曰」而非冒号）；注意不可按「行含曰」
	// 整行丢弃——大畜九三「曰闲舆卫」、困上六「曰动悔」等爻辞正文自带「曰」
	yaoRe := regexp.MustCompile(`^(初[六九]|九[二三四五]|六[二三四五]|上[六九]|用[六九])[：:](.+)$`)
	yaoSaidRe := regexp.MustCompile(`^(初[六九]|九[二三四五]|六[二三四五]|上[六九]|用[六九])曰`)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if yaoSaidRe.MatchString(line) {
			continue
		}
		if m := yaoRe.FindStringSubmatch(line); m != nil {
			yaoCi = append(yaoCi, m[2])
		}
	}
	return
}

// truncateAtYaoPrefix 在爻位前缀（初六：/九二：/用九：等）处截断文本，并去除尾部句号
func truncateAtYaoPrefix(s string) string {
	if loc := yaoPrefixRe.FindStringIndex(s); loc != nil {
		s = s[:loc[0]]
	}
	return strings.TrimSuffix(strings.TrimSpace(s), "。")
}

// removeSpaces 去除全部空白字符（彖传跨段落拼接用）
func removeSpaces(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '　':
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ParseCanonicalDocs 解析 docs 目录下全部 64 篇卦文档，按卦序返回
func ParseCanonicalDocs(docsDir string) ([]CanonicalDoc, error) {
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		return nil, fmt.Errorf("读取易经文档目录失败: %w", err)
	}
	docs := make([]CanonicalDoc, 0, 64)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		// 仅处理 64 篇卦文档（文件名含「卦」）；系辞/说卦/序卦/杂卦/index 不含
		if !strings.Contains(e.Name(), "卦") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(docsDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("读取 %s 失败: %w", e.Name(), err)
		}
		num, name, guaCi, daXiang, tuanCi, xiaoXiang, yaoCi := parseHexagramDocFull(string(raw))
		if num < 1 || num > 64 {
			continue // 非卦文档（不应到达）
		}
		docs = append(docs, CanonicalDoc{
			Number: num, Name: name,
			GuaCi: guaCi, DaXiang: daXiang, TuanCi: tuanCi,
			XiaoXiang: xiaoXiang, YaoCi: yaoCi,
		})
	}
	if len(docs) != 64 {
		return nil, fmt.Errorf("易经文档解析出 %d 卦，期望 64", len(docs))
	}
	return docs, nil
}
