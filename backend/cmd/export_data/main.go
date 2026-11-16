package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"name/internal/domain/hanzi"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/data"
	"name/internal/paths"
)

func writeJSON(path string, data interface{}) error {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func main() {
	// 在 data.Init 之前快照代码内置卦表：yijing.json 必须以内置表（docs 金标准
	// 对拍对象）为真源。data.Init 会用待导出的旧 JSON 覆盖 HexagramList，
	// 若拿覆盖后的表回写，历史遗留错误（如 23 卦 symbol 下卦在前）会自我复制。
	builtinHexagrams := make([]yijing.Hexagram, len(yijing.HexagramList))
	copy(builtinHexagrams, yijing.HexagramList)

	// 数据目录：基于本源文件位置向上探测仓库根 data/（与 cmd/server、namer-cli 同一套
	// paths.FindDataDir），探测无果回落 ./data 保留历史语义
	_, thisFile, _, _ := runtime.Caller(0)
	dataDir := paths.FindDataDir(thisFile)
	if dataDir == "" {
		dataDir = "data"
	}

	// 从 JSON 加载数据（硬编码数据已移除，需先加载再导出）
	if err := data.Init(dataDir); err != nil {
		fmt.Printf("加载数据失败: %v\n", err)
		os.Exit(1)
	}

	// 导出汉字数据（写入生成原料目录 raw/，与运行时数据隔离）
	rawDir := filepath.Join(dataDir, "raw")
	_ = os.MkdirAll(rawDir, 0755)
	hanziList := make([]hanzi.Hanzi, 0, len(hanzi.HanziData))
	for _, h := range hanzi.HanziData {
		hanziList = append(hanziList, h)
	}
	if err := writeJSON(filepath.Join(rawDir, "hanzi.json"), hanziList); err != nil {
		fmt.Printf("导出汉字数据失败: %v\n", err)
	} else {
		fmt.Printf("导出汉字数据: %d 条\n", len(hanziList))
	}

	// 导出易经64卦：合并 docs/玄学/易经 语料解析出的彖传/小象传
	//（金标准文本以文档为单一真源，运行库经 materials import 从本 JSON 装载）
	yjExport, n := mergeYijingCanonical(builtinHexagrams)
	if n == 0 {
		fmt.Println("警告: 未找到 docs/玄学/易经 卦文档，yijing.json 不含彖传/小象传")
	}
	if err := writeJSON(filepath.Join(dataDir, "yijing.json"), yjExport); err != nil {
		fmt.Printf("导出易经数据失败: %v\n", err)
	} else {
		fmt.Printf("导出易经数据: %d 卦（其中 %d 卦合并了彖传/小象）\n", len(yjExport), n)
	}

	// 诗经/楚辞语料为 chinese-poetry 上游导入件（docs/12、CHANGELOG 2026.11.01.0），
	// 不是本命令的派生产物：严禁在此回写。曾两次因「data.Init 未灌数 → 回写 null」
	// 覆空 shijing.json/chuci.json（156KB/141KB 语料丢失，golden 快照漂移），
	// 现整段移除回写逻辑；上游更新请直接从 chinese-poetry 仓库重新下载覆盖。

	// 导出生肖数据
	if err := writeJSON(filepath.Join(dataDir, "zodiac.json"), zodiac.ZodiacList); err != nil {
		fmt.Printf("导出生肖数据失败: %v\n", err)
	} else {
		fmt.Printf("导出生肖数据: %d 条\n", len(zodiac.ZodiacList))
	}

	fmt.Println("数据导出完成")
}

// mergeYijingCanonical 把 docs/玄学/易经 解析出的彖传/小象传合并到传入的
// 内置卦表快照，生成 yijing.json 的导出副本。
//
// 只合并 TuanCi / XiaoXiang（内置表这两字段为空）；卦辞、象辞、爻辞、符号、
// 上下卦一律沿用内置表。这样即使源 yijing.json 带历史错误，重跑导出也会
// 自动收敛到金标准，不会把旧快照原样写回。
//
// 文档目录基于本源文件位置向上查找（与数据目录同一套探测思路）。
// 返回导出副本与成功合并的卦数；文档缺失时返回原列表副本与 0（不阻塞导出）。
func mergeYijingCanonical(builtin []yijing.Hexagram) ([]yijing.Hexagram, int) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile)
	var docsDir string
	for i := 0; i < 12; i++ {
		cand := filepath.Join(dir, "docs", "玄学", "易经")
		if fi, err := os.Stat(cand); err == nil && fi.IsDir() {
			docsDir = cand
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	out := make([]yijing.Hexagram, 0, len(builtin))
	for _, h := range builtin {
		out = append(out, h)
	}
	if docsDir == "" {
		return out, 0
	}

	docs, err := yijing.ParseCanonicalDocs(docsDir)
	if err != nil {
		fmt.Printf("解析易经语料失败: %v\n", err)
		return out, 0
	}
	byNum := make(map[int]yijing.CanonicalDoc, len(docs))
	for _, d := range docs {
		byNum[d.Number] = d
	}

	merged := 0
	for i := range out {
		d, ok := byNum[out[i].Number]
		if !ok {
			continue
		}
		out[i].TuanCi = d.TuanCi
		out[i].XiaoXiang = d.XiaoXiang
		merged++
	}
	// 卦德随导出固化（docs/42 P1-⑤收尾）：硬编码表 init 已补算，
	// 此处对副本再兜底一次，保证 yijing.json 始终携带 gua_de。
	yijing.EnsureGuaDe(out)
	return out, merged
}
