// diag_curated 临时诊断程序：验证策展字加分在 verify_fate 链路的各个环节
//
// 用法: go run ./cmd/diag_curated ./data
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"name/internal/application/services"
	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/infrastructure/data"
)

func main() {
	dataDir := "."
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}
	absDir, _ := filepath.Abs(dataDir)
	if err := data.Init(absDir); err != nil {
		fmt.Fprintf(os.Stderr, "加载文化数据失败: %v\n", err)
		os.Exit(1)
	}
	if err := hanzi.LoadWordData(absDir); err != nil {
		fmt.Fprintf(os.Stderr, "加载 word.json 失败(忽略): %v\n", err)
	}
	services.SyncNamingIndexFromHanzi()

	// 1. 直接验证 IsCuratedNamingChar
	fmt.Println("=== 1. hanzi.IsCuratedNamingChar ===")
	for _, ch := range []string{"毅", "辉", "涛", "英", "琳", "雅", "梅", "强", "凯", "软", "际", "映", "耿", "宝", "念", "畅", "章", "好", "典", "贪", "疟", "骂"} {
		fmt.Printf("  %s → %v\n", ch, hanzi.IsCuratedNamingChar(ch))
	}

	// 2. 通过 HanziDataProvider 取字，验证 IsCurated 字段
	fmt.Println("=== 2. HanziDataProvider.GetCharacter.IsCurated ===")
	provider := &services.HanziDataProvider{}
	for _, ch := range []string{"毅", "辉", "涛", "软", "际", "映", "耿", "宝", "念", "畅", "章", "好", "典"} {
		c, err := provider.GetCharacter(ch)
		if err != nil {
			fmt.Printf("  %s → err=%v\n", ch, err)
			continue
		}
		fmt.Printf("  %s → IsCurated=%v CommonLevel=%d IsRegular=%v\n", ch, c.IsCurated, c.CommonLevel, c.IsRegular)
	}

	// 3. WenHuaRater 对策展/非策展字的实际文化分
	fmt.Println("=== 3. WenHuaRater 单名文化分 ===")
	rater := fate.NewWenHuaRaterWithWeight(0.20)
	for _, ch := range []string{"毅", "辉", "涛", "软", "际", "映", "耿", "宝", "念", "畅", "章", "好", "典"} {
		c, err := provider.GetCharacter(ch)
		if err != nil {
			continue
		}
		cand := &fate.NameCandidate{
			Char1:        c.Char,
			IsRegular:    c.IsRegular,
			CommonLevel1: c.CommonLevel,
			IsCurated1:   c.IsCurated,
			Meaning1:     c.Meaning,
			Stroke1:      c.SimplifiedStroke,
			HasPoetry:    true, // 跳过诗词数据依赖，聚焦策展差异
		}
		r := rater.Rate(cand, nil)
		fmt.Printf("  %s curated=%v wh=%.0f | %s\n", ch, c.IsCurated, r.Score, r.Detail)
	}
}
