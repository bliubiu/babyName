package yijing

import (
	"os"
	"path/filepath"
	"testing"
)

// GuaDeOf 金标准核对：八纯卦单字、异卦「上X下Y」
func TestGuaDeOf_Golden(t *testing.T) {
	cases := []struct {
		upper, lower int
		want         string
		note         string
	}{
		{1, 1, "健", "乾为天：八纯卦取单字"},
		{8, 8, "顺", "坤为地"},
		{6, 6, "陷", "坎为水"},
		{3, 3, "丽", "离为火"},
		{6, 4, "上陷下动", "水雷屯：坎上震下"},
		{4, 3, "上动下丽", "雷火丰：震上离下"},
		{8, 1, "上顺下健", "地天泰：坤上乾下"},
		{1, 8, "上健下顺", "天地否"},
		{2, 1, "上悦下健", "天泽履：兑上乾下"},
	}
	for _, c := range cases {
		if got := GuaDeOf(c.upper, c.lower); got != c.want {
			t.Errorf("GuaDeOf(%d,%d)=%q，期望 %q（%s）", c.upper, c.lower, got, c.want, c.note)
		}
	}
}

func TestGuaDeOf_InvalidTrigram(t *testing.T) {
	if got := GuaDeOf(0, 4); got != "" {
		t.Errorf("非法上卦应返回空串，实得 %q", got)
	}
	if got := GuaDeOf(6, 9); got != "" {
		t.Errorf("非法下卦应返回空串，实得 %q", got)
	}
}

// 硬编码表 init 兜底：64 卦卦德全部非空、格式合法（1 字或 5 字）
func TestHexagramList_GuaDePopulated(t *testing.T) {
	if len(HexagramList) != 64 {
		t.Fatalf("HexagramList=%d 卦，期望 64", len(HexagramList))
	}
	for _, h := range HexagramList {
		if h.GuaDe == "" {
			t.Errorf("卦 #%d %s 卦德为空", h.Number, h.Name)
			continue
		}
		runes := []rune(h.GuaDe)
		if len(runes) != 1 && len(runes) != 4 {
			t.Errorf("卦 #%d %s 卦德 %q 格式非法（应为 1 字或「上X下Y」4 字）", h.Number, h.Name, h.GuaDe)
		}
		if want := GuaDeOf(h.UpperTrigram, h.LowerTrigram); h.GuaDe != want {
			t.Errorf("卦 #%d %s 卦德 %q 与推导值 %q 不一致", h.Number, h.Name, h.GuaDe, want)
		}
	}
}

// EnsureGuaDe 幂等：已有值不覆盖，空值补算
func TestEnsureGuaDe_Idempotent(t *testing.T) {
	list := []Hexagram{
		{Number: 3, Name: "屯", UpperTrigram: 6, LowerTrigram: 4, GuaDe: "自定义值"},
		{Number: 55, Name: "丰", UpperTrigram: 4, LowerTrigram: 3},
	}
	EnsureGuaDe(list)
	if list[0].GuaDe != "自定义值" {
		t.Errorf("EnsureGuaDe 不应覆盖已有卦德，实得 %q", list[0].GuaDe)
	}
	if list[1].GuaDe != "上动下丽" {
		t.Errorf("空卦德应补算为「上动下丽」，实得 %q", list[1].GuaDe)
	}
}

// LoadFromJSON 对无卦德的历史 JSON 也要补算（装载路径收敛）
func TestLoadFromJSON_FillsGuaDe(t *testing.T) {
	// 保存现场
	orig := HexagramList
	defer func() { HexagramList = orig }()

	dir := t.TempDir()
	payload := `[{"id":3,"name":"屯","number":3,"symbol":"☵☳","upper_trigram":6,` +
		`"lower_trigram":4,"gua_ci":"元亨利贞","xiang_ci":"云雷屯","yao_ci":["a"]}]`
	if err := os.WriteFile(filepath.Join(dir, "yijing.json"), []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadFromJSON(dir); err != nil {
		t.Fatal(err)
	}
	got, ok := GetHexagram(3)
	if !ok {
		t.Fatal("装载后应能取到卦 #3")
	}
	if got.GuaDe != "上陷下动" {
		t.Errorf("JSON 装载应补算卦德为「上陷下动」，实得 %q", got.GuaDe)
	}
}
