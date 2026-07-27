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
