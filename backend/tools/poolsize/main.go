package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"name/internal/application/services"
	"name/internal/domain/classics"
	"name/internal/domain/fate"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/data"
)

func main() {
	if err := data.Init("../../data"); err != nil {
		fmt.Println("data.Init:", err)
	}
	services.SyncNamingIndexFromHanzi()
	cache.Init()
	provider := &services.HanziDataProvider{}
	analyzer := services.NewBaziAnalyzerAdapter()

	born := time.Date(2024, 5, 20, 10, 0, 0, 0, time.UTC)
	gender := "male"
	surname := "张"
	fmt.Println("=== 分寸诊断 ===", surname, gender, born.Format("2006-01-02 15:04"), "日主:", string(rune('?')))

	fd, err := analyzer.Analyze(born, fate.Gender(gender))
	if err != nil {
		fmt.Println("八字分析失败:", err)
		return
	}
	fmt.Println("日主五行:", fd.WuXingXiji.RiZhuWuXing, "强弱:", fd.WuXingXiji.QiangRuo)
	fmt.Println("喜用神:", fd.WuXingXiji.XiYongShen, "忌神:", fd.WuXingXiji.Ji)

	fo := fate.NewFilterOption().WithGenderFilter(gender).WithStrictness("moderate")
	filter := fo.Build()

	var q fate.CharacterQuery = fate.NewBasicCharacterQuery()
	q = filter.QueryFilter(q)
	all, _ := provider.FindCharacters(q)
	fmt.Printf("--- 候选池各阶段 ---\n1. 全库字(filter.QueryFilter 后): %d\n", len(all))

	valid := make([]*fate.Character, 0, len(all))
	for _, c := range all {
		if fate.IsHardNegativeChar(c.Char) || c.IsNegative {
			continue
		}
		if fate.IsSensitiveChar(c.Char) {
			continue
		}
		if filter.CheckCharacter(c) {
			valid = append(valid, c)
		}
	}
	fmt.Printf("2. 通过 CheckCharacter(性别/禁用字/负面语义): %d\n", len(valid))

	// —— 扩池信号量化：各"好字"信号在 validChars 中的覆盖 ——
	sigCurated := 0
	sigPoetry := 0
	sigCuratedOrPoetry := 0
	sigAny := 0
	sigPoetryLevel1 := 0
	sigPoetryLevel2 := 0
	sigFreq1 := 0
	sigFreqPoetryL1 := 0
	sigFreqPoetryL2 := 0
	for _, c := range valid {
		isC := c.IsCurated
		p, _, _ := classics.FindPoetryByChars(c.Char)
		if isC {
			sigCurated++
		}
		if p {
			sigPoetry++
		}
		if p && c.CommonLevel <= 1 {
			sigPoetryLevel1++
		}
		if p && c.CommonLevel <= 2 {
			sigPoetryLevel2++
		}
		if c.NameFreqTier >= 1 {
			sigFreq1++
		}
		if p && c.NameFreqTier >= 1 && c.CommonLevel <= 1 {
			sigFreqPoetryL1++
		}
		if c.NameFreqTier >= 1 && (c.CommonLevel <= 2 || p) {
			sigFreqPoetryL2++
		}
		if isC || p {
			sigCuratedOrPoetry++
		}
		if isC || p || c.PositiveScore >= 85 {
			sigAny++
		}
	}
	fmt.Printf("扩池信号统计(validChars %d 中):\n", len(valid))
	fmt.Printf("  A. 策展覆盖表 IsCurated: %d\n", sigCurated)
	fmt.Printf("  B. 诗词出典字: %d\n", sigPoetry)
	fmt.Printf("  B1. 诗词出典∩一级字: %d\n", sigPoetryLevel1)
	fmt.Printf("  B2. 诗词出典∩一二级字: %d\n", sigPoetryLevel2)
	fmt.Printf("  E. 人名频率Tier>=1: %d\n", sigFreq1)
	fmt.Printf("  F. 诗歌出典∩人名频率∩一级: %d\n", sigFreqPoetryL1)
	fmt.Printf("  G. 人频率∩(二级∪诗歌): %d\n", sigFreqPoetryL2)
	fmt.Printf("  C. 策展∪诗词: %d\n", sigCuratedOrPoetry)
	fmt.Printf("  D. 策展∪诗词∪评分>=85: %d\n", sigAny)

	curated := make([]*fate.Character, 0, len(valid))
	for _, c := range valid {
		if c.PositiveScore >= 85 {
			curated = append(curated, c)
		}
	}
	fmt.Printf("3. 策展白名单(PositiveScore>=85): %d\n", len(curated))

	xiSet := map[string]bool{}
	for _, wx := range fd.WuXingXiji.XiYongShen {
		xiSet[wx] = true
	}
	sheng := map[string]string{"木": "火", "火": "土", "土": "金", "金": "水", "水": "木"}
	for wx := range xiSet {
		for src, dst := range sheng {
			if dst == wx {
				xiSet[src] = true
			}
		}
	}
	narrowed := make([]*fate.Character, 0, len(curated))
	for _, c := range curated {
		if xiSet[c.WuXing] {
			narrowed = append(narrowed, c)
		}
	}
	pool := curated
	if len(narrowed) >= 80 {
		pool = narrowed
		fmt.Printf("4. 五行收窄(喜+生助 %v): %d (已采用)\n", sortedKeys(xiSet), len(pool))
	} else {
		fmt.Printf("4. 五行收窄结果 %d <80, 降级保留策展白名单 %d\n", len(narrowed), len(pool))
	}

	fmt.Printf("5. 最终候选池: %d 字 -> 双名笛卡尔积组合 %d 个\n", len(pool), len(pool)*len(pool))

	wxCount := map[string]int{}
	for _, c := range pool {
		wxCount[c.WuXing]++
	}
	keys := make([]string, 0, len(wxCount))
	for k := range wxCount {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, wxCount[k]))
	}
	fmt.Println("候选池五行分布:", strings.Join(parts, " "))

	sort.SliceStable(pool, func(i, j int) bool { return pool[i].PositiveScore > pool[j].PositiveScore })
	fmt.Println("--- 候选池 Top 40 字 (PositiveScore 降序) ---")
	for i, c := range pool {
		if i >= 40 {
			break
		}
		fmt.Printf("%2d. %s 五行=%s PositiveScore=%d 策展=%v\n", i+1, c.Char, c.WuXing, c.PositiveScore, c.IsCurated)
	}

	fmt.Println("--- 引擎实际生成 Top 30(定 seed 无随机) ---")
	engine := fate.NewEngine(provider, analyzer, fate.DefaultRaters())
	sess := engine.NewSessionWithFilter(filter)
	input := fate.Input{Surname: surname, Born: born,
		Gender:  fate.Gender(gender),
		Options: fate.GenerateOptions{Count: 100, NameLength: 2}}

	fmt.Println("--- 荒谬字词典(级/频率/诗词, 若三者非空则漏入扩展池) ---")
	for _, ch := range []string{"拤", "饹", "婊", "蚂", "蛞", "啰", "苯", "羟", "滃"} {
		found, desc, _ := classics.FindPoetryByChars(ch)
		var lvl, tier int
		if v, err := provider.GetCharacter(ch); err == nil && v != nil {
			lvl, tier = v.CommonLevel, v.NameFreqTier
		}
		fmt.Printf("  %s: level=%d tier=%d 诗词=%v %s\n", ch, lvl, tier, found, desc)
	}
	if err := sess.Start(context.Background(), &input); err != nil {
		fmt.Println("会话启动失败:", err)
		return
	}
	_ = sess.Wait() // 性能开关：不枚举全笛卡尔，仅确认会话可启动（池规模统计已在上面完成）
	out := sess.Result()
	charCount := map[string]int{}
	for i, n := range out.TopNames {
		if i >= 30 {
			break
		}
		fmt.Printf("%2d. %s (评分 %v)\n", i+1, surname+n.GivenName, n.Score)
		for _, rune_ := range n.GivenName {
			charCount[string(rune_)]++
		}
	}
	fmt.Println("--- Top30 名字用字频次 ---")
	type kv struct {
		k string
		v int
	}
	kvs := make([]kv, 0, len(charCount))
	for k, v := range charCount {
		kvs = append(kvs, kv{k, v})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].v > kvs[j].v })
	for _, e := range kvs {
		fmt.Printf("%s=%d  ", e.k, e.v)
	}
	fmt.Println()
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
