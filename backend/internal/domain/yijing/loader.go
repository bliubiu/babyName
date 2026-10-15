package yijing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

var yjMu sync.RWMutex

// LoadFromJSON 从 JSON 文件加载易经数据
func LoadFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "yijing.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var list []Hexagram
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	yjMu.Lock()
	defer yjMu.Unlock()

	HexagramList = list
	return nil
}

// ReloadFromJSON 重新加载易经数据（热更新）
func ReloadFromJSON(dataDir string) error {
	return LoadFromJSON(dataDir)
}

// SetHexagrams 直接以内存结构设置卦象列表（SQLite 物料装载路径，ADR-008）
//
// 与 LoadFromJSON 互为两条来源：运行时统一走 SQLite 后不再解析 JSON。
func SetHexagrams(list []Hexagram) {
	yjMu.Lock()
	defer yjMu.Unlock()
	HexagramList = list
}

// GetHexagram 线程安全获取卦象数据
func GetHexagram(id int) (Hexagram, bool) {
	yjMu.RLock()
	defer yjMu.RUnlock()
	for _, h := range HexagramList {
		if h.ID == id {
			return h, true
		}
	}
	return Hexagram{}, false
}
