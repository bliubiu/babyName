package zodiac

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

var zoMu sync.RWMutex

// LoadFromJSON 从 JSON 文件加载生肖数据
func LoadFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "zodiac.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var list []Zodiac
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	zoMu.Lock()
	defer zoMu.Unlock()

	ZodiacList = list
	return nil
}

// ReloadFromJSON 重新加载生肖数据（热更新）
func ReloadFromJSON(dataDir string) error {
	return LoadFromJSON(dataDir)
}

// SetZodiacs 直接以内存结构设置生肖列表（SQLite 物料装载路径，ADR-008）
func SetZodiacs(list []Zodiac) {
	zoMu.Lock()
	defer zoMu.Unlock()
	ZodiacList = list
}

// GetZodiac 线程安全获取生肖数据
func GetZodiac(name string) (Zodiac, bool) {
	zoMu.RLock()
	defer zoMu.RUnlock()
	for _, z := range ZodiacList {
		if z.Name == name {
			return z, true
		}
	}
	return Zodiac{}, false
}
