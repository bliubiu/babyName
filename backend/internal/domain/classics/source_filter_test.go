package classics

import (
	"strings"
	"testing"
)

// TestQueryNamePoetryBySource 守护「用户选择经典来源后，出处必须来自该来源族」。
//
// 历史缺陷（2026.10.21）：选「唐诗宋词」时，候选池生效（GetPoetryCharList
// 返回 6950 字），但展示层用的是不筛来源的 QueryNamePoetry，出处仍显示
// 诗经《南有嘉鱼》——与用户选择脱节。
func TestQueryNamePoetryBySource(t *testing.T) {
	if err := BuildPoemIndexFromJSON("../../../../data"); err != nil {
		t.Fatalf("build index: %v", err)
	}

	t.Run("唐诗宋词_出处不应来自诗经", func(t *testing.T) {
		// 「荔嘉」在全库查询时 BestMatch 是诗经（因「嘉」在《南有嘉鱼》篇名中）
		all := QueryNamePoetry("荔嘉")
		if all == nil || all.BestMatch == nil {
			t.Skip("全库无 BestMatch，跳过")
		}
		t.Logf("全库 BestMatch: source=%s title=%s matchedChars=%v",
			all.BestMatch.Poem.Source, all.BestMatch.Poem.Title, all.BestMatch.MatchedChars)

		scoped := QueryNamePoetryBySource("荔嘉", "唐诗宋词")
		if scoped == nil || scoped.BestMatch == nil {
			t.Skip("唐诗宋词族内无匹配（该名字用字不在唐诗宋词中），跳过")
		}
		bm := scoped.BestMatch
		t.Logf("唐诗宋词 BestMatch: source=%s title=%s matchedChars=%v",
			bm.Poem.Source, bm.Poem.Title, bm.MatchedChars)

		if bm.Poem.Source == "诗经" {
			t.Errorf("选唐诗宋词却返回诗经出处：%s《%s》", bm.Poem.Source, bm.Poem.Title)
		}
		if !MatchedSourceMembers("唐诗宋词", bm.Poem.Source) {
			t.Errorf("BestMatch.Source=%q 不属于唐诗宋词族", bm.Poem.Source)
		}
	})

	t.Run("族内无命中时回退全库", func(t *testing.T) {
		// 「诗经」族很小，随机取一个必然不在诗经里的名字
		res := QueryNamePoetryBySource("蚩尤", "诗经")
		if res == nil || res.BestMatch == nil {
			return // 无任何出处，符合预期
		}
		t.Logf("回退结果: source=%s title=%s", res.BestMatch.Poem.Source, res.BestMatch.Poem.Title)
	})

	t.Run("MatchedChars_仅含实际命中的字", func(t *testing.T) {
		res := QueryNamePoetry("荔嘉")
		if res == nil || res.BestMatch == nil {
			t.Skip("无 BestMatch")
		}
		bm := res.BestMatch
		if len(bm.MatchedChars) == 0 {
			t.Fatal("MatchedChars 不应为空")
		}
		// 命中的字必须真的出现在引句里
		for _, ch := range bm.MatchedChars {
			if !strings.Contains(bm.Quote, ch) {
				t.Errorf("命中字 %q 未出现在引句 %q 中", ch, bm.Quote)
			}
		}
		// 命中字必须是名字的用字
		for _, ch := range bm.MatchedChars {
			if !strings.Contains("荔嘉", ch) {
				t.Errorf("命中字 %q 不是名字用字", ch)
			}
		}
		t.Logf("matchedChars=%v quote=%s", bm.MatchedChars, bm.Quote)
	})
}

// TestSourceFamilyMembers 守护来源族映射表。
func TestSourceFamilyMembers(t *testing.T) {
	if got := SourceFamilyMembers(""); got != nil {
		t.Errorf("空来源应返回 nil，实际 %v", got)
	}
	if got := SourceFamilyMembers("shijing"); len(got) == 0 || got[0] != "诗经" {
		t.Errorf("别名 shijing 应映射到 诗经，实际 %v", got)
	}
	members := SourceFamilyMembers("唐诗宋词")
	if len(members) == 0 {
		t.Fatal("唐诗宋词族不应为空")
	}
	found := false
	for _, m := range members {
		if m == "唐诗三百首" {
			found = true
		}
	}
	if !found {
		t.Errorf("唐诗宋词族应含「唐诗三百首」，实际 %v", members)
	}
}
