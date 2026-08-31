package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"name/internal/domain/classics"
	"name/internal/domain/hanzi"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/data"
)

func writeJSON(path string, data interface{}) error {
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)

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
	dataDir := "data"

	// 从 JSON 加载数据（硬编码数据已移除，需先加载再导出）
	if err := data.Init(dataDir); err != nil {
		fmt.Printf("加载数据失败: %v\n", err)
		os.Exit(1)
	}

	// 导出汉字数据（写入生成原料目录 raw/，与运行时数据隔离）
	rawDir := filepath.Join(dataDir, "raw")
	os.MkdirAll(rawDir, 0755)
	hanziList := make([]hanzi.Hanzi, 0, len(hanzi.HanziData))
	for _, h := range hanzi.HanziData {
		hanziList = append(hanziList, h)
	}
	if err := writeJSON(filepath.Join(rawDir, "hanzi.json"), hanziList); err != nil {
		fmt.Printf("导出汉字数据失败: %v\n", err)
	} else {
		fmt.Printf("导出汉字数据: %d 条\n", len(hanziList))
	}

	// 导出易经64卦
	if err := writeJSON(filepath.Join(dataDir, "yijing.json"), yijing.HexagramList); err != nil {
		fmt.Printf("导出易经数据失败: %v\n", err)
	} else {
		fmt.Printf("导出易经数据: %d 卦\n", len(yijing.HexagramList))
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
