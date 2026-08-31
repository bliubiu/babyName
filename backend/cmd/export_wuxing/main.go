package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"name/internal/domain/hanzi"
	"name/internal/infrastructure/data"
)

type WuxingEntry struct {
	Char     string
	Pinyin   string
	Radical  string
	Strokes  int
	Wuxing   string
	Source   string // 字义法特例 / 部首映射 / JSON兜底
	Meaning  string
	UsageLv  int
}

func main() {
	dataDir := "data"

	// 加载数据（自动应用五行映射）
	if err := data.Init(dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "加载数据失败: %v\n", err)
		os.Exit(1)
	}

	// 按五行分组
	groups := map[string][]WuxingEntry{}
	wuxingOrder := []string{"金", "木", "水", "火", "土"}

	for _, h := range hanzi.HanziData {
		source := inferSource(h.Char, h.Radical, h.Wuxing)
		wx := h.Wuxing
		if wx == "" {
			wx = "未分配"
		}
		groups[wx] = append(groups[wx], WuxingEntry{
			Char:    h.Char,
			Pinyin:  h.Pinyin,
			Radical: h.Radical,
			Strokes: h.Strokes,
			Wuxing:  wx,
			Source:  source,
			Meaning: truncate(h.Meaning, 8),
			UsageLv: h.UsageLevel,
		})
	}

	// 按笔画排序（同一部内）
	for wx := range groups {
		sort.Slice(groups[wx], func(i, j int) bool {
			if groups[wx][i].Strokes != groups[wx][j].Strokes {
				return groups[wx][i].Strokes < groups[wx][j].Strokes
			}
			return groups[wx][i].Char < groups[wx][j].Char
		})
	}

	// 输出目录
	outDir := filepath.Join(dataDir, "wuxing_export")
	os.MkdirAll(outDir, 0755)

	// 1. 每个五行一个文件
	for _, wx := range wuxingOrder {
		entries := groups[wx]
		path := filepath.Join(outDir, fmt.Sprintf("wuxing_%s.txt", wx))
		writeCategoryFile(path, wx, entries)
	}

	// 2. 未分配
	if unassigned, ok := groups["未分配"]; ok && len(unassigned) > 0 {
		path := filepath.Join(outDir, "wuxing_未分配.txt")
		writeCategoryFile(path, "未分配", unassigned)
	}

	// 3. 统计文件
	writeSummaryFile(outDir, groups, wuxingOrder, len(hanzi.HanziData))

	fmt.Println("✅ 五行分类导出完成！")
	fmt.Printf("   输出目录: %s\n", outDir)
	for _, wx := range wuxingOrder {
		fmt.Printf("   %s: %d 字\n", wx, len(groups[wx]))
	}
}

// inferSource 判断该字的五行来源
func inferSource(char, radical, wuxing string) string {
	if _, ok := hanzi.CharacterWuxingOverride[char]; ok {
		return "字义法特例"
	}
	if radWx := hanzi.GetWuxingByRadical(radical); radWx != "" {
		return "部首映射"
	}
	return "JSON兜底"
}

func writeCategoryFile(path, wx string, entries []WuxingEntry) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建文件失败 %s: %v\n", path, err)
		return
	}
	defer f.Close()

	// 统计来源分布
	srcCount := map[string]int{}
	lvCount := map[int]int{}
	for _, e := range entries {
		srcCount[e.Source]++
		lvCount[e.UsageLv]++
	}

	levelName := map[int]string{
		1: "一级常用", 2: "二级通用", 3: "三级专用",
		4: "四级", 5: "五级",
	}

	// 标题
	fmt.Fprintf(f, "五行：%s（共 %d 字）\n", wx, len(entries))
	fmt.Fprintln(f, strings.Repeat("━", 60))
	fmt.Fprintln(f)
	fmt.Fprintln(f, "【来源分布】")
	for _, s := range []string{"字义法特例", "部首映射", "JSON兜底"} {
		if n := srcCount[s]; n > 0 {
			fmt.Fprintf(f, "  %s: %d 字 (%.1f%%)\n", s, n, float64(n)/float64(len(entries))*100)
		}
	}
	fmt.Fprintln(f)
	fmt.Fprintln(f, "【等级分布】")
	for lv := 1; lv <= 3; lv++ {
		if n := lvCount[lv]; n > 0 {
			fmt.Fprintf(f, "  %s: %d 字\n", levelName[lv], n)
		}
	}
	fmt.Fprintln(f)
	fmt.Fprintln(f, "【汉字列表】")
	fmt.Fprintf(f, "%-4s  %-10s  %-4s  %-4s  %-10s  %-4s  %s\n",
		"字", "拼音", "部首", "笔画", "来源", "等级", "含义")
	fmt.Fprintln(f, strings.Repeat("─", 60))

	levelShort := map[int]string{1: "一级", 2: "二级", 3: "三级", 4: "四级", 5: "五级"}

	for _, e := range entries {
		usage := levelShort[e.UsageLv]
		if e.UsageLv == 0 {
			usage = ""
		}
		fmt.Fprintf(f, "%-4s  %-10s  %-4s  %-4d  %-10s  %-4s  %s\n",
			e.Char, e.Pinyin, e.Radical, e.Strokes, e.Source, usage, e.Meaning)
	}

	fmt.Fprintf(f, "\n━━━━━━━━━━ 共 %d 字 ━━━━━━━━━━\n", len(entries))
}

func writeSummaryFile(outDir string, groups map[string][]WuxingEntry, wuxingOrder []string, total int) {
	path := filepath.Join(outDir, "wuxing_summary.txt")
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建统计文件失败: %v\n", err)
		return
	}
	defer f.Close()

	fmt.Fprintln(f, "五行分类统计汇总")
	fmt.Fprintln(f, strings.Repeat("━", 40))
	fmt.Fprintf(f, "\n总汉字数: %d\n", total)
	fmt.Fprintln(f)
	fmt.Fprintf(f, "%-6s  %8s  %8s  %8s  %8s  %8s\n",
		"五行", "字数", "占比", "特例数", "映射数", "兜底数")
	fmt.Fprintln(f, strings.Repeat("─", 60))

	for _, wx := range wuxingOrder {
		entries := groups[wx]
		pct := float64(len(entries)) / float64(total) * 100
		srcCount := map[string]int{}
		for _, e := range entries {
			srcCount[e.Source]++
		}
		fmt.Fprintf(f, "%-6s  %8d  %7.1f%%  %8d  %8d  %8d\n",
			wx, len(entries), pct,
			srcCount["字义法特例"],
			srcCount["部首映射"],
			srcCount["JSON兜底"],
		)
	}

	if ua, ok := groups["未分配"]; ok && len(ua) > 0 {
		fmt.Fprintf(f, "\n未分配: %d 字\n", len(ua))
	}

	// 按部首统计每部首的五行分布
	fmt.Fprintln(f)
	fmt.Fprintln(f, strings.Repeat("━", 60))
	fmt.Fprintln(f, "部首五行分布（按部首统计）")
	fmt.Fprintln(f)
	fmt.Fprintf(f, "%-6s  %-10s  %-8s  %s\n", "部首", "五行", "字数", "代表字（前5）")
	fmt.Fprintln(f, strings.Repeat("─", 60))

	type RadStat struct {
		wx    string
		count int
		chars []string
	}
	radStats := map[string]*RadStat{}

	for _, entries := range groups {
		for _, e := range entries {
			if _, ok := radStats[e.Radical]; !ok {
				radStats[e.Radical] = &RadStat{wx: e.Wuxing}
			}
			rs := radStats[e.Radical]
			rs.count++
			if len(rs.chars) < 5 {
				rs.chars = append(rs.chars, e.Char)
			}
		}
	}

	// 按部首排序
	rads := make([]string, 0, len(radStats))
	for r := range radStats {
		rads = append(rads, r)
	}
	sort.Strings(rads)

	for _, r := range rads {
		rs := radStats[r]
		fmt.Fprintf(f, "%-6s  %-10s  %-8d  %s\n",
			r, rs.wx, rs.count, strings.Join(rs.chars, " "))
	}
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
