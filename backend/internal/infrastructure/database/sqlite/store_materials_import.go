package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"name/internal/domain/hanzi"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/logger"
	"name/internal/materials/checksum"
	"name/internal/materials/normalize"
	"name/internal/materials/schema"
	"name/internal/materials/validate"
)

// 本文件实现 `materials import`：把 JSON/CSV 源物料归一化、校验后
// 单事务写入 SQLite，并把物料指纹登记到 materials_state（docs/31 ADR-008）。
//
// 写入语义（ADR-011）：一律 DELETE + INSERT 全量重建置于单事务内，
// 保证「同输入多次导入结果等价」且物料修订、删字都能传导。
// 未变更的物料按指纹短路跳过，避免数十 MB 物料的无谓重灌。

// maxImportErrors 单次导入最多收集的校验错误条数（超出即中止并报错）
const maxImportErrors = 20

// ImportOptions 导入参数
type ImportOptions struct {
	// DataDir 物料源目录（含 namer.json / word.json / raw/kangxi-strokecount.csv 等）
	DataDir string
	// Materials 只导入指定物料；为空表示全部
	Materials []string
	// Lenient 校验不通过时仅记录不阻断（数据清洗期取证用，ADR-009）
	Lenient bool
	// DryRun 只校验与统计，不写库
	DryRun bool
}

// ImportResult 单次导入的汇总结果
type ImportResult struct {
	Material string `json:"material"`
	Source   string `json:"source"`
	Checksum string `json:"checksum"`
	Version  string `json:"version"`
	Rows     int    `json:"rows"`
	Skipped  bool   `json:"skipped"`
	Reason   string `json:"reason,omitempty"`
	Err      string `json:"error,omitempty"`
}

// ImportReport 一次 import 命令的完整报告
type ImportReport struct {
	Results   []ImportResult `json:"results"`
	Written   int            `json:"written"`
	Skipped   int            `json:"skipped"`
	Failed    int            `json:"failed"`
	DryRun    bool           `json:"dryRun"`
	StartedAt string         `json:"startedAt"`
}

// Import 把全部（或指定）物料导入 SQLite
func (s *Store) Import(opt ImportOptions) (*ImportReport, error) {
	defs, err := resolveMaterials(opt.Materials)
	if err != nil {
		return nil, err
	}
	order, err := schema.TopoSort(defs)
	if err != nil {
		return nil, err
	}

	if !opt.DryRun {
		if err := createMaterialTables(s.writeDB); err != nil {
			return nil, err
		}
	}

	report := &ImportReport{DryRun: opt.DryRun, StartedAt: time.Now().Format("2006-01-02 15:04:05")}
	for _, name := range order {
		def := schema.Registry[name]
		r := s.importOne(def, opt)
		report.Results = append(report.Results, r)
		switch {
		case r.Err != "":
			report.Failed++
		case r.Skipped:
			report.Skipped++
		default:
			report.Written++
		}
	}
	return report, nil
}

// resolveMaterials 按名称过滤物料定义，并自动补齐依赖闭包
//
// 只导入 namer 时必须同时导入它依赖的 kangxi，否则康熙笔画合并无从进行；
// 依赖链上的物料同样参与拓扑排序，保证顺序正确。
func resolveMaterials(names []string) ([]schema.Definition, error) {
	if len(names) == 0 {
		return schema.Definitions, nil
	}

	picked := make(map[string]bool, len(names))
	unknown := make([]string, 0)
	for _, n := range names {
		if _, ok := schema.Registry[n]; !ok {
			unknown = append(unknown, n)
		}
		picked[n] = true
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("未知物料 %s（可用：%s）",
			strings.Join(unknown, ", "), strings.Join(schema.Names(), ", "))
	}

	// 广度优先补齐依赖闭包
	queue := make([]string, 0, len(picked))
	for n := range picked {
		queue = append(queue, n)
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		d, ok := schema.Registry[cur]
		if !ok {
			continue
		}
		for _, dep := range d.DependsOn {
			if picked[dep] {
				continue
			}
			picked[dep] = true
			queue = append(queue, dep)
		}
	}

	out := make([]schema.Definition, 0, len(picked))
	for _, d := range schema.Definitions {
		if picked[d.Name] {
			out = append(out, d)
		}
	}
	return out, nil
}

// importOne 导入单个物料
func (s *Store) importOne(def schema.Definition, opt ImportOptions) ImportResult {
	res := ImportResult{Material: def.Name, Source: def.Source}

	src := filepath.Join(opt.DataDir, filepath.FromSlash(def.Source))
	if _, err := os.Stat(src); err != nil {
		if def.Required {
			res.Err = fmt.Sprintf("必需物料源缺失: %s", src)
		} else {
			res.Skipped = true
			res.Reason = "可选物料源缺失，跳过"
		}
		return res
	}

	sum, err := checksum.File(src)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Checksum = sum

	// 指纹未变则短路，避免无谓重灌（ADR-014）
	if !opt.DryRun && !opt.Lenient && s.materialUnchanged(def.Name, sum) {
		res.Skipped = true
		res.Reason = "指纹未变更，跳过重灌"
		res.Rows = s.materialRowCount(def.Name)
		res.Version = s.storedMaterialVersion(def.Name)
		return res
	}

	// 解析 → 归一化 → 校验 → 写入：按注册表多态分发（store_material_registry.go）
	m, ok := materialRegistry[def.Name]
	if !ok {
		res.Err = fmt.Sprintf("物料 %s 未登记导入实现（见 store_material_registry.go）", def.Name)
		return res
	}
	return m.Import(s, src, res, opt)
}

// materialUnchanged 判断库内记录的指纹是否与传入一致且表非空
func (s *Store) materialUnchanged(material, sum string) bool {
	var stored string
	err := s.db.QueryRow("SELECT checksum FROM materials_state WHERE material = ?", material).Scan(&stored)
	if err != nil || stored == "" || stored != sum {
		return false
	}
	var n int
	for _, t := range materialTablesOf(material) {
		if err := s.db.QueryRow("SELECT COUNT(*) FROM " + t).Scan(&n); err != nil || n == 0 {
			return false
		}
	}
	return true
}

// materialTablesOf 物料对应的表
func materialTablesOf(material string) []string {
	if d, ok := schema.Registry[material]; ok {
		return d.Tables
	}
	return nil
}

// materialRowCount 物料当前落库行数（用于报告）
func (s *Store) materialRowCount(material string) int {
	total := 0
	for _, t := range materialTablesOf(material) {
		var n int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM " + t).Scan(&n); err == nil {
			total += n
		}
	}
	return total
}

// runImport 事务骨架：单事务内清表 → 写入 → 登记状态
func (s *Store) runImport(material string, tables []string, res *ImportResult, opt ImportOptions, write func(tx *sql.Tx) (int, error)) *ImportResult {
	if opt.DryRun {
		res.Rows = 0
		res.Reason = "试运行：仅校验，未写库"
		return res
	}

	tx, err := s.writeDB.Begin()
	if err != nil {
		res.Err = fmt.Sprintf("开启导入事务失败: %v", err)
		return res
	}
	defer func() { _ = tx.Rollback() }()

	for _, t := range tables {
		if _, err := tx.Exec("DELETE FROM " + t); err != nil {
			res.Err = fmt.Sprintf("清空表 %s 失败: %v", t, err)
			return res
		}
	}

	n, err := write(tx)
	if err != nil {
		res.Err = err.Error()
		return res
	}

	if _, err := tx.Exec(`INSERT INTO materials_state(material, version, checksum, row_count)
		VALUES(?,?,?,?)
		ON CONFLICT(material) DO UPDATE SET version = excluded.version,
			checksum = excluded.checksum, row_count = excluded.row_count,
			imported_at = CURRENT_TIMESTAMP`,
		material, res.Version, res.Checksum, n); err != nil {
		res.Err = fmt.Sprintf("登记物料状态失败: %v", err)
		return res
	}

	if err := tx.Commit(); err != nil {
		res.Err = fmt.Sprintf("提交导入事务失败: %v", err)
		return res
	}
	res.Rows = n
	logger.Info("sqlite: 物料已导入", logger.String("物料", material), logger.Int("行数", n))
	return res
}

// pendingErrors 收集待上报的校验错误（由各 importXxx 设置）
//
// 注意：不得用包级全局传递 —— 并发导入或多物料连续导入时会互相污染。
// 校验结果一律由各 importXxx 直接写入 res.Err 并返回。

// failValidate 记录校验错误；非 lenient 时直接返回错误字符串
func failValidate(errs validate.Errors, opt ImportOptions) string {
	if len(errs) == 0 {
		return ""
	}
	if opt.Lenient {
		logger.Warn("sqlite: 校验发现问题（lenient 模式继续导入）", logger.Int("问题数", len(errs)))
		return ""
	}
	return errs.Error()
}

// mustJSON 序列化标签切片；出错时退化为空数组而非中断导入
func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// --- 各物料导入实现 ---

// importKangxi 导入康熙笔画
func (s *Store) importKangxi(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	m, err := hanzi.ParseKangxiCSVFile(src)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Version = "csv"

	var items []schema.KangxiStroke
	for c, v := range m {
		items = append(items, schema.KangxiStroke{Char: c, Strokes: normalize.Stroke(v)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Char < items[j].Char })

	var errs validate.Errors
	for _, it := range items {
		errs = append(errs, validate.Kangxi(def.Name, it)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare("INSERT INTO kangxi_strokes(char, strokes) VALUES(?,?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, it := range items {
			if _, err := stmt.Exec(it.Char, it.Strokes); err != nil {
				return 0, fmt.Errorf("写入康熙笔画 %s 失败: %w", it.Char, err)
			}
		}
		return len(items), nil
	})
}

// namerJSON namer.json 的磁盘结构
type namerJSON struct {
	Version    string                    `json:"version"`
	Generated  string                    `json:"generated"`
	TotalChars int                       `json:"totalChars"`
	Chars      []hanzi.NamerChar         `json:"chars"`
	CharGroups []hanzi.StandardCharGroup `json:"charGroups"`
}

// importNamer 导入汉字表与偏旁分组（依赖康熙笔画做合并）
func (s *Store) importNamer(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var nj namerJSON
	if err := json.Unmarshal(raw, &nj); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = nj.Version + "@" + nj.Generated

	// 康熙笔画合并（docs/31 §9.6 跨物料合并优先级：康熙只填该列，不动简体笔画）
	kangxi := map[string]int{}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM kangxi_strokes").Scan(new(int)); err == nil {
		rows, qerr := s.db.Query("SELECT char, strokes FROM kangxi_strokes")
		if qerr == nil {
			for rows.Next() {
				var c string
				var v int
				if err := rows.Scan(&c, &v); err == nil {
					kangxi[c] = v
				}
			}
			_ = rows.Close()
		}
	}

	// 人名频率档位合并（docs/31 §9.6 步骤 4 / ADR-009）。
	//
	// 必须在此处从 name_frequency 表回读合并，而不能只依赖「频率物料导入时
	// UPDATE hanzi_data」：若仅 namer.json 变更（频率源不变），本函数会重写全表
	// name_freq_tier 为 namer 原值（namer.json 无该字段=0），而频率物料因指纹未变
	// 被短路跳过、其 UPDATE 不再执行 → 档位被静默清零。回读合并后，无论哪一侧变更，
	// 「库内频率表 → hanzi_data.name_freq_tier」这条派生链都不会断。
	// 口径与 JSON 运行时路径一致：仅 Tier ≥ 2（LoadFrequencyDB 过滤口径）。
	freqTier := map[string]int{}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM name_frequency").Scan(new(int)); err == nil {
		rows, qerr := s.db.Query("SELECT char, tier FROM name_frequency")
		if qerr == nil {
			for rows.Next() {
				var c string
				var t int
				if err := rows.Scan(&c, &t); err == nil && t >= 2 {
					freqTier[c] = t
				}
			}
			_ = rows.Close()
		}
	}

	// 逐字归一：namer.json 实测仅 17 个字段
	//   - 有真值：char/pinyin/strokes/radical/meaning/wuxing/gender/level/tone/
	//     styleTags/usageLevel/positiveScore/isRare/isNegative/isPolyphonic/namePenalty/
	//     namingCategories
	//   - 无真值（保持零值，不得伪造）：phoneticScore/modernScore/classicalScore/
	//     curationLevel/genderTags/pairBlacklist/nameFreqTier（后者由频率物料合并）
	words := make([]schema.Word, 0, len(nj.Chars))
	for _, nc := range nj.Chars {
		// 五行必须走与内存装载路径同一套解析规则（hanzi.ResolveWuxing）：
		// 字义法覆盖表 > 已有值 > 部首字形法兜底。
		// 此前此处直接落 nc.Wuxing 原值，漏掉了覆盖表与部首兜底，
		// 实测有 14 字在 hanzi_data 中为空而内存侧有值，
		// 导致「SQL 与内存结果等价」守护测试失败（docs/31 §11.9 第 2 项）。
		wuxing := hanzi.ResolveWuxing(nc.Char, nc.Wuxing, nc.Radical)

		w := normalize.Word(schema.Word{
			Char: nc.Char, Pinyin: nc.Pinyin, Strokes: nc.Strokes,
			KangxiStrokes: kangxi[nc.Char],
			Radical:       nc.Radical, Meaning: nc.Meaning, Wuxing: wuxing,
			Level: nc.Level, Gender: normalize.Gender(nc.Gender),
			Tone: nc.Tone, NameFreqTier: freqTier[nc.Char],
			UsageLevel: nc.UsageLevel, PositiveScore: nc.PositiveScore,
			NamePenalty:  nc.NamePenalty,
			IsPolyphonic: nc.IsPolyphonic, IsRare: nc.IsRare, IsNegative: nc.IsNegative,
			StyleTags:  nc.StyleTags,
			NamingCats: nc.NamingCategories,
		})
		words = append(words, w)
	}
	sort.Slice(words, func(i, j int) bool { return words[i].Char < words[j].Char })

	if msg := failValidate(validate.Words(def.Name, words, maxImportErrors), opt); msg != "" {
		res.Err = msg
		return res
	}

	// 偏旁分组：分组内汉字必须已入库（引用完整性）
	known := make(map[string]bool, len(words))
	for _, w := range words {
		known[w.Char] = true
	}
	groups := make([]schema.CharGroup, 0, len(nj.CharGroups))
	for _, g := range nj.CharGroups {
		groups = append(groups, normalize.CharGroup(schema.CharGroup{
			Radical: g.Radical, Name: g.Name, Meaning: g.Meaning, Chars: g.Chars,
		}))
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Radical < groups[j].Radical })

	var gerrs validate.Errors
	for _, g := range groups {
		gerrs = append(gerrs, validate.CharGroup(def.Name, g, known)...)
		if len(gerrs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(gerrs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare(hanziDataInsertSQL)
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		n := 0
		for _, w := range words {
			if _, err := stmt.Exec(wordValues(w)...); err != nil {
				return 0, fmt.Errorf("写入汉字 %s 失败: %w", w.Char, err)
			}
			n++
		}

		gstmt, err := tx.Prepare("INSERT INTO char_groups(radical, name, meaning, chars) VALUES(?,?,?,?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = gstmt.Close() }()
		for _, g := range groups {
			if _, err := gstmt.Exec(g.Radical, g.Name, g.Meaning, mustJSON(g.Chars)); err != nil {
				return 0, fmt.Errorf("写入偏旁分组 %s 失败: %w", g.Radical, err)
			}
			n++
		}
		return n, nil
	})
}

// wordValues 把规范汉字转成 hanzi_data 行值（与运行时投影保持同一语义）
func wordValues(w schema.Word) []any {
	return []any{
		w.Char, w.Pinyin, w.Strokes, w.KangxiStrokes, w.Radical, w.Meaning,
		w.Wuxing, string(w.Gender), w.Level, w.Tone, w.NameFreqTier,
		mustJSON(w.GenderTags), mustJSON(w.StyleTags), w.UsageLevel,
		w.PositiveScore, w.PhoneticScore, w.ModernScore, w.ClassicalSc,
		boolToInt(w.IsPolyphonic), boolToInt(w.IsRare), boolToInt(w.IsNegative),
		mustJSON(w.PairBlacklist), w.CurationLevel, w.NamePenalty,
		mustJSON(w.NamingCats),
	}
}

// wordJSON word.json 的磁盘结构
type wordJSON struct {
	Word        string `json:"word"`
	Explanation string `json:"explanation"`
}

// importWords 导入词典释义（取多字词的首字，首见即锁）
func (s *Store) importWords(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var entries []wordJSON
	if err := json.Unmarshal(raw, &entries); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = fmt.Sprintf("%d-entries", len(entries))

	// 首见即锁：按源文件顺序取首字，命中即不再覆盖（与运行时 hanzi.LoadWordData 同语义）
	seen := make(map[string]bool)
	items := make([]schema.Meaning, 0, len(entries))
	for _, e := range entries {
		w := strings.TrimSpace(e.Word)
		if w == "" {
			continue
		}
		r := []rune(w)
		if len(r) == 0 {
			continue
		}
		char := string(r[0])
		if seen[char] {
			continue
		}
		exp := normalize.Meaning(e.Explanation)
		if exp == "" {
			continue
		}
		seen[char] = true
		items = append(items, schema.Meaning{Char: char, Explanation: exp})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Char < items[j].Char })

	var errs validate.Errors
	for _, m := range items {
		errs = append(errs, validate.Meaning(def.Name, m)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare("INSERT INTO words(char, explanation) VALUES(?,?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, m := range items {
			if _, err := stmt.Exec(m.Char, m.Explanation); err != nil {
				return 0, fmt.Errorf("写入释义 %s 失败: %w", m.Char, err)
			}
		}
		return len(items), nil
	})
}

// freqJSON 人名频率的磁盘结构
type freqJSON struct {
	Meta struct {
		Source      string `json:"source"`
		TotalNames  int    `json:"total_names"`
		TotalChars  int    `json:"total_chars"`
		UniqueChars int    `json:"unique_chars"`
		BuiltAt     string `json:"built_at"`
	} `json:"meta"`
	Frequency []hanzi.CharFrequency `json:"frequency"`
}

// importCharFreq 导入单字人名频率
func (s *Store) importCharFreq(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var fj freqJSON
	if err := json.Unmarshal(raw, &fj); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = fj.Meta.BuiltAt

	items := make([]schema.CharFreq, 0, len(fj.Frequency))
	for _, f := range fj.Frequency {
		items = append(items, normalize.CharFreq(schema.CharFreq{
			Char: f.Char, Count: f.Count, Rank: f.Rank, Tier: f.Tier,
		}))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Char < items[j].Char })

	var errs validate.Errors
	for _, c := range items {
		errs = append(errs, validate.CharFreq(def.Name, c)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare("INSERT INTO name_frequency(char, count, rank, tier) VALUES(?,?,?,?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, c := range items {
			if _, err := stmt.Exec(c.Char, c.Count, c.Rank, c.Tier); err != nil {
				return 0, fmt.Errorf("写入单字频率 %s 失败: %w", c.Char, err)
			}
		}

		// 频率档回填（docs/31 §9.6 步骤4 / ADR-009）：namer.json 不含 nameFreqTier 真值，
		// JSON 运行时路径靠 GetCharTier 事后合并；库路径必须由频率物料在导入期
		// 把档位合并进 hanzi_data，落库即规范态，运行时读库无需再合并。
		// 回填口径与 JSON 路径一致：仅 Tier ≥ 2（LoadFrequencyDB 的过滤口径）。
		// 先清零再回填，保证重导入幂等（ADR-011）。
		if _, err := tx.Exec("UPDATE hanzi_data SET name_freq_tier = 0"); err != nil {
			return 0, fmt.Errorf("清零人名频率档位失败: %w", err)
		}
		upd, err := tx.Prepare("UPDATE hanzi_data SET name_freq_tier = ? WHERE char = ?")
		if err != nil {
			return 0, err
		}
		defer func() { _ = upd.Close() }()
		for _, c := range items {
			if c.Tier < 2 {
				continue
			}
			if _, err := upd.Exec(c.Tier, c.Char); err != nil {
				return 0, fmt.Errorf("回填人名频率档位 %s 失败: %w", c.Char, err)
			}
		}
		return len(items), nil
	})
}

// bigramJSON 双字频率的磁盘结构
type bigramJSON struct {
	Meta struct {
		Source  string `json:"source"`
		BuiltAt string `json:"built_at"`
	} `json:"meta"`
	Frequency []hanzi.BigramFrequency `json:"frequency"`
}

// importBigram 导入双字人名频率
func (s *Store) importBigram(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var bj bigramJSON
	if err := json.Unmarshal(raw, &bj); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = bj.Meta.BuiltAt

	items := make([]schema.BigramFreq, 0, len(bj.Frequency))
	for _, f := range bj.Frequency {
		items = append(items, normalize.BigramFreq(schema.BigramFreq{
			Char1: f.Char1, Char2: f.Char2, Count: f.Count, Rank: f.Rank, Tier: f.Tier,
		}))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Key() < items[j].Key() })

	var errs validate.Errors
	for _, b := range items {
		errs = append(errs, validate.BigramFreq(def.Name, b)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare("INSERT INTO name_bigram(key, char1, char2, count, rank, tier) VALUES(?,?,?,?,?,?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, b := range items {
			if _, err := stmt.Exec(b.Key(), b.Char1, b.Char2, b.Count, b.Rank, b.Tier); err != nil {
				return 0, fmt.Errorf("写入双字频率 %s 失败: %w", b.Key(), err)
			}
		}
		return len(items), nil
	})
}

// importGate 导入命名质量门禁字表
func (s *Store) importGate(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	chars, err := readStringSlice(src)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Version = fmt.Sprintf("%d-chars", len(chars))

	items := normalize.Tags(chars)
	var errs validate.Errors
	for _, c := range items {
		errs = append(errs, validate.Rune(def.Name, "char", c)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare("INSERT INTO naming_gate(char) VALUES(?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, c := range items {
			if _, err := stmt.Exec(c); err != nil {
				return 0, fmt.Errorf("写入门禁字 %s 失败: %w", c, err)
			}
		}
		return len(items), nil
	})
}

// importForbidden 导入禁忌双字组合
func (s *Store) importForbidden(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	combos, err := readStringSlice(src)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Version = fmt.Sprintf("%d-combos", len(combos))

	items := normalize.Tags(combos)
	var errs validate.Errors
	for _, c := range items {
		errs = append(errs, validate.Combo(def.Name, c)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare("INSERT INTO forbidden_combos(combo) VALUES(?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, c := range items {
			if _, err := stmt.Exec(c); err != nil {
				return 0, fmt.Errorf("写入禁忌组合 %s 失败: %w", c, err)
			}
		}
		return len(items), nil
	})
}

// importAllowlist 导入显式白名单（字 / 组合 / 祈福组合），docs/33 门禁层 M1
//
// 源格式：{"version":"...","chars":[...],"combos":[...],"blessings":[...]}
// 落库：allowlist(entry PK, kind)，kind ∈ char/combo/blessing（白名单优先于黑名单）。
func (s *Store) importAllowlist(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var doc struct {
		Version   string   `json:"version"`
		Chars     []string `json:"chars"`
		Combos    []string `json:"combos"`
		Blessings []string `json:"blessings"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = doc.Version

	type allowRow struct {
		entry string
		kind  string
	}
	rows := make([]allowRow, 0, len(doc.Chars)+len(doc.Combos)+len(doc.Blessings))
	var errs validate.Errors
	for _, c := range normalize.Tags(doc.Chars) {
		errs = append(errs, validate.Rune(def.Name, "char", c)...)
		rows = append(rows, allowRow{entry: c, kind: schema.AllowKindChar})
	}
	for _, c := range normalize.Tags(doc.Combos) {
		errs = append(errs, validate.Combo(def.Name, c)...)
		rows = append(rows, allowRow{entry: c, kind: schema.AllowKindCombo})
	}
	for _, c := range normalize.Tags(doc.Blessings) {
		errs = append(errs, validate.Combo(def.Name, c)...)
		rows = append(rows, allowRow{entry: c, kind: schema.AllowKindBlessing})
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare("INSERT INTO allowlist(entry, kind) VALUES(?,?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		seen := make(map[string]bool, len(rows))
		n := 0
		for _, r := range rows {
			if r.entry == "" || seen[r.entry] {
				continue
			}
			seen[r.entry] = true
			if _, err := stmt.Exec(r.entry, r.kind); err != nil {
				return 0, fmt.Errorf("写入白名单 %s 失败: %w", r.entry, err)
			}
			n++
		}
		return n, nil
	})
}

// importYijing 导入易经六十四卦
func (s *Store) importYijing(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var list []yijing.Hexagram
	if err := json.Unmarshal(raw, &list); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = fmt.Sprintf("%d-hexagrams", len(list))

	items := make([]schema.Hexagram, 0, len(list))
	for _, h := range list {
		items = append(items, schema.Hexagram{
			ID: h.ID, Name: h.Name, Number: h.Number, Symbol: h.Symbol,
			UpperTrigram: h.UpperTrigram, LowerTrigram: h.LowerTrigram,
			GuaCi: h.GuaCi, XiangCi: h.XiangCi, YaoCi: normalize.Tags(h.YaoCi),
			Interpretation: h.Interpretation,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	seen := make(map[int]bool, len(items))
	var errs validate.Errors
	for _, h := range items {
		errs = append(errs, validate.Hexagram(def.Name, h, seen)...)
		seen[h.ID] = true
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare(`INSERT INTO yijing_hexagrams(id, name, number, symbol,
			upper_trigram, lower_trigram, gua_ci, xiang_ci, yao_ci, interpretation)
			VALUES(?,?,?,?,?,?,?,?,?,?)`)
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, h := range items {
			if _, err := stmt.Exec(h.ID, h.Name, h.Number, h.Symbol, h.UpperTrigram,
				h.LowerTrigram, h.GuaCi, h.XiangCi, mustJSON(h.YaoCi), h.Interpretation); err != nil {
				return 0, fmt.Errorf("写入卦象 %d 失败: %w", h.ID, err)
			}
		}
		return len(items), nil
	})
}

// importZodiac 导入十二生肖
func (s *Store) importZodiac(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var list []zodiac.Zodiac
	if err := json.Unmarshal(raw, &list); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = fmt.Sprintf("%d-zodiacs", len(list))

	items := make([]schema.Zodiac, 0, len(list))
	for _, z := range list {
		items = append(items, schema.Zodiac{
			ID: z.ID, Name: z.Name, Wuxing: normalize.Wuxing(z.Wuxing),
			Compatible: normalize.Tags(z.Compatible), Conflicting: normalize.Tags(z.Conflicting),
			AvoidChars: strings.TrimSpace(z.AvoidChars), LuckyNumbers: z.LuckyNumbers,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	var errs validate.Errors
	for _, z := range items {
		errs = append(errs, validate.Zodiac(def.Name, z)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare(`INSERT INTO zodiac(id, name, wuxing, compatible,
			conflicting, avoid_chars, lucky_numbers) VALUES(?,?,?,?,?,?,?)`)
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, z := range items {
			if _, err := stmt.Exec(z.ID, z.Name, z.Wuxing, mustJSON(z.Compatible),
				mustJSON(z.Conflicting), z.AvoidChars, mustJSON(z.LuckyNumbers)); err != nil {
				return 0, fmt.Errorf("写入生肖 %d 失败: %w", z.ID, err)
			}
		}
		return len(items), nil
	})
}

// readStringSlice 读取一个 JSON 字符串数组文件
func readStringSlice(src string) ([]string, error) {
	raw, err := os.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("读取 %s 失败: %w", src, err)
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", src, err)
	}
	return out, nil
}
