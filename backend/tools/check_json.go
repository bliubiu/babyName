//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	base := filepath.Join("..", "data")
	files := []string{
		"shici.json",
		"cifu.json",
		"yuanqu.json",
		"yuefu.json",
		"guwenguanzhi.json",
		"lunyu.json",
		"mengzi.json",
		"daxue.json",
		"zhongyong.json",
		"sanzijing-new.json",
		"qianziwen.json",
		"dizigui.json",
		"youxueqionglin.json",
		"zengguangxianwen.json",
		"qianjiashi.json",
		"wenzimengqiu.json",
		"zhuzijiaxun.json",
	}

	for _, fn := range files {
		path := filepath.Join(base, fn)
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("❌ %s: %v\n", fn, err)
			continue
		}
		fmt.Printf("\n====== %s (%dKB) ======\n", fn, len(data)/1024)

		raw := string(data)
		if len(raw) > 0 {
			firstChar := raw[0]
			if firstChar == '{' {
				// 对象
				var obj map[string]interface{}
				if err := json.Unmarshal(data, &obj); err != nil {
					fmt.Printf("  JSON解析失败: %v\n", err)
					continue
				}
				fmt.Printf("  类型: JSON对象\n")
				fmt.Printf("  顶层键: ")
				keys := make([]string, 0, len(obj))
				for k := range obj {
					keys = append(keys, k)
				}
				fmt.Println(keys)
				// 检查 content 字段结构
				if content, ok := obj["content"]; ok {
					switch v := content.(type) {
					case []interface{}:
						fmt.Printf("  content: 数组, 长度=%d\n", len(v))
						if len(v) > 0 {
							first := v[0]
							switch f := first.(type) {
							case map[string]interface{}:
								fmt.Printf("  content[0]键: ")
								cks := make([]string, 0, len(f))
								for k := range f {
									cks = append(cks, k)
								}
								fmt.Println(cks)
								if paras, ok := f["paragraphs"]; ok {
									switch p := paras.(type) {
									case []interface{}:
										fmt.Printf("  content[0].paragraphs: 数组, 长度=%d\n", len(p))
										if len(p) > 0 {
											fmt.Printf("  示例行: %q\n", p[0])
										}
									}
								}
							}
						}
					}
				}
			} else if firstChar == '[' {
				fmt.Printf("  类型: JSON数组\n")
				var arr []interface{}
				if err := json.Unmarshal(data, &arr); err != nil {
					fmt.Printf("  JSON解析失败: %v\n", err)
					continue
				}
				fmt.Printf("  数组长度: %d\n", len(arr))
				if len(arr) > 0 {
					switch f := arr[0].(type) {
					case map[string]interface{}:
						fmt.Printf("  [0]键: ")
						ks := make([]string, 0, len(f))
						for k := range f {
							ks = append(ks, k)
						}
						fmt.Println(ks)
					}
				}
			}
		}
	}
}
