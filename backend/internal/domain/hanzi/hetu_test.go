package hanzi

import (
	"testing"
)

// --- 河图数理基本映射测试 ---

func TestHetuWuxing_Water(t *testing.T) {
	tests := []int{1, 6, 11, 16, 21, 26}
	for _, strokes := range tests {
		if got := HetuWuxing(strokes); got != "水" {
			t.Errorf("HetuWuxing(%d) = %q, want 水", strokes, got)
		}
	}
}

func TestHetuWuxing_Fire(t *testing.T) {
	tests := []int{2, 7, 12, 17, 22, 27}
	for _, strokes := range tests {
		if got := HetuWuxing(strokes); got != "火" {
			t.Errorf("HetuWuxing(%d) = %q, want 火", strokes, got)
		}
	}
}

func TestHetuWuxing_Wood(t *testing.T) {
	tests := []int{3, 8, 13, 18, 23, 28}
	for _, strokes := range tests {
		if got := HetuWuxing(strokes); got != "木" {
			t.Errorf("HetuWuxing(%d) = %q, want 木", strokes, got)
		}
	}
}

func TestHetuWuxing_Metal(t *testing.T) {
	tests := []int{4, 9, 14, 19, 24, 29}
	for _, strokes := range tests {
		if got := HetuWuxing(strokes); got != "金" {
			t.Errorf("HetuWuxing(%d) = %q, want 金", strokes, got)
		}
	}
}

func TestHetuWuxing_Earth(t *testing.T) {
	tests := []int{5, 10, 15, 20, 25, 30}
	for _, strokes := range tests {
		if got := HetuWuxing(strokes); got != "土" {
			t.Errorf("HetuWuxing(%d) = %q, want 土", strokes, got)
		}
	}
}

func TestHetuWuxing_Zero(t *testing.T) {
	if got := HetuWuxing(0); got != "土" {
		t.Errorf("HetuWuxing(0) = %q, want 土", got)
	}
}

func TestHetuWuxing_Negative(t *testing.T) {
	if got := HetuWuxing(-1); got != "" {
		t.Errorf("HetuWuxing(-1) = %q, want 空字符串", got)
	}
}

// --- 汉字河图五行测试 ---

func TestHetuWuxingOfChar_Known(t *testing.T) {
	tests := []struct {
		char string
		want string
	}{
		{"一", "水"},   // 1画 → 水
		{"二", "火"},   // 2画 → 火
		{"三", "木"},   // 3画 → 木
		{"四", "土"},   // 四=5画（传统笔画：囗+儿=5）→ 5→土
		{"五", "金"},   // 五=4画 → 4→金
	}

	for _, tc := range tests {
		t.Run(tc.char, func(t *testing.T) {
			got := HetuWuxingOfChar(tc.char)
			if got != tc.want {
				if data, ok := HanziData[tc.char]; ok {
					t.Errorf("HetuWuxingOfChar(%q) = %q, want %q; strokes=%d", tc.char, got, tc.want, data.Strokes)
				} else {
					t.Errorf("HetuWuxingOfChar(%q) = %q, want %q; char not found", tc.char, got, tc.want)
				}
			}
		})
	}
}

// --- 姓名河图五行测试 ---

func TestHetuWuxingOfName_Single(t *testing.T) {
	// "王"=4画 → 金
	wxs := HetuWuxingOfName("王")
	if len(wxs) != 1 {
		t.Fatalf("HetuWuxingOfName(\"王\") 返回 %d 个元素, want 1", len(wxs))
	}
	if wxs[0] != "金" {
		t.Errorf("姓\"王\"的河图五行 = %q, want 金 (4画)", wxs[0])
	}
}

func TestHetuWuxingOfName_Double(t *testing.T) {
	// "李"=7画 → 火, "明"=8画 → 木
	wxs := HetuWuxingOfName("李", "明")
	if len(wxs) != 2 {
		t.Fatalf("HetuWuxingOfName(\"李\",\"明\") 返回 %d 个元素, want 2", len(wxs))
	}
	if wxs[0] != "火" {
		t.Errorf("姓\"李\"(7画)的河图五行 = %q, want 火", wxs[0])
	}
	if wxs[1] != "木" {
		t.Errorf("名\"明\"(8画)的河图五行 = %q, want 木", wxs[1])
	}
}

func TestHetuTotalWuxing_Known(t *testing.T) {
	// "王"=4画 + "伟"=6画 = 10 → 土
	if got := HetuTotalWuxing("王", "伟"); got != "土" {
		data1, _ := HanziData["王"]
		data2, _ := HanziData["伟"]
		t.Errorf("HetuTotalWuxing(王=%d画, 伟=%d画) = %q, want 土", data1.Strokes, data2.Strokes, got)
	}
}

func TestHetuTotalWuxing_Triple(t *testing.T) {
	// 李(7) + 小(3) + 明(8) = 18 → 木
	if got := HetuTotalWuxing("李", "小", "明"); got != "木" {
		data1, _ := HanziData["李"]
		data2, _ := HanziData["小"]
		data3, _ := HanziData["明"]
		t.Errorf("HetuTotalWuxing(李=%d画, 小=%d画, 明=%d画) = %q, want 木", data1.Strokes, data2.Strokes, data3.Strokes, got)
	}
}

// --- 边界测试 ---

func TestHetuWuxingOfChar_NotFound(t *testing.T) {
	if got := HetuWuxingOfChar("𪚥"); got != "" {
		t.Errorf("HetuWuxingOfChar(不存在的字) = %q, want 空字符串", got)
	}
}

func TestHetuWuxingOfName_EmptyGivenNames(t *testing.T) {
	wxs := HetuWuxingOfName("王")
	if len(wxs) == 0 {
		t.Error("HetuWuxingOfName(王) should return at least one element")
	}
}
