package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"name/internal/domain/classics"
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
	yjExport, n := mergeYijingCanonical()
	if n == 0 {
		fmt.Println("警告: 未找到 docs/玄学/易经 卦文档，yijing.json 不含彖传/小象传")
	}
	if err := writeJSON(filepath.Join(dataDir, "yijing.json"), yjExport); err != nil {
		fmt.Printf("导出易经数据失败: %v\n", err)
	} else {
		fmt.Printf("导出易经数据: %d 卦（其中 %d 卦合并了彖传/小象）\n", len(yjExport), n)
	}

	// 导出诗经数据
	if err := writeJSON(filepath.Join(dataDir, "shijing.json"), classics.ShijingNames); err != nil {
		fmt.Printf("导出诗经数据失败: %v\n", err)
	} else {
		fmt.Printf("导出诗经数据: %d 条\n", len(classics.ShijingNames))
	}

	// 导出楚辞数据
	if err := writeJSON(filepath.Join(dataDir, "chuci.json"), classics.ChuciNames); err != nil {
		fmt.Printf("导出楚辞数据失败: %v\n", err)
	} else {
		fmt.Printf("导出楚辞数据: %d 条\n", len(classics.ChuciNames))
	}

	// 导出生肖数据
	if err := writeJSON(filepath.Join(dataDir, "zodiac.json"), zodiac.ZodiacList); err != nil {
		fmt.Printf("导出生肖数据失败: %v\n", err)
	} else {
		fmt.Printf("导出生肖数据: %d 条\n", len(zodiac.ZodiacList))
	}

	fmt.Println("数据导出完成")
}

// mergeYijingCanonical 合并 docs/玄学/易经 语料解析出的彖传/小象传到内存卦表。
// 文档目录基于本源文件位置向上查找（与数据目录同一套探测思路）。
// 返回导出副本与成功合并的卦数；文档缺失时返回原列表副本与 0（不阻塞导出）。
func mergeYijingCanonical() ([]yijing.Hexagram, int) {
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

	out := make([]yijing.Hexagram, 0, len(yijing.HexagramList))
	for _, h := range yijing.HexagramList {
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
	return out, merged
}
