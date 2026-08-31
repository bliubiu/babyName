// clean_curated 清洗 curated_names.json 中的垃圾条目
//
// 背景：探查发现策展库混入大量历史人物字/号（仲尼/叔敖/献之/与砺/若兮 等）
// 与荒谬组合（自姐/隶赌/暮货 等），合计约 962 条垃圾，不得再作为
// sqlite 种子 / API 推荐池 / 共现分白名单的数据源。
//
// 实测复核（verify_fate 第二轮）：清洗后仍有「鲁章/鲁听/震男/骏男/男兆」
// 等荒谬组合霸榜，根因是策展库 4697 条中 4619 条（98%）为
// source=「古人云_历史人名」、meaning=「历史人物字号/别名」的采集条目。
// 历史人物字号（鲁章/鲁听/震男/骏男/男兆/渊明 等）是古人字号而非现代好名，
// 混入策展库后，荒谬组合恰好命中 IsCuratedName 白名单 → 豁免封顶 → 高分霸榜。
// 因此新增规则 0：按来源/标签剔除历史人物字号采集条目（数据级根治）。
//
// 清洗规则（与 fate 引擎语义门禁完全一致，避免规则漂移）：
//   0. 来源/标签为历史人物字号采集（source=古人云_历史人名 或
//      meaning=历史人物字号/别名 或 tags 含「历史人名」）→ 剔除（数据级根治）
//   1. 名字长度必须为 2（策展名是双字名）
//   2. 任一位置为语义门禁字（IsNonNamingChar：虚词/排行字/口语物名/数字量词）→ 剔除
//   3. 历史人物字/号专名（IsHistoricalFigureCombo）→ 剔除
//   4. 禁忌组合（IsBadCombo：父母/蜂蜜 等）→ 剔除
//   5. 任一位置为硬禁用字（IsHardNegativeChar：屎/尸/淫/盗/暴 等）→ 剔除
//   6. 任一位置为软惩罚字（病/疾/哀/愁 等），且非祈福豁免组合 → 剔除
//
// 用法: go run ./cmd/clean_curated <dataDir>
//  例: go run ./cmd/clean_curated ../data
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"name/internal/domain/fate"
)

// curatedEntry 对应 curated_names.json 的条目结构（仅保留清洗所需的字段与原文）
type curatedEntry struct {
	Name        string   `json:"name"`
	Pinyin      string   `json:"pinyin,omitempty"`
	Gender      string   `json:"gender,omitempty"`
	Source      string   `json:"source,omitempty"`
	Meaning     string   `json:"meaning,omitempty"`
	WuXing      string   `json:"wuxing,omitempty"`
	YinYunScore int      `json:"yinyun_score,omitempty"`
	Styles      []string `json:"styles,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// rejectReason 记录剔除原因分类统计
type rejectReason struct {
	label string
	names []string
}

// isHistoricalFigureEntry 判断策展条目是否为「历史人物字号/别名」采集来源
//
// 数据特征（curated_names.json 实测）：
//   - source = "古人云_历史人名"
//   - meaning = "历史人物字号/别名"
//   - tags 含 "历史人名"
//
// 历史人物字号（鲁章/鲁听/震男/骏男/男兆/渊明 等）是古人字号而非现代好名，
// 混入策展库会使荒谬组合命中 IsCuratedName 白名单 → 豁免四维封顶 → 高分霸榜。
// 因此只要来源/标签命中其一即剔除（数据级根治）。
func isHistoricalFigureEntry(e curatedEntry) bool {
	if strings.Contains(e.Source, "历史人名") {
		return true
	}
	if strings.Contains(e.Meaning, "历史人物字号") || strings.Contains(e.Meaning, "历史人物别名") {
		return true
	}
	for _, t := range e.Tags {
		if t == "历史人名" {
			return true
		}
	}
	return false
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "用法: go run ./cmd/clean_curated <dataDir>")
		os.Exit(1)
	}
	dataDir := os.Args[1]
	path := filepath.Join(dataDir, "curated_names.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取 %s 失败: %v\n", path, err)
		os.Exit(1)
	}
	var entries []curatedEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		fmt.Fprintf(os.Stderr, "解析 %s 失败: %v\n", path, err)
		os.Exit(1)
	}

	reasons := []*rejectReason{
		{label: "历史人物字号采集"},
		{label: "非双字名"},
		{label: "含语义门禁字"},
		{label: "历史人物字/号"},
		{label: "禁忌组合"},
		{label: "含硬禁用字"},
		{label: "含软惩罚字"},
	}
	rejectIndex := map[string]*rejectReason{}
	for _, r := range reasons {
		rejectIndex[r.label] = r
	}

	kept := make([]curatedEntry, 0, len(entries))
	for _, e := range entries {
		name := strings.TrimSpace(e.Name)
		runes := []rune(name)
		if len(runes) != 2 {
			rejectIndex["非双字名"].names = append(rejectIndex["非双字名"].names, name)
			continue
		}
		// 规则 0：按来源/标签剔除历史人物字号采集条目（数据级根治）
		if isHistoricalFigureEntry(e) {
			rejectIndex["历史人物字号采集"].names = append(rejectIndex["历史人物字号采集"].names, name)
			continue
		}
		c1, c2 := string(runes[0]), string(runes[1])
		switch {
		case fate.IsNonNamingChar(c1) || fate.IsNonNamingChar(c2):
			rejectIndex["含语义门禁字"].names = append(rejectIndex["含语义门禁字"].names, name)
			continue
		case fate.IsHistoricalFigureCombo(c1, c2):
			rejectIndex["历史人物字/号"].names = append(rejectIndex["历史人物字/号"].names, name)
			continue
		case fate.IsBadCombo(c1, c2):
			rejectIndex["禁忌组合"].names = append(rejectIndex["禁忌组合"].names, name)
			continue
		case fate.IsHardNegativeChar(c1) || fate.IsHardNegativeChar(c2):
			rejectIndex["含硬禁用字"].names = append(rejectIndex["含硬禁用字"].names, name)
			continue
		case (fate.IsSoftNegativeChar(c1) || fate.IsSoftNegativeChar(c2)) && !fate.IsBlessingCombo(c1, c2):
			rejectIndex["含软惩罚字"].names = append(rejectIndex["含软惩罚字"].names, name)
			continue
		}
		kept = append(kept, e)
	}

	fmt.Printf("原条目: %d\n", len(entries))
	fmt.Printf("保留条目: %d\n", len(kept))
	fmt.Printf("剔除条目: %d\n", len(entries)-len(kept))
	for _, r := range reasons {
		fmt.Printf("  - %s: %d\n", r.label, len(r.names))
	}
	// 展示每类前 10 条，便于人工核对规则是否合理
	for _, r := range reasons {
		if len(r.names) == 0 {
			continue
		}
		sample := r.names
		if len(sample) > 10 {
			sample = sample[:10]
		}
		fmt.Printf("  %s 样例: %s\n", r.label, strings.Join(sample, "、"))
	}

	// 写回清洗结果（覆盖原文件；git 可回滚）
	out, err := json.MarshalIndent(kept, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "序列化清洗结果失败: %v\n", err)
		os.Exit(1)
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "写回 %s 失败: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("已写回清洗结果 -> %s\n", path)
}
