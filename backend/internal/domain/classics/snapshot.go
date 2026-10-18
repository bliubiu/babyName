package classics

import (
	"fmt"
	"sort"
	"strings"
)

// 本文件定义经典物料的「导入期物化 / 运行期还原」数据契约。
//
// 背景（docs/31 ADR-008）：运行时统一只读 SQLite。经典字集（*Extracted）与
// 全局诗词索引（GlobalPoemIndex）都是**由源 JSON 派生的计算产物**，不是源数据
// 本身。若运行期改为「从 classics_* 三表重新提取」，就必须把 18 个文件的提取
// 规则在第二处重新实现一遍，两套实现必然漂移（文件集合、source 语义、
// baijiaxing/qianziwen 的 spells 处理等各不相同）。
//
// 因此改为：导入期用**与 JSON 路径完全相同的提取逻辑**算出结果并落库，
// 运行期逐行还原。提取规则只有 extractFromEntries 一份实现，
// 物化是纯粹的搬运，保真度由构造保证而非由测试比对保证。

// extractMaterialNames 经典字集的规范物料标识（按装载顺序）
//
// 标识取 JSON 文件名去后缀，与 classicsFileCategory 中的文件名一一对应。
const (
	MaterialShijing          = "shijing"
	MaterialChuci            = "chuci"
	MaterialGuwenguanzhi     = "guwenguanzhi"
	MaterialShici            = "shici"
	MaterialLunyu            = "lunyu"
	MaterialMengzi           = "mengzi"
	MaterialDaxue            = "daxue"
	MaterialZhongyong        = "zhongyong"
	MaterialSanzijing        = "sanzijing-new"
	MaterialQianziwen        = "qianziwen"
	MaterialDizigui          = "dizigui"
	MaterialYouxueqionglin   = "youxueqionglin"
	MaterialZengguangxianwen = "zengguangxianwen"
	MaterialShenglvqimeng    = "shenglvqimeng"
	MaterialZhuzijiaxun      = "zhuzijiaxun"
	MaterialQianjiashi       = "qianjiashi"
	MaterialWenzimengqiu     = "wenzimengqiu"
	MaterialBaijiaxing       = "baijiaxing"
)

// extractTargets 字集物料标识 → 包级结果变量的对应关系（唯一映射真源）
//
// 顺序即 LoadFromJSON 的装载顺序，导出与还原共用，保证两侧一一对应。
var extractTargets = []struct {
	material string
	target   *[]PoetryChar
}{
	{MaterialShijing, &ShijingExtracted},
	{MaterialChuci, &ChuciExtracted},
	{MaterialGuwenguanzhi, &GuwenGuanzhiExtracted},
	{MaterialShici, &ShiCiExtracted},
	{MaterialLunyu, &LunyuExtracted},
	{MaterialMengzi, &MengziExtracted},
	{MaterialDaxue, &DaxueExtracted},
	{MaterialZhongyong, &ZhongyongExtracted},
	{MaterialSanzijing, &SanzijingExtracted},
	{MaterialQianziwen, &QianziwenExtracted},
	{MaterialDizigui, &DiziguiExtracted},
	{MaterialYouxueqionglin, &YouxueqionglinExtracted},
	{MaterialZengguangxianwen, &ZengguangxianwenExtracted},
	{MaterialShenglvqimeng, &ShenglvqimengExtracted},
	{MaterialZhuzijiaxun, &ZhuzijiaxunExtracted},
	{MaterialQianjiashi, &QianjiashiExtracted},
	{MaterialWenzimengqiu, &WenzimengqiuExtracted},
	{MaterialBaijiaxing, &BaijiaxingExtracted},
}

// ExtractSet 一个经典字集（导入期物化的最小单元）
type ExtractSet struct {
	// Material 字集物料标识（见 MaterialShijing 等常量）
	Material string
	// Chars 提取出的起名用字，按汉字码点升序
	Chars []PoetryChar
}

// ExtractRow classics_extracts 表的一行（物化后的单字诗词条目）
type ExtractRow struct {
	Material string
	Char     string
	Pinyin   string
	Meaning  string
	Work     string
	Chapter  string
	Sentence string
	Wuxing   string
	Gender   string
}

// PoemEntryRow classics_poem_entries 表的一行（物化后的整首诗词）
//
// 不落 Chars / FullText：两者都是 Content 的纯函数（见 LoadPoemEntries），
// 落库会引入可与 Content 不一致的冗余副本。
type PoemEntryRow struct {
	Source   string
	Title    string
	Author   string
	Dynasty  string
	Type     string
	Category string
	Book     string
	Tags     []string
	Content  []string
}

// ExportExtractSets 导出全部经典字集（导入期物化用）
//
// 调用前需已完成装载（LoadFromJSON）。内部对 shici 走有界等待，确保
// 后台异步加载的 6.7MB 数据已就绪再取，避免物化出一个空字集。
func ExportExtractSets() []ExtractSet {
	ensureShiCiLoaded()

	sets := make([]ExtractSet, 0, len(extractTargets))
	for _, t := range extractTargets {
		sets = append(sets, ExtractSet{Material: t.material, Chars: copyExtracted(t.target)})
	}
	return sets
}

// LoadExtractSets 从库行还原全部经典字集（运行期装载用）
//
// 每行一个 (material, char) 条目；按 material 归组后按汉字码点升序写回
// 包级变量，与 mapToSortedSlice 的产出顺序一致（下游候选池注入顺序依赖它）。
// 未知 material 视为数据损坏并报错，避免静默丢弃。
func LoadExtractSets(rows []ExtractRow) error {
	grouped := make(map[string][]PoetryChar, len(extractTargets))
	for _, r := range rows {
		grouped[r.Material] = append(grouped[r.Material], PoetryChar{
			Char:     r.Char,
			Pinyin:   r.Pinyin,
			Meaning:  r.Meaning,
			Work:     r.Work,
			Chapter:  r.Chapter,
			Sentence: r.Sentence,
			Wuxing:   r.Wuxing,
			Gender:   r.Gender,
		})
	}

	for material := range grouped {
		if target := extractTargetOf(material); target == nil {
			return fmt.Errorf("经典字集物料 %q 未注册，无法还原", material)
		}
	}

	for _, t := range extractTargets {
		chars := grouped[t.material]
		sort.Slice(chars, func(i, j int) bool { return chars[i].Char < chars[j].Char })
		setExtracted(t.target, chars)
	}
	return nil
}

// extractTargetOf 按物料标识取包级结果变量，未注册返回 nil
func extractTargetOf(material string) *[]PoetryChar {
	for _, t := range extractTargets {
		if t.material == material {
			return t.target
		}
	}
	return nil
}

// ExportPoemEntries 导出全局诗词索引条目（导入期物化用）
//
// 未构建过索引时返回空切片（调用方据此判定「无诗词语料」）。
func ExportPoemEntries() []PoemEntry {
	idx := GetGlobalPoemIndex()
	if idx == nil {
		return nil
	}
	entries := idx.GetEntries()
	out := make([]PoemEntry, 0, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		out = append(out, *e)
	}
	return out
}

// LoadPoemEntries 从库行重建全局诗词索引（运行期装载用）
//
// FullText 与 Chars 均由 Content 纯函数派生（与 JSON 路径同源），
// 因此不落库、也不引入漂移面。
func LoadPoemEntries(rows []PoemEntryRow) {
	idx := NewPoemIndex()
	for _, r := range rows {
		content := r.Content
		idx.AddEntry(&PoemEntry{
			Title:    r.Title,
			Author:   r.Author,
			Dynasty:  r.Dynasty,
			Type:     r.Type,
			Category: r.Category,
			Content:  content,
			FullText: strings.Join(content, ""),
			Source:   r.Source,
			Book:     r.Book,
			Tags:     r.Tags,
			Chars:    extractCharsFromContent(content),
		})
	}

	poemIndexMu.Lock()
	GlobalPoemIndex = idx
	poemIndexMu.Unlock()
}