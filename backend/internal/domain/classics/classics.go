package classics

import "sync"

// classicsMu 保护所有经典数据的并发读写安全
// 写入仅在启动时进行一次，为后续热更新预留保护
var classicsMu sync.RWMutex

type ClassicName struct {
	Char     string `json:"char"`
	Pinyin   string `json:"pinyin"`
	Meaning  string `json:"meaning"`
	Source   string `json:"source"`
	Work     string `json:"work"`
	Chapter  string `json:"chapter"`
	Wuxing   string `json:"wuxing"`
	Gender   string `json:"gender"`
}

// ShijingNames 诗经用字列表，启动时由 LoadFromJSON 从 shijing.json 加载
var ShijingNames []ClassicName

// ChuciNames 楚辞用字列表，启动时由 LoadFromJSON 从 chuci.json 加载
var ChuciNames []ClassicName

// ShijingExtracted 从诗经原文提取的起名用字（包内使用，通过 getExtracted/setExtracted 访问）
var ShijingExtracted []PoetryChar

// ChuciExtracted 从楚辞原文提取的起名用字（包内使用，通过 getExtracted/setExtracted 访问）
var ChuciExtracted []PoetryChar

// GuwenGuanzhiExtracted 从 guwenguanzhi.json 原文提取的起名用字
var GuwenGuanzhiExtracted []PoetryChar

// ShiCiExtracted 从 shici.json 原文提取的起名用字
var ShiCiExtracted []PoetryChar

// LunyuExtracted 论语
var LunyuExtracted []PoetryChar

// MengziExtracted 孟子
var MengziExtracted []PoetryChar

// DaxueExtracted 大学
var DaxueExtracted []PoetryChar

// ZhongyongExtracted 中庸
var ZhongyongExtracted []PoetryChar

// SanzijingExtracted 三字经
var SanzijingExtracted []PoetryChar

// QianziwenExtracted 千字文
var QianziwenExtracted []PoetryChar

// DiziguiExtracted 弟子规
var DiziguiExtracted []PoetryChar

// YouxueqionglinExtracted 幼学琼林
var YouxueqionglinExtracted []PoetryChar

// ZengguangxianwenExtracted 增广贤文
var ZengguangxianwenExtracted []PoetryChar

// ShenglvqimengExtracted 声律启蒙
var ShenglvqimengExtracted []PoetryChar

// ZhuzijiaxunExtracted 朱子家训
var ZhuzijiaxunExtracted []PoetryChar

// QianjiashiExtracted 千家诗
var QianjiashiExtracted []PoetryChar

// WenzimengqiuExtracted 文字蒙求
var WenzimengqiuExtracted []PoetryChar

// BaijiaxingExtracted 百家姓
var BaijiaxingExtracted []PoetryChar

// setExtracted 设置提取结果（写锁保护，供 loader 使用）
func setExtracted(target *[]PoetryChar, val []PoetryChar) {
	classicsMu.Lock()
	*target = val
	classicsMu.Unlock()
}
