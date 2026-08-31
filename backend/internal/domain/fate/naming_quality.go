package fate

// naming_quality.go 名字用字质量门禁
//
// 解决的问题：荒谬字（口语化/无实义/物名化字）在评分体系中拿到高分，
// 根因是 hanzi.json 的 isNegative/namePenalty 人工标注覆盖率不足。
//
// 设计原则 —— 质量门禁必须「宁缺毋滥」，只禁任何语境下都无命名价值的字：
//   - 虚词/助词/叹词：之/其/于/以/而/则/且/乃/焉/哉/兮/乎/也/矣 等，
//     在文言中承担语法功能，几乎无实义，作为名字没有寓意支撑。
//   - 排行字：伯/仲/叔/季（现代命名体系已弃用，单独入名显复古老旧）。
//   - 口语/物名/无实义字：伙/腕/膨/沽/混/冒/濒/懈/迄/熔/绰/秧 等，
//     现代口语联想消极（同伙/手腕/膨胀/沽名/冒失），无美好寓意。
//   - 数字/量词/方位口语字：二/四/五/六/七/八/十/半/个/只/块/片 等。
//   - 否定/疑问虚字：勿/毋/弗/否 等。
//
// 明确「不收录」的高频好名字（避免误伤）：
//   - 若（若曦/若水）、如（如初/如画）、然（浩然/亦然）、斯（斯年/斯文）、
//     唯/惟、一（一诺/一鸣）、三（三省/三思）、九（九思，语出《论语》）、
//     百（百川）、千（千帆）、万（万象）。这些字虽有虚词/数字属性，
//     但已形成稳定积极的命名寓意，必须放行。
//
// 注意：本门禁是「字级」硬剔除，与「组合级」检测（IsBadCombo/IsHistoricalFigureCombo）
// 正交互补；与评分阶段软惩罚（softNegativeChars/NamePenalty）分层协作：
//   - 门禁字：无法通过任何搭配挽救 → 候选池阶段直接剔除。
//   - 软惩罚字（病/疾等）：有「去病/弃疾」式祈福组合 → 保留入池，评分重罚。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// nonNamingChars 名字用字质量门禁表（候选池阶段剔除）
//
// 数据外置：表主体在 data/naming_quality.json（1094 字），
// 由 LoadNamingQualityFromJSON 启动时加载，本文件不再内嵌 325KB 字表。
var (
	nonNamingCharsMu sync.RWMutex
	nonNamingChars   = map[string]bool{}
)

// LoadNamingQualityFromJSON 从 data 目录加载命名质量门禁字表
func LoadNamingQualityFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "naming_quality.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取命名门禁字表失败: %w", err)
	}
	var chars []string
	if err := json.Unmarshal(data, &chars); err != nil {
		return fmt.Errorf("解析命名门禁字表失败: %w", err)
	}

	nonNamingCharsMu.Lock()
	defer nonNamingCharsMu.Unlock()
	nonNamingChars = make(map[string]bool, len(chars))
	for _, c := range chars {
		nonNamingChars[c] = true
	}
	return nil
}

// NonNamingCharCount 返回门禁字数（诊断/测试用）
func NonNamingCharCount() int {
	nonNamingCharsMu.RLock()
	defer nonNamingCharsMu.RUnlock()
	return len(nonNamingChars)
}

// IsNonNamingChar 判断单字是否为「不适合入名」的质量门禁字
// 用于候选池阶段硬剔除：伙/腕/膨/仲/之/六 等任何语境下都无命名价值的字
func IsNonNamingChar(char string) bool {
	if char == "" {
		return false
	}
	nonNamingCharsMu.RLock()
	defer nonNamingCharsMu.RUnlock()
	return nonNamingChars[char]
}