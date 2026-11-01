package yijing

import (
	"os"
	"path/filepath"
	"testing"
)

// ============================================================================
// 彖传 / 小象传 结构化提取测试（docs/42 P1-② 语料结构化）
//
// 解析器 parseHexagramDocFull 是「docs/玄学/易经/*.md → data/yijing.json」
// 的单一真源：canonical_full_test.go 的金标准对拍与本文件共享同一实现，
// 避免双口径解析漂移。
//
// 文档格式两种：
//   - 常规式（62 卦）：彖曰/象曰成段，逐爻「象曰：…」独立行（如蒙、坤）
//   - 合体式（乾）：彖曰独立行，大象与小象合在一行，按句号分段
// ============================================================================

// writeTempDoc 在临时目录写入一篇卦文档样例
func writeTempDoc(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入样例文档失败: %v", err)
	}
	return dir
}

const sampleMeng = `《易经》第四卦 蒙 山水蒙 艮上坎下

蒙：亨。 匪我求童蒙，童蒙求我。利贞。彖曰：蒙，山下有险，险而止，蒙。 蒙以养正，圣功也。 象曰：山下出泉，蒙；君子以果行育德。

初六：发蒙，利用刑人，用说桎梏。

象曰：利用刑人，以正法也。

九二：包蒙吉。

象曰：子克家，刚柔接也。
`

func TestParseHexagramDocFull_Meng(t *testing.T) {
	num, name, guaCi, daXiang, tuanCi, xiaoXiang, yaoCi := parseHexagramDocFull(sampleMeng)
	if num != 4 || name != "蒙" {
		t.Fatalf("卦序/卦名解析错误: num=%d name=%q", num, name)
	}
	if guaCi == "" {
		t.Error("卦辞为空")
	}
	if want := "山下出泉，蒙；君子以果行育德"; daXiang != want {
		t.Errorf("大象=%q，期望 %q（须在爻位前缀处截断）", daXiang, want)
	}
	if want := "蒙，山下有险，险而止，蒙。蒙以养正，圣功也。"; tuanCi != want {
		t.Errorf("彖传=%q，期望 %q", tuanCi, want)
	}
	if len(xiaoXiang) != 2 {
		t.Fatalf("小象 %d 条，期望 2", len(xiaoXiang))
	}
	if xiaoXiang[0] != "利用刑人，以正法也。" || xiaoXiang[1] != "子克家，刚柔接也。" {
		t.Errorf("小象内容不符: %q", xiaoXiang)
	}
	if len(yaoCi) != 2 {
		t.Fatalf("爻辞 %d 条，期望 2", len(yaoCi))
	}
}

const sampleQian = `《易经》第一卦 乾 乾为天 乾上乾下

乾：元，亨，利，贞。

初九：潜龙，勿用。

用九：见群龙无首，吉。

彖曰：大哉乾元，万物资始，乃统天。

象曰：天行健，君子以自强不息。潜龙勿用，阳在下也。 用九，天德不可为首也。
`

func TestParseHexagramDocFull_Qian(t *testing.T) {
	num, name, _, daXiang, tuanCi, xiaoXiang, yaoCi := parseHexagramDocFull(sampleQian)
	if num != 1 || name != "乾" {
		t.Fatalf("卦序/卦名解析错误: num=%d name=%q", num, name)
	}
	if want := "大哉乾元，万物资始，乃统天。"; tuanCi != want {
		t.Errorf("彖传=%q，期望 %q", tuanCi, want)
	}
	if want := "天行健，君子以自强不息"; daXiang != want {
		t.Errorf("大象=%q，期望 %q（合体行首段）", daXiang, want)
	}
	// 合体行：大象后按句号拆出小象（潜龙 + 用九），不足 7 段的样例按实际段数
	if len(xiaoXiang) != 2 {
		t.Fatalf("小象 %d 条，期望 2", len(xiaoXiang))
	}
	if xiaoXiang[0] != "潜龙勿用，阳在下也。" {
		t.Errorf("小象[0]=%q", xiaoXiang[0])
	}
	if xiaoXiang[1] != "用九，天德不可为首也。" {
		t.Errorf("小象[1]=%q", xiaoXiang[1])
	}
	if len(yaoCi) != 2 {
		t.Fatalf("爻辞 %d 条，期望 2（初九+用九）", len(yaoCi))
	}
}

// sampleYue 覆盖爻辞正文含「曰」的两种真实形态：
//   - 大畜九三「曰闲舆卫」（爻辞正文自带「曰」，不得整行丢弃）
//   - 乾文言「九四曰：…」（爻位后接「曰」，文言行须排除）
const sampleYue = `《易经》第二十六卦 大畜 山天大畜 艮上乾下

大畜：利贞，不家食吉，利涉大川。彖曰：大畜，刚健笃实辉光。 象曰：天在山中，大畜；君子以多识前言往行，以畜其德。

初九：有厉利已。

九三：良马逐，利艰贞。 曰闲舆卫，利有攸往。

九四曰：童牛之牿，元吉，何谓也。
`

func TestParseHexagramDocFull_爻辞含曰(t *testing.T) {
	_, _, _, _, _, _, yaoCi := parseHexagramDocFull(sampleYue)
	if len(yaoCi) != 2 {
		t.Fatalf("爻辞 %d 条，期望 2（初九+九三；正文含「曰」的行不得被丢弃）: %q", len(yaoCi), yaoCi)
	}
	want := "良马逐，利艰贞。 曰闲舆卫，利有攸往。"
	if yaoCi[1] != want {
		t.Errorf("九三爻辞=%q，期望 %q", yaoCi[1], want)
	}
}

// TestParseCanonicalDocs_RealDocs 对真实 docs 目录做集成校验：
// 64 卦齐备、彖传非空、小象条数与爻辞条数一致（乾坤 7，其余 6）
func TestParseCanonicalDocs_RealDocs(t *testing.T) {
	dir := findYijingDocsDir(t)
	if dir == "" {
		t.Skip("未找到 docs/玄学/易经 目录，跳过")
	}
	docs, err := ParseCanonicalDocs(dir)
	if err != nil {
		t.Fatalf("ParseCanonicalDocs 失败: %v", err)
	}
	if len(docs) != 64 {
		t.Fatalf("解析出 %d 篇，期望 64", len(docs))
	}
	byNum := map[int]CanonicalDoc{}
	for _, d := range docs {
		byNum[d.Number] = d
	}
	for n := 1; n <= 64; n++ {
		d, ok := byNum[n]
		if !ok {
			t.Errorf("卦 %d 缺失", n)
			continue
		}
		if d.TuanCi == "" {
			t.Errorf("卦 %d（%s）彖传为空", n, d.Name)
		}
		want := 6
		if n == 1 || n == 2 {
			want = 7
		}
		if len(d.XiaoXiang) != want {
			t.Errorf("卦 %d（%s）小象 %d 条，期望 %d", n, d.Name, len(d.XiaoXiang), want)
		}
	}
	// 抽样内容锁：彖传须含卦名特征句
	if q := byNum[1]; q.TuanCi == "" || !contains(q.TuanCi, "大哉乾元") {
		t.Errorf("乾彖传内容异常: %q", q.TuanCi)
	}
	if k := byNum[2]; !contains(k.TuanCi, "至哉坤元") {
		t.Errorf("坤彖传内容异常: %q", k.TuanCi)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
