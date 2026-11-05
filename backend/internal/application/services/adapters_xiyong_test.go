package services

// adapters_xiyong_test.go — 喜用神单一真源一致性守护（docs/43 P1-3 阶段1）
//
// 背景：BaziAnalyzerAdapter.Analyze 装配 FateData.WuXingXiji 时，喜用神
// 曾取 fate.BalanceXiYongJi（fate 引擎自算，total/2 简单多数判强弱），
// 而 QiangRuo 与响应展示取 bazi weighted 算法（40/28/18 阈值），两套口径
// 混搭且可能自相矛盾；测名路径（fate_name_service.classicXiyongShen）
// 早已统一为 bazi 经典口径。收敛后本适配器以 bazi weighted 为唯一权威：
// 喜用神、强弱、忌仇全部同源于一次 bazi.AnalyzeBazi 结果。

import (
	"testing"
	"time"

	"name/internal/domain/bazi"
	"name/internal/domain/fate"
)

// TestBaziAnalyze_XiyongUsesWeightedAuthority 展示口径 == 评分器口径（同一真源）
func TestBaziAnalyze_XiyongUsesWeightedAuthority(t *testing.T) {
	born := time.Date(2024, 5, 20, 10, 0, 0, 0, time.UTC)

	adapter := NewBaziAnalyzerAdapter()
	fd, err := adapter.Analyze(born, fate.Gender("male"))
	if err != nil {
		t.Fatalf("Analyze 失败: %v", err)
	}

	// 权威源：同生辰直接调 bazi weighted 算法
	ref, err := bazi.AnalyzeBazi(2024, 5, 20, 10, 0)
	if err != nil {
		t.Fatalf("AnalyzeBazi 失败: %v", err)
	}
	if len(ref.Xiyongshen) == 0 {
		t.Fatal("参照结果 Xiyongshen 为空，用例失效")
	}

	// ① 喜用神列表与 bazi 权威源逐元素一致（此前为 BalanceXiYongJi 口径）
	if len(fd.WuXingXiji.XiYongShen) != len(ref.Xiyongshen) {
		t.Fatalf("XiYongShen = %v，权威源 = %v（元素数不同，非同源）",
			fd.WuXingXiji.XiYongShen, ref.Xiyongshen)
	}
	for i := range ref.Xiyongshen {
		if fd.WuXingXiji.XiYongShen[i] != ref.Xiyongshen[i] {
			t.Fatalf("XiYongShen[%d] = %s，权威源 = %s（顺序或取值不同，非同源）",
				i, fd.WuXingXiji.XiYongShen[i], ref.Xiyongshen[i])
		}
	}

	// ② 用神 = 权威源首元素
	if fd.WuXingXiji.YongShen != ref.Xiyongshen[0] {
		t.Errorf("YongShen = %s，期望权威源首元素 %s", fd.WuXingXiji.YongShen, ref.Xiyongshen[0])
	}

	// ③ 强弱与权威源同源（强弱栏不得再与喜用神分属两套算法）
	if fd.WuXingXiji.QiangRuo != ref.DayMasterStrength {
		t.Errorf("QiangRuo = %s，权威源 = %s", fd.WuXingXiji.QiangRuo, ref.DayMasterStrength)
	}

	// ④ 忌/仇按同一强弱结论推导（此前 fallback 错把 Xiyongshen[1] 当忌神）
	wantJi, wantChou := bazi.DeriveJiChou(ref.RishouWuxing, ref.DayMasterStrength)
	if fd.WuXingXiji.Ji != wantJi || fd.WuXingXiji.Chou != wantChou {
		t.Errorf("忌/仇 = (%s, %s)，期望 (%s, %s)",
			fd.WuXingXiji.Ji, fd.WuXingXiji.Chou, wantJi, wantChou)
	}
}
