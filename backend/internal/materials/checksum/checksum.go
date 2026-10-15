// Package checksum 计算物料源文件的内容指纹
//
// 存在理由（docs/31 ADR-014）：实测 namer.json 的 version 字段是静态 "1.0"、
// generated 是 "2026-07-22"，内容变更时并不随之更新 —— 用 version 做 diff 判据
// 永远不会触发，热更新与「库过期自动导入」形同虚设。因此以内容 sha256 为唯一依据。
//
// 性能：大文件（如 word.json 26MB）每次全量 hash 约数十毫秒。
// 提供基于 size+mtime 的快速路径：命中则直接复用上次的指纹，
// 仅当 size/mtime 都变化时才重算，避免每次启动重复扫描全部物料。
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

// Stat 记录一次指纹计算的文件状态，供 Cache 判断快速路径是否可用
type Stat struct {
	Size    int64
	ModTime int64
	Sum     string
}

// StatOf 读取文件的 size 与修改时间（纳秒）
func StatOf(path string) (Stat, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return Stat{}, fmt.Errorf("读取文件状态失败 %s: %w", path, err)
	}
	return Stat{Size: fi.Size(), ModTime: fi.ModTime().UnixNano()}, nil
}

// Cache 带快速路径的指纹缓存：size+mtime 未变则复用历史结果
//
// 用途是装载门控（docs/31 ADR-014）：物料未变更时不重复灌库，
// 同时避免每次启动对数十 MB 的物料重算 sha256。
type Cache struct {
	entries map[string]Stat
}

// NewCache 创建空缓存
func NewCache() *Cache {
	return &Cache{entries: make(map[string]Stat)}
}

// Of 返回 path 的指纹，命中快速路径时不重复读文件
func (c *Cache) Of(path string) (string, error) {
	st, err := StatOf(path)
	if err != nil {
		return "", err
	}

	if old, ok := c.entries[path]; ok && old.Size == st.Size && old.ModTime == st.ModTime {
		return old.Sum, nil
	}

	sum, err := File(path)
	if err != nil {
		return "", err
	}
	c.entries[path] = Stat{Size: st.Size, ModTime: st.ModTime, Sum: sum}
	return sum, nil
}

// Forget 清掉某路径的缓存记录（强制下次重算）
func (c *Cache) Forget(path string) {
	delete(c.entries, path)
}
