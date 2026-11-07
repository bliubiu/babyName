package services

import (
	"fmt"
	"time"

	"name/internal/domain/bazi"
	"name/internal/domain/fate"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/database"
)

// BaziAnalyzerAdapter 基于 domain/bazi 的 BaziAnalyzer 实现
type BaziAnalyzerAdapter struct{}

func NewBaziAnalyzerAdapter() *BaziAnalyzerAdapter {
	return &BaziAnalyzerAdapter{}
}

func (a *BaziAnalyzerAdapter) Analyze(born time.Time, gender fate.Gender) (*fate.FateData, error) {
	year, month, day := born.Date()
	hour := born.Hour()
	minute := born.Minute()

	baziResult, err := bazi.AnalyzeBazi(year, int(month), day, hour, minute)
	if err != nil {
		return nil, err
	}

	zodiacName := zodiac.GetZodiacByYear(year).Name
	fourPillars := baziGanzhiToArray(baziResult.Bazi)

	// 喜用神单一真源（docs/43 P1-3）：以 bazi weighted 算法（calculateXiyongshenWeighted）
	// 为唯一权威——喜用神列表、强弱、忌仇全部同源于本次 AnalyzeBazi 结果。
	// 此前此处另调 fate.BalanceXiYongJi（total/2 简单多数判强弱）再与
	// bazi 的 DayMasterStrength 混装，同一响应内喜用神与强弱栏可能自相矛盾；
	// 测名路径（fate_name_service.classicXiyongShen）早已统一为经典口径，
	// 本适配器是最后一块，收敛后 /generate、/generate/analysis、测名三链路同源。
	xiYongShen := baziResult.Xiyongshen
	yongShen := ""
	xi := ""
	if len(xiYongShen) > 0 {
		yongShen = xiYongShen[0]
		xi = xiYongShen[0]
	}
	if len(xiYongShen) > 1 {
		xi = xiYongShen[1]
	}
	// 忌/仇按同一强弱结论推导（weighted 算法无忌仇概念，扶抑法补齐）
	ji, chou := bazi.DeriveJiChou(baziResult.RishouWuxing, baziResult.DayMasterStrength)

	// 构建 FateData
	fateData := &fate.FateData{
		BaziInfo: fate.BaziInfo{
			FourPillars: fourPillars,
			Zodiac:      zodiacName,
		},
		WuXingXiji: fate.WuXingXiji{
			XiYongShen:  xiYongShen,
			YongShen:    yongShen,
			Xi:          xi,
			Ji:          ji,
			Chou:        chou,
			RiZhu:       baziResult.Rishou,
			RiZhuWuXing: baziResult.RishouWuxing,
			QiangRuo:    baziResult.DayMasterStrength,
		},
		TiaoHouYongShen: joinStrings(baziResult.TiaohouShen, "、"),
		// 出生季节（月支节气月）：卦气旺衰依据（docs/42 P3-12 卦象评分）
		BirthSeason: baziResult.Season,
	}

	// 设置四柱五行和纳音
	// bazi.BaziAnalysis 没有直接提供四柱五行/纳音数组，
	// 通过四个天干推算
	tiangan := []rune(baziResult.Bazi.YearGanzhi + baziResult.Bazi.MonthGanzhi + baziResult.Bazi.DayGanzhi + baziResult.Bazi.HourGanzhi)
	for i, tg := range tiangan {
		if i%2 == 0 { // 天干
			if i/2 < len(fateData.BaziInfo.WuXing) {
				fateData.BaziInfo.WuXing[i/2] = bazi.WuxingMap[string(tg)]
			}
		}
	}

	// 四柱纳音
	if four := baziResult.FourNayin; four != nil {
		fateData.BaziInfo.NaYin = [4]string{
			four.Year.Nayin, four.Month.Nayin, four.Day.Nayin, four.Hour.Nayin,
		}
	} else {
		fateData.BaziInfo.NaYin[0] = baziResult.Nayin
	}

	return fateData, nil
}

// baziGanzhiToArray 将 Bazi 的四个柱转为 [4]string
func baziGanzhiToArray(b bazi.Bazi) [4]string {
	return [4]string{b.Year, b.Month, b.Day, b.Hour}
}

// joinStrings 拼接字符串切片
func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for _, v := range s[1:] {
		result += sep + v
	}
	return result
}

// --- 接口守卫（编译时检查） ---
// SQL 失败时返回非 nil error，由调用方降级到 Go 端全表过滤。
func (p *HanziDataProvider) queryViaSQL(pat filterPattern) ([]database.Hanzi, error) {
	if p.sqlFilter == nil {
		return nil, fmt.Errorf("sqlFilter 未注入")
	}

	// 多值 wuxingIn 时 SQL 端仅支持单值等值匹配，不下推，交由下方 Go 端二次过滤。
	wuxing := ""
	if len(pat.wuxingIn) == 1 {
		wuxing = pat.wuxingIn[0]
	}
	hanzis, err := p.sqlFilter.SearchHanziByFilter(
		wuxing, pat.strokeGTE, pat.strokeLTE,
		false, false, pat.charIn, 10000,
	)
	if err != nil {
		return nil, fmt.Errorf("SQL 汉字筛选失败: %w", err)
	}
	if len(hanzis) == 0 {
		// 查询成功且确实无匹配 —— 与上面的失败路径必须可区分（docs/29 B8）
		return nil, nil
	}

	// 二次过滤：与 FindCharacters 的 pattern 匹配逐项一致，但字段全部取自 SQL 行。
	filtered := make([]database.Hanzi, 0, len(hanzis))
	for i := range hanzis {
		h := &hanzis[i]
		c := databaseHanziToCharacter(h)

		if pat.regularFilter && !c.IsRegular {
			continue
		}
		if pat.nameableFilter && !c.IsNameable {
			continue
		}
		if pat.strokeEQ > 0 && c.ScienceStroke != pat.strokeEQ {
			continue
		}
		if pat.strokeGTE > 0 && c.ScienceStroke < pat.strokeGTE {
			continue
		}
		if pat.strokeLTE > 0 && c.ScienceStroke > pat.strokeLTE {
			continue
		}
		if len(pat.wuxingIn) > 0 && !inSlice(c.WuXing, pat.wuxingIn) {
			continue
		}
		if len(pat.wuxingNotIn) > 0 && inSlice(c.WuXing, pat.wuxingNotIn) {
			continue
		}
		if len(pat.charIn) > 0 && !inSlice(c.Char, pat.charIn) {
			continue
		}
		if pat.genderHint != "" && c.GenderHint != pat.genderHint && c.GenderHint != "neutral" {
			continue
		}
		if pat.namingCategory != "" && !inSlice(pat.namingCategory, c.NamingCategory) {
			continue
		}
		filtered = append(filtered, *h)
	}
	return filtered, nil
}

// SQLiteCharStore 最小接口（避免在 services 包 import sqlite 包造成循环依赖）
//
// sqlite.Store 已实现 SearchHanziByFilter(HanziFilter) []*database.Hanzi，
// 但参数类型 HanziFilter 在 sqlite 包内部定义。本接口用更宽松的签名
// 桥接，使 services 包只需 database.Hanzi 数据 + 基础过滤条件。
type SQLiteCharStore interface {
	SearchHanziByFilter(wuxing string, minStrokes, maxStrokes int,
		hasPositive, isRegular bool, chars []string, limit int) ([]*database.Hanzi, error)
	// HanziRowCount 返回 hanzi_data 行数，供 HasData 甄别「库无数据」
	// 与「有数据但无匹配」（空库回落降级的判据）
	HanziRowCount() (int, error)
}

// sqliteHanziFilterAdapter 桥接实现：把 SQLHanziFilter 调用转给 sqlite.Store
//
// 为什么不直接 import sqlite 包：services 包已被 application/handlers 引用，
// 而 cmd/server 同时引用 services + sqlite，加 import 会让 services 反向依赖
// infrastructure/database/sqlite，破坏分层。
// 解法：services 定义接口 SQLiteCharStore（够用即可），由调用方注入实现。
type sqliteHanziFilterAdapter struct {
	store SQLiteCharStore
}

// NewSQLiteHanziFilter 创建基于 sqlite.Store 的过滤适配器
//
// 装配示例（cmd/server/main.go）：
//
//	filter := services.NewSQLiteHanziFilter(store)
//	provider := &services.HanziDataProvider{}
//	provider.SetSQLFilter(filter)
//
// 运行时：FindCharacters 优先走 SQL + 索引，失败降级到 Go 端全表过滤。
func NewSQLiteHanziFilter(store SQLiteCharStore) SQLHanziFilter {
	return &sqliteHanziFilterAdapter{store: store}
}

func (a *sqliteHanziFilterAdapter) SearchHanziByFilter(
	wuxing string, minStrokes, maxStrokes int,
	hasPositive, isRegular bool, chars []string, limit int,
) ([]database.Hanzi, error) {
	if a.store == nil {
		return nil, fmt.Errorf("底层 store 未注入")
	}
	// 调底层 store 拿到 []*database.Hanzi，转 []database.Hanzi 返回
	raw, err := a.store.SearchHanziByFilter(
		wuxing, minStrokes, maxStrokes,
		hasPositive, isRegular, chars, limit,
	)
	if err != nil {
		// 错误必须上抛，让 FindCharacters 走 Go 端降级路径（docs/29 B8）
		return nil, err
	}
	result := make([]database.Hanzi, 0, len(raw))
	for i := range raw {
		result = append(result, *raw[i])
	}
	return result, nil
}

// HasData 底层 hanzi_data 是否有行
//
// 探测失败（如库损坏）时保守返回 true：让调用方把空结果当作
// 「确实无匹配」如实返回，而不是借一次探测失败就降级 —— 探测本身
// 不应成为新的故障面。真正的查询失败仍由 SearchHanziByFilter 的
// error 路径负责降级（docs/29 B8）。
func (a *sqliteHanziFilterAdapter) HasData() bool {
	if a.store == nil {
		return false
	}
	n, err := a.store.HanziRowCount()
	if err != nil {
		return true
	}
	return n > 0
}

// 接口守卫：sqliteHanziFilterAdapter 必须实现 SQLHanziFilter
var _ SQLHanziFilter = (*sqliteHanziFilterAdapter)(nil)

// HasSQLiteCharStore 报告任意 store 是否实现了 SQLiteCharStore 接口（用于类型断言）
//
// 调用方在装配时不必关心 store 具体类型，通过此 helper 自动识别：
//
//	if services.HasSQLiteCharStore(store) {
//	    filter := services.NewSQLiteHanziFilter(store.(services.SQLiteCharStore))
//	    provider.SetSQLFilter(filter)
//	}
func HasSQLiteCharStore(store database.Store) bool {
	_, ok := store.(SQLiteCharStore)
	return ok
}

// AsSQLiteCharStore 桥接 database.Store 到 SQLiteCharStore（仅在 HasSQLiteCharStore 返回 true 时安全）
//
// 返回值供 NewSQLiteHanziFilter 消费。type assertion 失败时返回 nil。
func AsSQLiteCharStore(store database.Store) SQLiteCharStore {
	if s, ok := store.(SQLiteCharStore); ok {
		return s
	}
	return nil
}
