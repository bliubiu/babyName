package yijing

import (
	"strings"
	"testing"
)

// canonicalXiangCi 通行本《周易》大象（彖传/大象传）金标准抽样
// 覆盖此前已确认有误的剥/鼎，以及若干常见卦，防止再次写错。
var canonicalXiangCi = map[int]string{
	1:  "天行健，君子以自强不息",
	2:  "地势坤，君子以厚德载物",
	3:  "云雷屯，君子以经纶",
	4:  "山下出泉，蒙，君子以果行育德",
	11: "天地交泰，后以财成天地之道",
	12: "天地不交，否，君子以俭德辟难",
	15: "地中有山，谦，君子以裒多益寡",
	23: "山附地上，剥。上以厚下安宅",
	24: "雷在地中，复，先王以至日闭关",
	29: "水洊至，习坎，君子以常德行，习教事",
	30: "明两作，离，大人以继明照于四方",
	50: "木上有火，鼎。君子以正位凝命",
	51: "洊雷震，君子以恐惧修省",
	52: "兼山艮，君子以思不出其位",
	63: "水在火上，既济，君子以思患而豫防之",
	64: "火在水上，未济，君子以慎辨物居方",
}

func TestHexagram_CanonicalXiangCi(t *testing.T) {
	for id, want := range canonicalXiangCi {
		h := GetHexagramByNumber(id)
		if h == nil {
			t.Errorf("卦 #%d 缺失", id)
			continue
		}
		if h.XiangCi != want {
			t.Errorf("卦 #%d（%s）象辞 = %q，期望通行本 %q", id, h.Name, h.XiangCi, want)
		}
	}
}

// TestHexagram_XiangCiNotEqualGuaCi 防止「把卦辞当象辞」类错误（剥卦历史缺陷）
func TestHexagram_XiangCiNotEqualGuaCi(t *testing.T) {
	for _, h := range HexagramList {
		if strings.TrimSpace(h.XiangCi) == strings.TrimSpace(h.GuaCi) {
			t.Errorf("卦 #%d（%s）象辞与卦辞完全相同，疑似复制错误", h.Number, h.Name)
		}
	}
}

// TestHexagram_StructuralIntegrity 结构完整性：64 卦、6 爻、非空字段
func TestHexagram_StructuralIntegrity(t *testing.T) {
	if len(HexagramList) != 64 {
		t.Fatalf("卦数 = %d，期望 64", len(HexagramList))
	}
	seen := map[int]bool{}
	for _, h := range HexagramList {
		if h.Number < 1 || h.Number > 64 {
			t.Errorf("卦 %q Number=%d 越界", h.Name, h.Number)
		}
		if seen[h.Number] {
			t.Errorf("卦序 #%d 重复", h.Number)
		}
		seen[h.Number] = true

		if h.GuaCi == "" || h.XiangCi == "" || h.Name == "" {
			t.Errorf("卦 #%d（%s）卦辞/象辞/名称存在空字段", h.Number, h.Name)
		}
		// 乾坤另有用九/用六，允许 7 条；其余卦固定 6 爻
		wantYao := 6
		if h.Number == 1 || h.Number == 2 {
			wantYao = 7
		}
		if len(h.YaoCi) != wantYao {
			t.Errorf("卦 #%d（%s）爻辞条数 = %d，期望 %d", h.Number, h.Name, len(h.YaoCi), wantYao)
		}
		for i, y := range h.YaoCi {
			if strings.TrimSpace(y) == "" {
				t.Errorf("卦 #%d（%s）第 %d 爻辞为空", h.Number, h.Name, i+1)
			}
		}
	}
	for i := 1; i <= 64; i++ {
		if !seen[i] {
			t.Errorf("缺少卦序 #%d", i)
		}
	}
}
