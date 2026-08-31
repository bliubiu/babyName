// probe_chars 临时探针：输出双名荒谬字的 hanzi/namer 字段，辅助质量门禁决策
// 用法: go run ./cmd/probe_chars data
// 验证后删除，不进入正式代码
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
)

func main() {
	dataDir := "."
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}
	absDir, _ := filepath.Abs(dataDir)

	if err := hanzi.LoadNamerFromJSON(absDir); err != nil {
		fmt.Fprintf(os.Stderr, "加载 namer 失败: %v\n", err)
		os.Exit(1)
	}

	// 注入策展库（共现白名单），用于检查荒谬字是否曾获策展命名认可
	if names, err := loadCurated(absDir); err == nil {
		fate.SetCuratedNames(names)
		fmt.Printf("策展好名 %d 个\n\n", len(names))
	}

	// 双名 Top5 荒谬组合去重单字 + 对照好字
	chars := []string{
		"都", "腰", "路", "侍", "年", "免", "梨", "摘", "老", "冉",
		"忘", "莫", "微", "熏", "白", "河", "江", "北", "血", "凡",
		"被", "墨", "漱", "便", "游", "鹤", "沉", "媚", "复", "回",
		// 对照好字（应已在策展池、不该入门禁）
		"浩", "然", "文", "子",
	}

	fmt.Printf("%-4s %-5s %-7s %-10s %-18s %-10s %-5s %-5s %s\n",
		"字", "等级", "IsReg", "IsNameable", "策展出现", "已在门禁", "扣分", "消极", "字义(截断)")
	seen := map[string]bool{}
	for _, ch := range chars {
		if seen[ch] {
			continue
		}
		seen[ch] = true

		lvl := hanzi.GetNamerLevel(ch)
		h, ok := hanzi.HanziData[ch]
		meaning, penalty, neg, stroke := "", 0, false, 0
		if ok {
			meaning = h.Meaning
			penalty = h.NamePenalty
			neg = h.IsNegative
			stroke = h.Strokes
		}
		isReg := lvl == 1
		if lvl == 0 {
			isReg = stroke <= 25 && !hanzi.RareChars[ch]
		}
		isNameable := stroke <= 20 && isReg

		meaning = truncate(meaning, 24)
		fmt.Printf("%-4s %-5d %-7v %-10v %-18v %-10v %-5d %-5v %s\n",
			ch, lvl, isReg, isNameable, fate.IsCuratedChar(ch), fate.IsNonNamingChar(ch), penalty, neg, meaning)
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func loadCurated(dataDir string) ([]string, error) {
	path := filepath.Join(dataDir, "curated_names.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("解析 curated_names.json 失败: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name != "" {
			names = append(names, e.Name)
		}
	}
	return names, nil
}