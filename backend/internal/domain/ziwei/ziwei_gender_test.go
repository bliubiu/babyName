package ziwei

import "testing"

// TestCalculateDaXian_性别枚举一致性 回归：API 枚举是 "male"/"female"，
// 历史缺陷是 isMale 只认 "男"——"male" 全被按女性逆行计算，紫微大限
// 方向整片反置且无任何报错。中文与英文枚举必须产出完全一致的大限表。
func TestCalculateDaXian_性别枚举一致性(t *testing.T) {
	const (
		yearGanZhi   = "甲子" // 甲年，子为阳支
		mingGong     = "寅宫" // 命宫（函数内部取第二个字为地支）
		fiveElements = "水二局"
	)

	maleCN := calculateDaXian(yearGanZhi, mingGong, fiveElements, "男")
	maleEN := calculateDaXian(yearGanZhi, mingGong, fiveElements, "male")
	femaleEN := calculateDaXian(yearGanZhi, mingGong, fiveElements, "female")

	if len(maleCN) == 0 || len(maleEN) == 0 || len(femaleEN) == 0 {
		t.Fatalf("大限表为空：len(maleCN)=%d", len(maleCN))
	}

	// 同义性别枚举：结果必须逐宫一致
	if !daXianEqual(maleCN, maleEN) {
		t.Fatalf("性别 \"男\" 与 \"male\" 的大限表不一致（历史缺陷：male 被按女性计算）")
	}
	// 异性别：命宫与年支同为阳，男女顺逆必相反，大限表应可区分
	if daXianEqual(maleCN, femaleEN) {
		t.Errorf("男/女的大限表完全一致，疑似性别未生效")
	}
}

func daXianEqual(a, b map[string]DaXianEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok || va != vb {
			return false
		}
	}
	return true
}
