package schema

import (
	"reflect"
	"testing"
)

// 本文件守护 docs/31 ADR-009：规范基座与受控词表是离线/在线唯一的真源，
// 物料注册表与拓扑排序决定导入顺序的确定性。

func TestValidGender(t *testing.T) {
	for _, g := range []Gender{Male, Female, Neutral} {
		if !ValidGender(g) {
			t.Errorf("性别 %q 应判合法", g)
		}
	}
	for _, g := range []Gender{"", "unknown", "男", "MALE"} {
		if ValidGender(g) {
			t.Errorf("性别 %q 应判非法", g)
		}
	}
}

func TestValidWuxing(t *testing.T) {
	for _, w := range []string{"金", "木", "水", "火", "土", ""} {
		if !ValidWuxing(w) {
			t.Errorf("五行 %q 应判合法", w)
		}
	}
	for _, w := range []string{"气", "金 ", "metal", "土木"} {
		if ValidWuxing(w) {
			t.Errorf("五行 %q 应判非法", w)
		}
	}
}

func TestNormalizeTags(t *testing.T) {
	// 空输入必须返回非 nil 空切片：JSON 序列化稳定为 [] 而不是 null
	if got := NormalizeTags(nil); got == nil || len(got) != 0 {
		t.Fatalf("空输入期望非 nil 空切片，实得 %#v", got)
	}
	if got := NormalizeTags([]string{}); got == nil || len(got) != 0 {
		t.Fatalf("空切片输入期望非 nil 空切片，实得 %#v", got)
	}

	got := NormalizeTags([]string{" 古典 ", "古典", "", "  ", "大气", "古典"})
	want := []string{"古典", "大气"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("归一后期望 %v，实得 %v", want, got)
	}

	// 幂等：归一后的结果再次归一必须不变
	if again := NormalizeTags(got); !reflect.DeepEqual(again, got) {
		t.Fatalf("归一不幂等：%v → %v", got, again)
	}
}

func TestBigramFreqKey(t *testing.T) {
	b := BigramFreq{Char1: "司", Char2: "马"}
	if got := b.Key(); got != "司马" {
		t.Fatalf("Key() 期望「司马」，实得 %q", got)
	}
}

func TestDefinitions_注册表一致(t *testing.T) {
	if len(Definitions) != len(Registry) {
		t.Fatalf("Definitions 有 %d 项但 Registry 有 %d 项", len(Definitions), len(Registry))
	}
	seen := make(map[string]bool, len(Definitions))
	for _, d := range Definitions {
		if d.Name == "" {
			t.Error("存在未命名的物料定义")
		}
		if seen[d.Name] {
			t.Errorf("物料 %q 重复注册", d.Name)
		}
		seen[d.Name] = true
		if _, ok := Registry[d.Name]; !ok {
			t.Errorf("物料 %q 未进 Registry", d.Name)
		}
		if d.Source == "" {
			t.Errorf("物料 %q 缺少源文件路径", d.Name)
		}
		if len(d.Tables) == 0 {
			t.Errorf("物料 %q 未声明目标表", d.Name)
		}
	}
}

func TestNames_字典序且完整(t *testing.T) {
	names := Names()
	if len(names) != len(Definitions) {
		t.Fatalf("Names() 返回 %d 个，期望 %d 个", len(names), len(Definitions))
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Fatalf("Names() 非严格字典序：%v", names)
		}
	}
	// 核心物料必须在册
	for _, want := range []string{NamerMaterial, KangxiMaterial, GateMaterial} {
		found := false
		for _, n := range names {
			if n == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Names() 缺少物料 %q", want)
		}
	}
}

func TestTopoSort_依赖先于自身(t *testing.T) {
	order, err := TopoSort(Definitions)
	if err != nil {
		t.Fatalf("TopoSort 失败: %v", err)
	}
	if len(order) != len(Definitions) {
		t.Fatalf("TopoSort 返回 %d 个，期望 %d 个", len(order), len(Definitions))
	}
	pos := make(map[string]int, len(order))
	for i, n := range order {
		pos[n] = i
	}
	for _, d := range Definitions {
		for _, dep := range d.DependsOn {
			if pos[dep] >= pos[d.Name] {
				t.Errorf("物料 %q 排在其依赖 %q 之前（%d >= %d）", d.Name, dep, pos[d.Name], pos[dep])
			}
		}
	}

	// 确定性：同一输入必须得到同一顺序
	again, err := TopoSort(Definitions)
	if err != nil {
		t.Fatalf("二次 TopoSort 失败: %v", err)
	}
	if !reflect.DeepEqual(order, again) {
		t.Fatalf("TopoSort 不确定：\n首次=%v\n二次=%v", order, again)
	}
}

func TestTopoSort_环检测(t *testing.T) {
	defs := []Definition{
		{Name: "a", DependsOn: []string{"b"}},
		{Name: "b", DependsOn: []string{"c"}},
		{Name: "c", DependsOn: []string{"a"}},
	}
	_, err := TopoSort(defs)
	if err == nil {
		t.Fatal("成环依赖应报错")
	}
	var ce *CycleError
	if !asCycleError(err, &ce) {
		t.Fatalf("期望 *CycleError，实得 %T: %v", err, err)
	}
	if len(ce.Cycle) == 0 {
		t.Error("CycleError 应携带环路径")
	}
	if want := "物料依赖成环: "; len(ce.Error()) < len(want) {
		t.Errorf("CycleError 消息格式异常: %q", ce.Error())
	}
}

func TestTopoSort_自环检测(t *testing.T) {
	_, err := TopoSort([]Definition{{Name: "a", DependsOn: []string{"a"}}})
	var ce *CycleError
	if err == nil || !asCycleError(err, &ce) {
		t.Fatalf("自环应报 *CycleError，实得 %v", err)
	}
}

func TestTopoSort_未知依赖(t *testing.T) {
	_, err := TopoSort([]Definition{{Name: "a", DependsOn: []string{"ghost"}}})
	var ue *UnknownMaterialError
	if err == nil || !asUnknownError(err, &ue) {
		t.Fatalf("未知依赖应报 *UnknownMaterialError，实得 %v", err)
	}
	if ue.Name != "ghost" {
		t.Errorf("错误应指向 ghost，实得 %q", ue.Name)
	}
	if ue.Referrer != "a" {
		t.Errorf("错误应记录引用方 a，实得 %q", ue.Referrer)
	}
	if want := "物料 a 依赖了未注册的物料 ghost"; ue.Error() != want {
		t.Errorf("错误消息期望 %q，实得 %q", want, ue.Error())
	}
}

// TestUnknownMaterialError_无引用方 直接构造（不经 TopoSort）时 Referrer 为空，
// 消息走「未注册」分支（TopoSort 路径下 Referrer 恒为引用方，该分支仅可达于
// 调用方自行构造错误的场景）
func TestUnknownMaterialError_无引用方(t *testing.T) {
	direct := (&UnknownMaterialError{Name: "x"}).Error()
	if direct != "物料 x 未注册" {
		t.Errorf("无引用方期望「物料 x 未注册」，实得 %q", direct)
	}
	withRef := (&UnknownMaterialError{Name: "x", Referrer: "a"}).Error()
	if withRef != "物料 a 依赖了未注册的物料 x" {
		t.Errorf("有引用方期望「物料 a 依赖了未注册的物料 x」，实得 %q", withRef)
	}
}

func TestTopoSort_空输入(t *testing.T) {
	order, err := TopoSort(nil)
	if err != nil || len(order) != 0 {
		t.Fatalf("空输入期望空顺序无错误，实得 %v / %v", order, err)
	}
}

// asCycleError / asUnknownError 用显式类型断言代替 errors.As，
// 避免为两个指针错误类型引入 errors 包的间接层
func asCycleError(err error, target **CycleError) bool {
	ce, ok := err.(*CycleError)
	if ok {
		*target = ce
	}
	return ok
}

func asUnknownError(err error, target **UnknownMaterialError) bool {
	ue, ok := err.(*UnknownMaterialError)
	if ok {
		*target = ue
	}
	return ok
}

// TestPathOrEmpty TopoSort 顶层 path 恒为 nil、依赖路径恒非空，
// 两个分支直接调用来守护（未导出辅助函数的契约）
func TestPathOrEmpty(t *testing.T) {
	if got := pathOrEmpty(nil); got != "" {
		t.Errorf("pathOrEmpty(nil) 期望空串，实得 %q", got)
	}
	if got := pathOrEmpty([]string{}); got != "" {
		t.Errorf("pathOrEmpty(空切片) 期望空串，实得 %q", got)
	}
	if got := pathOrEmpty([]string{"a", "b"}); got != "b" {
		t.Errorf("pathOrEmpty 期望取末元素 b，实得 %q", got)
	}
}
