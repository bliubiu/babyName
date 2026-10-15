package hanzi

// 本文件是 NamerChar → Hanzi 的**唯一映射真源**（docs/31 ADR-012）。
//
// 修订前，namer_loader.go 与 loader.go 各自内联构造 Hanzi 字面量，两处字段清单不一致：
// namer 路径漏掉 Level / NameFreqTier / IsPolyphonic 等字段，导致 hanzi_data 表
// 「列已存在但值恒为默认」——结构对齐而语义不对齐。

// ToHanzi 把 namer.json 的一条记录投影为运行时结构 Hanzi
//
// 字段真值来源（实测 namer.json 仅 17 个字段，8105 字）：
//   - 有真值：char/pinyin/strokes/radical/meaning/wuxing/gender/level/tone/
//     styleTags/usageLevel/positiveScore/isRare/isNegative/isPolyphonic/namePenalty/
//     namingCategories/nameFreqTier
//   - 无真值（结构无对应物料，保持零值，不得以默认值伪装成有值）：
//     genderTags / phoneticScore / modernScore / classicalScore / pairBlacklist / curationLevel
func ToHanzi(nc NamerChar) Hanzi {
	return Hanzi{
		Char:    nc.Char,
		Pinyin:  nc.Pinyin,
		Strokes: nc.Strokes,
		// 康熙笔画不在 namer.json 中，由 mergeKangxiStrokes 事后注入
		KangxiStrokes: nc.KangxiStrokes,
		Radical:       nc.Radical,
		Meaning:       nc.Meaning,
		Wuxing:        nc.Wuxing,
		Gender:        normalizeGender(nc.Gender),

		Level:            nc.Level,
		Tone:             nc.Tone,
		StyleTags:        nc.StyleTags,
		UsageLevel:       nc.UsageLevel,
		PositiveScore:    nc.PositiveScore,
		IsRare:           nc.IsRare,
		IsNegative:       nc.IsNegative,
		IsPolyphonic:     nc.IsPolyphonic,
		NamePenalty:      nc.NamePenalty,
		NameFreqTier:     nc.NameFreqTier,
		NamingCategories: nc.NamingCategories,
	}
}

// ToHanziFromJSON 把历史 hanzi.json 的一条记录投影为运行时结构 Hanzi
//
// wuxing 已由调用方按「覆盖表 > 部首派生 > JSON 原值」的优先级算好；
// gender 统一走归一化，避免两处各写一份枚举映射
func ToHanziFromJSON(jh JSONHanzi, wuxing string, namingCats []string) Hanzi {
	return Hanzi{
		Char:    jh.Char,
		Pinyin:  jh.Pinyin,
		Strokes: jh.Strokes,
		Radical: jh.Radical,
		Meaning: jh.Meaning,
		Wuxing:  wuxing,
		Gender:  normalizeGender(jh.Gender),

		Tone:             jh.Tone,
		GenderTags:       jh.GenderTags,
		StyleTags:        jh.StyleTags,
		UsageLevel:       jh.UsageLevel,
		PositiveScore:    jh.PositiveScore,
		PhoneticScore:    jh.PhoneticScore,
		ModernScore:      jh.ModernScore,
		ClassicalScore:   jh.ClassicalScore,
		IsPolyphonic:     jh.IsPolyphonic,
		IsRare:           jh.IsRare,
		IsNegative:       jh.IsNegative,
		PairBlacklist:    jh.PairBlacklist,
		CurationLevel:    jh.CurationLevel,
		NamePenalty:      jh.NamePenalty,
		NamingCategories: namingCats,
	}
}

// ToNamerChar 将 Hanzi 逆向投影为 NamerChar（ADR-012 单一映射真源）
//
// 用途：SQLite 装载路径按行扫描出 Hanzi 后，经本函数重建 NamerCharMap，
// 保证「库路径」与「JSON 路径」产出的内存状态一致（docs/31 ADR-008）。
// 仅覆盖 NamerChar 所需字段；Hanzi 独有字段（性别标签、忌讳配对、释义明细等）
// 在 namer.json 中本就无真值，不参与投影。
func ToNamerChar(h Hanzi) NamerChar {
	return NamerChar{
		Char:             h.Char,
		Pinyin:           h.Pinyin,
		Strokes:          h.Strokes,
		KangxiStrokes:    h.KangxiStrokes,
		Radical:          h.Radical,
		Meaning:          h.Meaning,
		Wuxing:           h.Wuxing,
		Level:            h.Level,
		Gender:           h.Gender,
		NamingCategories: h.NamingCategories,
		IsNegative:       h.IsNegative,
		IsRare:           h.IsRare,
		IsPolyphonic:     h.IsPolyphonic,
		NamePenalty:      h.NamePenalty,
		Tone:             h.Tone,
		StyleTags:        h.StyleTags,
		UsageLevel:       h.UsageLevel,
		PositiveScore:    h.PositiveScore,
		NameFreqTier:     h.NameFreqTier,
	}
}
