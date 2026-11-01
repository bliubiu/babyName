package fate

import (
	"fmt"
	"strings"
	"sync"

	"name/internal/domain/yijing"
)

// ============================================================================
// 卦象体用评分（docs/42 P3-12：卦象匹配分计入总分）
//
// 此前姓名卦只对首个推荐名起卦、纯展示不计分（name_service.calculateHexagramAndZiweiParallel）；
// 本评分器把体用断卦结果映射为分值，使「姓+名」的梅花卦象成为综合评分的一维。
//
// 口径统一（硬约束，防双口径漂移）：
//   - 起卦入口唯一：yijing.GetMeihuaDetailByStrokes（姓%8上卦、名总%8下卦、
//     和%6动爻，docs/42 P0-1/P0-5）；
//   - 笔画口径与结果页姓名卦一致（康熙优先）：候选携带 KangxiStroke1/2
//     （kangxiStrokeOf：未收录降级科学笔画，与 GivenNameStrokes 同规则），
//     姓氏康熙笔画经 FateData.SurnameStrokes 注入（engine.generate 查
//     GetSurnameStrokes，测名路径 RateGivenName 幂等补查）；
//   - 体用断卦复用 yijing.AnalyzeTiYong（动者为用，docs/42 P1-③）。
//
// 打分映射（基准 ± 卦气旺衰微调）：
//   用生体/比和 90；体克用/体生用 70；用克体 45；
//   体卦旺/相 +5，休 0，囚/死 -5（卦气旺衰 = 体卦得令与否）。
//   喜用神匹配不在此维度重复计分（五行八字维度已覆盖）。
//
// 性能：姓名卦只由（姓笔画, 名总笔画, 季节）决定，与候选具体用字无关，
// 包级缓存以三元组为键：130 万候选枚举热路径每次评分仅一次缓存查询，
// 未命中的起卦+断卦计算量每请求至多 ≈ 名总笔画取值数（数十次）。
// ============================================================================

const (
	// guaXiangNeutral 卦象信息不足（无姓氏笔画/笔画缺失/卦库未就绪）时的中平分
	guaXiangNeutral = 60.0
	// guaXiangCacheLimit 卦象缓存上限；超限整体重置，防内存无界增长
	// （记忆约束：所有缓存必须限容）
	guaXiangCacheLimit = 8192
)

// GuaXiangRater 卦象体用评分器（数理层）
type GuaXiangRater struct {
	weight float64
}

func NewGuaXiangRater() *GuaXiangRater {
	return &GuaXiangRater{weight: 0.10}
}

func NewGuaXiangRaterWithWeight(w float64) *GuaXiangRater {
	return &GuaXiangRater{weight: w}
}

func (r *GuaXiangRater) Name() string    { return "卦象" }
func (r *GuaXiangRater) Weight() float64 { return r.weight }

// kangxiStrokeOf 候选字康熙笔画：未收录降级科学笔画（与 GivenNameStrokes 同规则）
func kangxiStrokeOf(c *Character) int {
	if c.KangxiStroke > 0 {
		return c.KangxiStroke
	}
	return c.ScienceStroke
}

// guaXiangKey 卦象缓存键：姓名卦只由这三个量决定
type guaXiangKey struct {
	surnameStrokes int
	nameTotal      int
	season         string
}

// guaXiangVerdict 起卦+体用断卦的提取结论。
// 不缓存喜用神相关文本，保证缓存条目跨请求复用安全（喜用神因人而异）。
type guaXiangVerdict struct {
	tiGua, tiWuxing     string
	yongGua, yongWuxing string
	relation            string // 用生体/比和/体克用/体生用/用克体
	verdict             string // 吉/中/凶
	wangShuai           string // 旺/相/休/囚/死（无季节输入时空）
}

var guaXiangCache = struct {
	sync.RWMutex
	m map[guaXiangKey]guaXiangVerdict
}{m: make(map[guaXiangKey]guaXiangVerdict)}

// tiYongForName 起卦并提取体用结论（带缓存）；ok=false 表示无法起卦
func tiYongForName(surnameStrokes, nameTotal int, season string) (guaXiangVerdict, bool) {
	key := guaXiangKey{surnameStrokes: surnameStrokes, nameTotal: nameTotal, season: season}
	guaXiangCache.RLock()
	v, ok := guaXiangCache.m[key]
	guaXiangCache.RUnlock()
	if ok {
		return v, true
	}

	d := yijing.GetMeihuaDetailByStrokes(surnameStrokes, nameTotal)
	if d == nil {
		return guaXiangVerdict{}, false
	}
	// 喜用神传 nil：卦象分只断体用生克与旺衰，喜用匹配由五行八字维度表达
	a := yijing.AnalyzeTiYong(d.UpperTrigramID, d.LowerTrigramID, d.MovingYao, season, nil)
	if a == nil {
		return guaXiangVerdict{}, false
	}
	v = guaXiangVerdict{
		tiGua: a.TiGua, tiWuxing: a.TiWuxing,
		yongGua: a.YongGua, yongWuxing: a.YongWuxing,
		relation: a.Relation, verdict: a.Verdict, wangShuai: a.WangShuai,
	}

	guaXiangCache.Lock()
	if len(guaXiangCache.m) >= guaXiangCacheLimit {
		guaXiangCache.m = make(map[guaXiangKey]guaXiangVerdict, guaXiangCacheLimit/2)
	}
	guaXiangCache.m[key] = v
	guaXiangCache.Unlock()
	return v, true
}

// guaXiangScore 体用关系 × 卦气旺衰 → 分值映射（纯函数，单测钉住）
func guaXiangScore(relation, wangShuai string) float64 {
	var base float64
	switch relation {
	case "用生体", "比和":
		base = 90
	case "体克用", "体生用":
		base = 70
	case "用克体":
		base = 45
	default:
		return guaXiangNeutral
	}
	switch wangShuai {
	case "旺", "相":
		base += 5
	case "囚", "死":
		base -= 5
	}
	return base
}

func (r *GuaXiangRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	if fateData == nil || fateData.SurnameStrokes <= 0 {
		return NameRating{Score: guaXiangNeutral, Detail: "姓氏笔画未注入，卦象按中平计"}
	}
	nameTotal := candidate.KangxiStroke1 + candidate.KangxiStroke2
	if nameTotal <= 0 {
		return NameRating{Score: guaXiangNeutral, Detail: "名字笔画缺失，卦象按中平计"}
	}

	v, ok := tiYongForName(fateData.SurnameStrokes, nameTotal, fateData.BirthSeason)
	if !ok {
		return NameRating{Score: guaXiangNeutral, Detail: "卦库未就绪，卦象按中平计"}
	}

	score := guaXiangScore(v.relation, v.wangShuai)
	if candidate.skipDetail {
		return NameRating{Score: score}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "体卦%s（%s），用卦%s（%s），%s（%s）",
		v.tiGua, v.tiWuxing, v.yongGua, v.yongWuxing, v.relation, v.verdict)
	if v.wangShuai != "" {
		fmt.Fprintf(&b, "；%s月体卦「%s」", fateData.BirthSeason, v.wangShuai)
	}
	return NameRating{Score: score, Detail: b.String()}
}
