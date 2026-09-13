// check_json 校验 data 目录 JSON 数据质量（可作 CI 步骤）
//
// 用法：在 backend 目录执行
//
//	go run ./cmd/check_json -data ./data
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type issue struct {
	File string
	Msg  string
}

func main() {
	dataDir := flag.String("data", "./data", "数据目录")
	flag.Parse()

	var issues []issue
	fail := func(file, format string, args ...any) {
		issues = append(issues, issue{File: file, Msg: fmt.Sprintf(format, args...)})
	}

	// --- yijing.json：64 卦、6 爻（乾坤 7）、象辞≠卦辞 ---
	yjPath := filepath.Join(*dataDir, "yijing.json")
	if data, err := os.ReadFile(yjPath); err != nil {
		fail("yijing.json", "读取失败: %v", err)
	} else {
		var list []map[string]any
		if err := json.Unmarshal(data, &list); err != nil {
			fail("yijing.json", "解析失败: %v", err)
		} else {
			if len(list) != 64 {
				fail("yijing.json", "卦数=%d，期望 64", len(list))
			}
			seen := map[float64]bool{}
			for _, h := range list {
				num, _ := h["number"].(float64)
				name, _ := h["name"].(string)
				gua, _ := h["gua_ci"].(string)
				xiang, _ := h["xiang_ci"].(string)
				yao, _ := h["yao_ci"].([]any)
				if num < 1 || num > 64 {
					fail("yijing.json", "卦 %s number 越界: %v", name, num)
				}
				if seen[num] {
					fail("yijing.json", "卦序 %v 重复", num)
				}
				seen[num] = true
				if gua == "" || xiang == "" || name == "" {
					fail("yijing.json", "卦 %v 存在空字段", num)
				}
				if strings.TrimSpace(gua) == strings.TrimSpace(xiang) {
					fail("yijing.json", "卦 %s 象辞与卦辞相同", name)
				}
				want := 6
				if num == 1 || num == 2 {
					want = 7
				}
				if len(yao) != want {
					fail("yijing.json", "卦 %s 爻辞 %d 条，期望 %d", name, len(yao), want)
				}
			}
		}
	}

	// --- namer.json：非空、含核心字段 ---
	namerPath := filepath.Join(*dataDir, "namer.json")
	if data, err := os.ReadFile(namerPath); err != nil {
		fail("namer.json", "读取失败: %v", err)
	} else {
		var arr []map[string]any
		if err := json.Unmarshal(data, &arr); err != nil {
			// 可能是对象包数组
			var obj map[string]any
			if err2 := json.Unmarshal(data, &obj); err2 != nil {
				fail("namer.json", "解析失败: %v", err)
			}
		} else if len(arr) == 0 {
			fail("namer.json", "条目为 0")
		}
	}

	// --- 经典 JSON：可解析且非空 ---
	classics := []string{
		"shijing.json", "chuci.json", "lunyu.json", "mengzi.json",
		"shici.json", "guwenguanzhi.json", "daxue.json", "zhongyong.json",
	}
	for _, fn := range classics {
		p := filepath.Join(*dataDir, fn)
		data, err := os.ReadFile(p)
		if err != nil {
			fail(fn, "读取失败: %v", err)
			continue
		}
		if len(data) == 0 {
			fail(fn, "文件为空")
			continue
		}
		var raw any
		if err := json.Unmarshal(data, &raw); err != nil {
			fail(fn, "JSON 解析失败: %v", err)
		}
	}

	// --- forbidden_combos.json ---
	fcPath := filepath.Join(*dataDir, "forbidden_combos.json")
	if data, err := os.ReadFile(fcPath); err != nil {
		fail("forbidden_combos.json", "读取失败: %v", err)
	} else {
		var arr []any
		if err := json.Unmarshal(data, &arr); err != nil {
			fail("forbidden_combos.json", "解析失败: %v", err)
		} else if len(arr) < 900 {
			fail("forbidden_combos.json", "条目 %d，期望约 962", len(arr))
		}
	}

	if len(issues) == 0 {
		fmt.Println("OK: 数据校验全部通过")
		return
	}
	fmt.Fprintf(os.Stderr, "FAIL: %d 项问题\n", len(issues))
	for _, is := range issues {
		fmt.Fprintf(os.Stderr, "  [%s] %s\n", is.File, is.Msg)
	}
	os.Exit(1)
}
