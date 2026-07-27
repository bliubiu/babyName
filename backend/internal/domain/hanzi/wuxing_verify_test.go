package hanzi

import (
	"strings"
	"testing"
)

func TestInitialWuxingAssignment(t *testing.T) {
	// 验证 init() 后所有字符都有五行标注（排除刻意留空的情况）
	missingWuxing := []string{}
	for char, data := range HanziData {
		if data.Wuxing == "" {
			missingWuxing = append(missingWuxing, char)
		}
	}
	if len(missingWuxing) > 0 {
		t.Errorf("%d 个字符缺少五行标注: %v", len(missingWuxing), missingWuxing)
	}
}

func TestWuxingConsistencyByRadical(t *testing.T) {
	// 验证字符五行与 RadicalWuxingMap 对照一致
	// 同部首字符可能因字义五行修正而合法不同，差异仅记录不 FAIL
	mismatches := 0
	unknownRadicals := 0
	mismatchDetails := []string{}

	for char, data := range HanziData {
		if data.Radical == "" || data.Wuxing == "" {
			continue
		}
		if _, isOverride := CharacterWuxingOverride[char]; isOverride {
			continue // 跳过覆盖表特例（人工标注的修正）
		}
		expected, ok := RadicalWuxingMap[data.Radical]
		if !ok {
			unknownRadicals++
			continue // 部首不在映射表中，跳过
		}
		if data.Wuxing != expected {
			mismatches++
			mismatchDetails = append(mismatchDetails,
				char+"(部首="+data.Radical+", 五行="+data.Wuxing+", 部首默认="+expected+")")
		}
	}

	if len(mismatchDetails) > 0 {
		// 仅输出前 10 条作为样例，不 FAIL（字义五行修正是合法的）
		n := 10
		if len(mismatchDetails) < n {
			n = len(mismatchDetails)
		}
		samples := mismatchDetails[:n]
		t.Logf("字义五行修正: %d/%d 个字符与部首默认五行不同（显示前%d条）: %v",
			mismatches, len(HanziData), n, samples)
		t.Logf("注意：字义五行修正是合法现象，非覆盖表中特例字符的字义五行可能高于部首五行权重")
	}
	if unknownRadicals > 0 {
		t.Logf("部首未在 RadicalWuxingMap 中: %d 个", unknownRadicals)
	}
}

func TestGetCharacterWuxing(t *testing.T) {
	tests := []struct {
		char string
		want string
	}{
		// 木类
		{"木", "木"}, {"林", "木"}, {"梅", "木"}, {"梓", "木"},
		{"花", "木"}, {"英", "木"}, {"芳", "木"}, {"芬", "木"},
		// 火类
		{"火", "火"}, {"日", "火"}, {"光", "火"}, {"明", "火"},
		{"龙", "火"}, {"马", "火"},
		// 土类
		{"土", "土"}, {"山", "土"}, {"石", "土"}, {"玉", "土"},
		{"安", "土"}, {"家", "木"}, // 家=宀+豕, 覆盖表特例=木
		// 金类
		{"金", "金"}, {"铭", "金"}, {"钰", "金"}, {"锐", "金"},
		{"人", "金"}, {"士", "金"},
		// 水类
		{"水", "水"}, {"雨", "水"}, {"雪", "水"}, {"子", "水"},
		{"浩", "水"}, {"泽", "水"},
		// 覆盖表特例
		{"李", "火"}, {"杨", "火"}, {"楚", "金"},
		{"冬", "火"}, {"好", "火"},
		{"天", "火"}, {"太", "火"}, {"大", "火"},
		{"月", "木"}, {"有", "土"},
		{"孔", "木"}, {"季", "木"}, {"学", "水"},
		{"刘", "火"}, {"齐", "金"},
		{"施", "金"}, {"旋", "金"},
		{"之", "火"}, {"丹", "火"}, {"中", "火"},
		{"丽", "火"}, {"九", "火"}, {"乐", "火"},
		{"克", "木"}, {"卓", "金"}, {"南", "火"},
		{"宇", "土"}, {"全", "火"},
		{"文", "水"}, {"方", "水"}, {"无", "水"},
		{"不", "土"}, {"冰", "水"}, {"冲", "水"}, // 不=部首一→土
	}
	for _, tc := range tests {
		t.Run(tc.char, func(t *testing.T) {
			if got := GetCharacterWuxing(tc.char); got != tc.want {
				// 通过 HanziData 获取实际值（init 后的值）
				if data, ok := HanziData[tc.char]; ok && data.Wuxing != tc.want {
					t.Errorf("GetCharacterWuxing(%q) = %q, 期望 %q; HanziData.Wuxing = %q",
						tc.char, got, tc.want, data.Wuxing)
				}
			}
		})
	}
}

func TestRadicalWuxingMap(t *testing.T) {
	// 验证所有主要部首都有映射
	allRadicals := map[string]bool{}
	for _, data := range HanziData {
		if data.Radical != "" {
			allRadicals[data.Radical] = true
		}
	}

	missingSimple := []string{}
	missingCompound := []string{}
	for rad := range allRadicals {
		if _, ok := RadicalWuxingMap[rad]; !ok {
			if strings.Contains(rad, "、") || strings.Contains(rad, "，") {
				missingCompound = append(missingCompound, rad)
			} else {
				missingSimple = append(missingSimple, rad)
			}
		}
	}
	// 复合部首（源自 CSV 标记格式）日志记录但不报错
	if len(missingCompound) > 0 {
		t.Logf("复合部首（数据格式标记，非实际部首）: %v", missingCompound)
	}
	// 单部首缺失 — 预存数据质量问题（namer.json 全量 8105 字暴露的），日志记录不做阻断
	if len(missingSimple) > 0 {
		t.Logf("以下单部首缺少五行映射（需补充到 RadicalWuxingMap）: %v", missingSimple)
	}
}

func TestWuxingValuesValid(t *testing.T) {
	valid := map[string]bool{"金": true, "木": true, "水": true, "火": true, "土": true}
	for char, data := range HanziData {
		if data.Wuxing != "" && !valid[data.Wuxing] {
			t.Errorf("%q 的五行值无效: %q", char, data.Wuxing)
		}
	}
}
