// Package checksum 计算物料源文件的内容指纹
//
// 存在理由（docs/31 ADR-014）：实测 namer.json 的 version 字段是静态 "1.0"、
// generated 是 "2026-07-22"，内容变更时并不随之更新 —— 用 version 做 diff 判据
// 永远不会触发，热更新与「库过期自动导入」形同虚设。因此以内容 sha256 为唯一依据。
//
// 历史：曾提供基于 size+mtime 的 Cache 快速路径，docs/34 阶段 B1 评审后删除——
// 单次导入中每个源文件只计算一次指纹，缓存无收益；且 size+mtime 判定在
// 「等长修改 + mtime 未变」（git checkout / CI 复刻）场景下会漏判，属于
// 用正确性换不存在的性能。当前最大物料（kangxi CSV 6.4 万行）全量 hash
// 约数十毫秒，可接受。
package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// File 计算单个文件内容的 sha256 指纹
func File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开文件计算指纹失败 %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("读取文件计算指纹失败 %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Bytes 计算内存内容的 sha256 指纹
func Bytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
