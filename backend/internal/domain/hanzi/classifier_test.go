package hanzi

import "testing"

// TestIsCuratedNamingChar 策展起名用字覆盖表判断
//
// 策展覆盖表（characterCategoryList，17 个起名分类约 446 字）是人工精选的起名好字，
// verify_fate 复测验证它是荒谬字与好字的强区分信号：
//   - 经典好字（毅/辉/涛/英/琳/雅/梅/强/凯）8/12 在表内
//   - 荒谬字（贪/疟/骂/吠/靶/振/凑/递/备/宦/够/播/蚊/浅/眉/驳/圃/沈/亩/泥/沟）
//     21/22 不在表内（仅"清"在，而清本身是优质起名字，不属荒谬字）
//
// 注意：本表与 NamingCategories（可能被自动分类/五行兜底污染，如"贪"水→"清新水韵"
// 也会获分类）不同，是唯一的「人工策展」信号，WenHuaRater 据此给策展字文化加分。
func TestIsCuratedNamingChar(t *testing.T) {
	// 策展覆盖表内的优质起名字 → 应返回 true
	inTable := []string{"毅", "辉", "涛", "英", "琳", "雅", "梅", "强", "凯"}
	for _, ch := range inTable {
		if !IsCuratedNamingChar(ch) {
			t.Errorf("IsCuratedNamingChar(%q) = false, 期望 true（策展好字应在表内）", ch)
		}
	}

	// 荒谬字（verify_fate 单名 Top5 反复混入）→ 应返回 false
	outTable := []string{
		"贪", "疟", "骂", "吠", "靶", "振", "凑", "递", "备", "宦",
		"够", "播", "蚊", "浅", "眉", "驳", "圃", "沈", "亩", "泥", "沟",
	}
	for _, ch := range outTable {
		if IsCuratedNamingChar(ch) {
			t.Errorf("IsCuratedNamingChar(%q) = true, 期望 false（荒谬字不应在策展表）", ch)
		}
	}

	// 空串安全
	if IsCuratedNamingChar("") {
		t.Error("IsCuratedNamingChar(\"\") = true, 期望 false")
	}
}
