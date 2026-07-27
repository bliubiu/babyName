// Package data 统一数据加载器
//
// 统筹管理所有域包的数据加载流程，提供统一的启动加载、热更新和缓存访问。
// 各域包保留自己的数据结构（如 HanziData, HexagramList, ZodiacList 等），
// 但加载过程统一由本包协调。
package data

import (
	"fmt"
	"sync"

	"name/internal/domain/classics"
	"name/internal/domain/hanzi"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
)

// 全局缓存：文件名 → 原始字节（支持热更新比对）
type fileCache struct {
	mu   sync.RWMutex
	data map[string][]byte
}

var cache = &fileCache{data: make(map[string][]byte)}

// Init 统一加载所有JSON数据文件
// 按依赖顺序加载：hanzi → yijing → classic → zodiac
func Init(dataDir string) error {
	// 1. 加载《通用规范汉字表》8105字数据（namer.json — 主数据源）
	//
	// namer.json 为统一的起名用字数据，含五行校正和起名分类标注，覆盖全部 8105 标准字。
	// hanzi.json（14814 字）中的非标准字已不在启动时加载，以符合项目以标准字表为准的要求。
	// namer.json 为统一的起名用字数据，含五行校正和起名分类标注。
	// 通过 namer_loader.go 同步到 HanziData，使所有依赖 HanziData 的代码自动受益。
	if err := hanzi.LoadNamerFromJSON(dataDir); err != nil {
		return fmt.Errorf("加载 namer 数据失败: %w", err)
	}

	// 2. 加载易经数据
	if err := yijing.LoadFromJSON(dataDir); err != nil {
		return fmt.Errorf("加载易经数据失败: %w", err)
	}

	// 3. 加载诗词数据
	if err := classics.LoadFromJSON(dataDir); err != nil {
		return fmt.Errorf("加载诗词数据失败: %w", err)
	}

	// 4. 加载生肖数据
	if err := zodiac.LoadFromJSON(dataDir); err != nil {
		return fmt.Errorf("加载生肖数据失败: %w", err)
	}

	return nil
}

// ReloadAll 热更新所有数据（重新加载并通知观察者）
func ReloadAll(dataDir string) error {
	// 将来可扩展：比对文件变更时间，只重载有变化的文件
	return Init(dataDir)
}
