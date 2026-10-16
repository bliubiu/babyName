package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/logger"
)

// 本文件实现「运行时统一只读 SQLite 数据源」的物料读取路径
// （docs/31 ADR-008）：从库读出 typed 结构后交给各域装载函数，
// 运行时不再解析 JSON/CSV，也不再做字段归一化。

// LoadMaterialsFromDB 从库载入全部可用物料
//
// 采用「DB 优先、物料级降级」策略：某物料在库中缺失时只跳过该物料并记 warn，
// 不影响其他物料装载；调用方可据此回落到 JSON 路径（见 data.SourcePolicy）。
func (s *Store) LoadMaterialsFromDB() []string {
	loaded := make([]string, 0, len(MaterialTables()))

	if s.loadKangxiFromDB() {
		loaded = append(loaded, "kangxi")
	}
	if s.loadHanziFromDB() {
		loaded = append(loaded, "namer")
	}
	if s.loadCharFreqFromDB() {
		loaded = append(loaded, "name_frequency")
	}
	if s.loadBigramFromDB() {
		loaded = append(loaded, "name_bigram")
	}
	if s.loadWordsFromDB() {
		loaded = append(loaded, "word")
	}
	if s.loadGateFromDB() {
		loaded = append(loaded, "naming_gate")
	}
	if s.loadForbiddenFromDB() {
		loaded = append(loaded, "forbidden_combos")
	}
	if s.loadAllowlistFromDB() {
		loaded = append(loaded, "allowlist")
	}
	if s.loadYijingFromDB() {
		loaded = append(loaded, "yijing")
	}
	if s.loadZodiacFromDB() {
		loaded = append(loaded, "zodiac")
	}
	return loaded
}

// loadKangxiFromDB 从库载入康熙笔画（复用 store_kangxi.go 的实现）
func (s *Store) loadKangxiFromDB() bool {
	if err := s.LoadKangxiStrokesFromDB(); err != nil {
		logger.Warn("sqlite: 康熙笔画从库载入失败，回落其他来源", logger.ErrField(err))
		return false
	}
	return s.tableHasRows("kangxi_strokes")
}

// loadHanziFromDB 从 hanzi_data 读出字库并填充内存索引
func (s *Store) loadHanziFromDB() bool {
	if !s.tableHasRows("hanzi_data") {
		logger.Warn("sqlite: hanzi_data 为空，运行时回落 JSON 字库")
		return false
	}

	rows, err := s.db.Query(`SELECT char, pinyin, strokes, kangxi_strokes, radical, meaning,
		wuxing, gender, level, tone, name_freq_tier, gender_tags, style_tags, usage_level,
		positive_score, phonetic_score, modern_score, classical_score,
		is_polyphonic, is_rare, is_negative, pair_blacklist, curation_level, name_penalty,
		naming_categories FROM hanzi_data`)
	if err != nil {
		logger.Error("sqlite: 查询 hanzi_data 失败", logger.ErrField(err))
		return false
	}
	defer func() { _ = rows.Close() }()

	data := make(map[string]hanzi.Hanzi, 8192)
	namers := make(map[string]hanzi.NamerChar, 8192)
	for rows.Next() {
		h, err := scanHanziRow(rows)
		if err != nil {
			logger.Error("sqlite: 扫描 hanzi_data 行失败", logger.ErrField(err))
			continue
		}
		data[h.Char] = h
		namers[h.Char] = hanzi.ToNamerChar(h)
	}
	if err := rows.Err(); err != nil {
		logger.Error("sqlite: 遍历 hanzi_data 失败", logger.ErrField(err))
		return false
	}
	if len(data) == 0 {
		logger.Warn("sqlite: hanzi_data 读出为空，运行时回落 JSON 字库")
		return false
	}

	// 同时重建 NamerCharMap：JSON 路径会建这张表，库路径此前漏建导致
	// GetNamerLevel/GetNamerChar 失效（等级闸门放行三级/表外字，e2e 门禁暴露），
	// 现统一经 ToNamerChar 逆向投影对齐两条路径（docs/31 ADR-012）
	hanzi.ReplaceNamerCharMap(namers)
	hanzi.ReplaceHanziData(data)
	hanzi.ApplyWuxingOverrides()
	s.loadCharGroupsFromDB()
	return true
}

// loadCharGroupsFromDB 从 char_groups 表载入偏旁分组
//
// 失败不影响字库装载：分组只影响展示性筛选，缺失时退回空集合
func (s *Store) loadCharGroupsFromDB() {
	if !s.tableHasRows("char_groups") {
		return
	}
	rows, err := s.db.Query("SELECT radical, name, meaning, chars FROM char_groups ORDER BY radical")
	if err != nil {
		logger.Warn("sqlite: 载入偏旁分组失败", logger.ErrField(err))
		return
	}
	defer func() { _ = rows.Close() }()

	groups := make([]hanzi.StandardCharGroup, 0, 64)
	for rows.Next() {
		var (
			g     hanzi.StandardCharGroup
			chars string
		)
		if err := rows.Scan(&g.Radical, &g.Name, &g.Meaning, &chars); err != nil {
			continue
		}
		g.Chars = unmarshalTags(chars)
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		logger.Warn("sqlite: 遍历偏旁分组失败", logger.ErrField(err))
		return
	}
	hanzi.SetNamerGroups(groups)
	logger.Info("sqlite: 偏旁分组已从库载入", logger.Int("分组数", len(groups)))
}

// scanHanziRow 把一行 hanzi_data 扫描为 hanzi.Hanzi
func scanHanziRow(rows *sql.Rows) (hanzi.Hanzi, error) {
	var (
		h                                                hanzi.Hanzi
		genderTags, styleTags, pairBlacklist, namingCats string
		polyphonic, rare, negative                       int
	)
	err := rows.Scan(&h.Char, &h.Pinyin, &h.Strokes, &h.KangxiStrokes, &h.Radical, &h.Meaning,
		&h.Wuxing, &h.Gender, &h.Level, &h.Tone, &h.NameFreqTier,
		&genderTags, &styleTags, &h.UsageLevel,
		&h.PositiveScore, &h.PhoneticScore, &h.ModernScore, &h.ClassicalScore,
		&polyphonic, &rare, &negative, &pairBlacklist, &h.CurationLevel, &h.NamePenalty,
		&namingCats)
	if err != nil {
		return h, err
	}
	h.IsPolyphonic = polyphonic != 0
	h.IsRare = rare != 0
	h.IsNegative = negative != 0
	h.GenderTags = unmarshalTags(genderTags)
	h.StyleTags = unmarshalTags(styleTags)
	h.PairBlacklist = unmarshalTags(pairBlacklist)
	h.NamingCategories = unmarshalTags(namingCats)
	return h, nil
}

// unmarshalTags 解析标签 JSON 数组，失败返回空切片
func unmarshalTags(s string) []string {
	if s == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return []string{}
	}
	if out == nil {
		return []string{}
	}
	return out
}

// loadCharFreqFromDB 从库载入单字人名频率
func (s *Store) loadCharFreqFromDB() bool {
	if !s.tableHasRows("name_frequency") {
		return false
	}
	rows, err := s.db.Query("SELECT char, count, rank, tier FROM name_frequency")
	if err != nil {
		logger.Error("sqlite: 查询 name_frequency 失败", logger.ErrField(err))
		return false
	}
	defer func() { _ = rows.Close() }()

	var items []hanzi.CharFrequency
	for rows.Next() {
		var f hanzi.CharFrequency
		if err := rows.Scan(&f.Char, &f.Count, &f.Rank, &f.Tier); err != nil {
			continue
		}
		items = append(items, f)
	}
	if err := rows.Err(); err != nil || len(items) == 0 {
		return false
	}
	hanzi.SetFrequencyFromData(items)
	return true
}

// loadBigramFromDB 从库载入双字人名频率
func (s *Store) loadBigramFromDB() bool {
	if !s.tableHasRows("name_bigram") {
		return false
	}
	rows, err := s.db.Query("SELECT char1, char2, count, rank, tier FROM name_bigram")
	if err != nil {
		logger.Error("sqlite: 查询 name_bigram 失败", logger.ErrField(err))
		return false
	}
	defer func() { _ = rows.Close() }()

	var items []hanzi.BigramFrequency
	for rows.Next() {
		var b hanzi.BigramFrequency
		if err := rows.Scan(&b.Char1, &b.Char2, &b.Count, &b.Rank, &b.Tier); err != nil {
			continue
		}
		items = append(items, b)
	}
	if err := rows.Err(); err != nil || len(items) == 0 {
		return false
	}
	hanzi.SetBigramFromData(items)
	return true
}

// loadWordsFromDB 从库载入词典释义
func (s *Store) loadWordsFromDB() bool {
	if !s.tableHasRows("words") {
		return false
	}
	rows, err := s.db.Query("SELECT char, explanation FROM words")
	if err != nil {
		logger.Error("sqlite: 查询 words 失败", logger.ErrField(err))
		return false
	}
	defer func() { _ = rows.Close() }()

	entries := make([]hanzi.WordEntry, 0, 16384)
	for rows.Next() {
		var w hanzi.WordEntry
		if err := rows.Scan(&w.Word, &w.Explanation); err != nil {
			continue
		}
		entries = append(entries, w)
	}
	if err := rows.Err(); err != nil || len(entries) == 0 {
		return false
	}
	hanzi.SetWordMeanings(entries)
	return true
}

// loadGateFromDB 从库载入命名质量门禁字表
func (s *Store) loadGateFromDB() bool {
	if !s.tableHasRows("naming_gate") {
		return false
	}
	chars, err := s.queryStrings("SELECT char FROM naming_gate")
	if err != nil || len(chars) == 0 {
		return false
	}
	fate.SetNonNamingChars(chars)
	return true
}

// loadForbiddenFromDB 从库载入禁忌双字组合
func (s *Store) loadForbiddenFromDB() bool {
	if !s.tableHasRows("forbidden_combos") {
		return false
	}
	combos, err := s.queryStrings("SELECT combo FROM forbidden_combos")
	if err != nil || len(combos) == 0 {
		return false
	}
	fate.SetForbiddenCombos(combos)
	return true
}

// loadAllowlistFromDB 从库载入显式白名单（豁免黑名单 + 祈福组合，docs/33 门禁层 M1）
func (s *Store) loadAllowlistFromDB() bool {
	if !s.tableHasRows("allowlist") {
		return false
	}
	rows, err := s.db.Query("SELECT entry, kind FROM allowlist")
	if err != nil {
		logger.Error("sqlite: 查询 allowlist 失败", logger.ErrField(err))
		return false
	}
	defer func() { _ = rows.Close() }()

	entries := make([]fate.AllowEntry, 0, 256)
	for rows.Next() {
		var e fate.AllowEntry
		if err := rows.Scan(&e.Entry, &e.Kind); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil || len(entries) == 0 {
		return false
	}
	fate.SetAllowlist(entries)
	return true
}

// loadYijingFromDB 从库载入易经六十四卦
func (s *Store) loadYijingFromDB() bool {
	if !s.tableHasRows("yijing_hexagrams") {
		return false
	}
	rows, err := s.db.Query(`SELECT id, name, number, symbol, upper_trigram, lower_trigram,
		gua_ci, xiang_ci, yao_ci, interpretation FROM yijing_hexagrams ORDER BY id`)
	if err != nil {
		logger.Error("sqlite: 查询 yijing_hexagrams 失败", logger.ErrField(err))
		return false
	}
	defer func() { _ = rows.Close() }()

	var list []yijing.Hexagram
	for rows.Next() {
		var (
			h     yijing.Hexagram
			yaoCi string
		)
		if err := rows.Scan(&h.ID, &h.Name, &h.Number, &h.Symbol, &h.UpperTrigram,
			&h.LowerTrigram, &h.GuaCi, &h.XiangCi, &yaoCi, &h.Interpretation); err != nil {
			continue
		}
		h.YaoCi = unmarshalTags(yaoCi)
		list = append(list, h)
	}
	if err := rows.Err(); err != nil || len(list) == 0 {
		return false
	}
	yijing.SetHexagrams(list)
	return true
}

// loadZodiacFromDB 从库载入十二生肖
func (s *Store) loadZodiacFromDB() bool {
	if !s.tableHasRows("zodiac") {
		return false
	}
	rows, err := s.db.Query(`SELECT id, name, wuxing, compatible, conflicting,
		avoid_chars, lucky_numbers FROM zodiac ORDER BY id`)
	if err != nil {
		logger.Error("sqlite: 查询 zodiac 失败", logger.ErrField(err))
		return false
	}
	defer func() { _ = rows.Close() }()

	var list []zodiac.Zodiac
	for rows.Next() {
		var (
			z                             zodiac.Zodiac
			compatible, conflicting, luck string
		)
		if err := rows.Scan(&z.ID, &z.Name, &z.Wuxing, &compatible, &conflicting,
			&z.AvoidChars, &luck); err != nil {
			continue
		}
		z.Compatible = unmarshalTags(compatible)
		z.Conflicting = unmarshalTags(conflicting)
		z.LuckyNumbers = unmarshalInts(luck)
		list = append(list, z)
	}
	if err := rows.Err(); err != nil || len(list) == 0 {
		return false
	}
	zodiac.SetZodiacs(list)
	return true
}

// unmarshalInts 解析整型 JSON 数组
func unmarshalInts(s string) []int {
	if s == "" {
		return []int{}
	}
	var out []int
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return []int{}
	}
	if out == nil {
		return []int{}
	}
	return out
}

// tableHasRows 判断表是否存在且非空
func (s *Store) tableHasRows(table string) bool {
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

// MaterialLoadReport 运行时物料装载报告
type MaterialLoadReport struct {
	Loaded  []string `json:"loaded"`
	Missing []string `json:"missing"`
}

// LoadMaterialsFromDBWithReport 装载全部物料并给出「已载/缺失」清单
func (s *Store) LoadMaterialsFromDBWithReport() MaterialLoadReport {
	loaded := map[string]bool{}
	for _, name := range s.LoadMaterialsFromDB() {
		loaded[name] = true
	}

	report := MaterialLoadReport{}
	for _, name := range materialNames() {
		if loaded[name] {
			report.Loaded = append(report.Loaded, name)
		} else {
			report.Missing = append(report.Missing, name)
		}
	}
	if len(report.Missing) > 0 {
		logger.Warn("sqlite: 部分物料库中缺失，运行时应回落到对应源文件",
			logger.String("缺失物料", strings.Join(report.Missing, ",")))
	}
	logger.Info("sqlite: 物料已从库载入", logger.Int("物料数", len(report.Loaded)))
	return report
}

// materialNames 运行时支持的物料名（有序）
func materialNames() []string {
	return []string{
		"kangxi", "namer", "name_frequency", "name_bigram", "word",
		"naming_gate", "forbidden_combos", "yijing", "zodiac",
	}
}

// VerifyMaterialsFromDB 校验库中物料是否齐备，用于启动前 fail-fast 引导
func (s *Store) VerifyMaterialsFromDB() error {
	report := s.LoadMaterialsFromDBWithReport()
	if len(report.Missing) == 0 {
		return nil
	}
	return fmt.Errorf("物料库不完整，缺失 %d 项: %v（请执行 materials import 导入）",
		len(report.Missing), report.Missing)
}
