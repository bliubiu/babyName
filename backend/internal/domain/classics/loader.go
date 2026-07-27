package classics

import (
	"fmt"

	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// LoadFromJSON 从 JSON 文件加载诗词数据
// P0 经典同步加载，shici.json（6.7MB）异步加载不阻塞启动
// P2/P3 经典同步加载
func LoadFromJSON(dataDir string) error {
	// ===== P0：已实现的核心文本 =====

	if err := loadShiJingFromJSON(dataDir); err != nil {
		return fmt.Errorf("诗经加载失败: %w", err)
	}
	if err := loadChuCiFromJSON(dataDir); err != nil {
		return fmt.Errorf("楚辞加载失败: %w", err)
	}
	if err := loadGuWenFromJSON(dataDir); err != nil {
		logger.Warn("古文观止加载跳过", zap.Error(err))
	}

	// shici.json（6.7MB）→ 后台异步加载，不阻塞 HTTP 启动
	loadShiCiAsync(dataDir)

	// ===== P2：儒家经典 =====

	loadClassicSafe(dataDir, "lunyu.json", "论语", &LunyuExtracted)
	loadClassicSafe(dataDir, "mengzi.json", "孟子", &MengziExtracted)
	loadClassicSafe(dataDir, "daxue.json", "大学", &DaxueExtracted)
	loadClassicSafe(dataDir, "zhongyong.json", "中庸", &ZhongyongExtracted)

	// ===== P3：蒙学经典 =====

	loadClassicSafe(dataDir, "sanzijing-new.json", "三字经", &SanzijingExtracted)
	loadClassicSafe(dataDir, "qianziwen.json", "千字文", &QianziwenExtracted)
	loadClassicSafe(dataDir, "dizigui.json", "弟子规", &DiziguiExtracted)
	loadClassicSafe(dataDir, "youxueqionglin.json", "幼学琼林", &YouxueqionglinExtracted)
	loadClassicSafe(dataDir, "zengguangxianwen.json", "增广贤文", &ZengguangxianwenExtracted)
	loadClassicSafe(dataDir, "shenglvqimeng.json", "声律启蒙", &ShenglvqimengExtracted)
	loadClassicSafe(dataDir, "zhuzijiaxun.json", "朱子家训", &ZhuzijiaxunExtracted)
	loadClassicSafe(dataDir, "qianjiashi.json", "千家诗", &QianjiashiExtracted)
	loadClassicSafe(dataDir, "wenzimengqiu.json", "文字蒙求", &WenzimengqiuExtracted)
	loadClassicSafe(dataDir, "baijiaxing.json", "百家姓", &BaijiaxingExtracted)

	return nil
}

// loadClassicSafe 安全加载经典文件，失败仅打印警告不中断流程
func loadClassicSafe(dataDir, filename, name string, target *[]PoetryChar) {
	if err := loadClassicFromJSON(dataDir, filename, name, target); err != nil {
		logger.Warn("经典加载跳过", zap.String("name", name), zap.Error(err))
	}
}

// ReloadFromJSON 重新加载诗词数据（热更新）
func ReloadFromJSON(dataDir string) error {
	return LoadFromJSON(dataDir)
}

// ============================================================
// Getter 函数——外部调用时触发惰性等待
// ============================================================

func GetShijingPoetryChars() []PoetryChar {
	result := make([]PoetryChar, len(ShijingExtracted))
	copy(result, ShijingExtracted)
	return result
}

func GetChuciPoetryChars() []PoetryChar {
	result := make([]PoetryChar, len(ChuciExtracted))
	copy(result, ChuciExtracted)
	return result
}

func GetGuwenGuanzhiPoetryChars() []PoetryChar {
	result := make([]PoetryChar, len(GuwenGuanzhiExtracted))
	copy(result, GuwenGuanzhiExtracted)
	return result
}

func GetShiCiPoetryChars() []PoetryChar {
	ensureShiCiLoaded()
	result := make([]PoetryChar, len(ShiCiExtracted))
	copy(result, ShiCiExtracted)
	return result
}

func GetLunyuPoetryChars() []PoetryChar {
	result := make([]PoetryChar, len(LunyuExtracted))
	copy(result, LunyuExtracted)
	return result
}
