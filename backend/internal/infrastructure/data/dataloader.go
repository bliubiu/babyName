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
	"name/internal/domain/fate"
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
// 按依赖顺序加载：namer → yijing → classic → zodiac
func Init(dataDir string) error {
	// 1. 加载《通用规范汉字表》8105字数据（namer.json — 运行时唯一汉字数据源）
	//
	// namer.json 为统一的起名用字数据，含五行校正、起名分类标注，
	// 以及并入的精选偏旁分组（charGroups），覆盖全部 8105 标准字。
	// 通过 namer_loader.go 同步到 HanziData，使所有依赖 HanziData 的代码自动受益。
	// 历史遗留的 hanzi.json / standard_chars.json 已降级为 export_namer 的
	// 离线生成原料，不再于运行时加载。
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

	// 5. 加载命名质量门禁字表（nonNamingChars，候选池硬剔除）
	if err := fate.LoadNamingQualityFromJSON(dataDir); err != nil {
		return fmt.Errorf("加载命名质量门禁字表失败: %w", err)
	}

	return nil
}

// ReloadAll 热更新所有数据（重新加载并通知观察者）
func ReloadAll(dataDir string) error {
	// 将来可扩展：比对文件变更时间，只重载有变化的文件
	return Init(dataDir)
}
