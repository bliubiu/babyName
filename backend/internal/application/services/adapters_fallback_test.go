package services

// adapters_fallback_test.go — SQL 失败必须降级到 Go 端过滤（docs/29 B8）
//
// B8 的原始病灶：SearchHanziByFilter 的签名只有返回值没有 error。SQL 查询
// 出错时 store 返回 nil，上层判空后把它当成「SQL 正常执行但没有匹配字」，
// 于是 FindCharacters 里注释承诺的「SQL 失败降级 Go 端全表过滤」路径
// 永不可达 —— 数据库故障伪装成业务空结果，前端只看到「生成不出名字」，
// 运维从接口响应里完全看不出 DB 已经挂了。
//
// 这里锁死「nil + nil error = 真的没有匹配」与「nil + error = 故障」的区别。

import (
	"errors"
	"testing"

	"name/internal/infrastructure/database"
	"name/internal/domain/fate"
)

// errStore 模拟一个查询失败的 store
type errStore struct {
	err error
}

func (s *errStore) SearchHanziByFilter(wuxing string, minStrokes, maxStrokes int,
	hasPositive, isRegular bool, chars []string, limit int) ([]*database.Hanzi, error) {
	return nil, s.err
}

// emptyStore 模拟一个查询成功但无匹配的 store
type emptyStore struct{}

func (s *emptyStore) SearchHanziByFilter(wuxing string, minStrokes, maxStrokes int,
	hasPositive, isRegular bool, chars []string, limit int) ([]*database.Hanzi, error) {
	return nil, nil
}

// TestSQLiteHanziFilter_PropagatesError store 的错误必须上抛，不能被适配层吞掉
func TestSQLiteHanziFilter_PropagatesError(t *testing.T) {
	boom := errors.New("database is locked")
	filter := NewSQLiteHanziFilter(&errStore{err: boom})

	_, err := filter.SearchHanziByFilter("", 0, 0, false, false, nil, 100)
	if !errors.Is(err, boom) {
		t.Fatalf("适配层必须上抛 store 错误，实际 err=%v", err)
	}
}

// TestSQLiteHanziFilter_NilStoreReturnsError store 未注入时也要报错而不是返回空
func TestSQLiteHanziFilter_NilStoreReturnsError(t *testing.T) {
	filter := NewSQLiteHanziFilter(nil)
	if _, err := filter.SearchHanziByFilter("", 0, 0, false, false, nil, 100); err == nil {
		t.Fatal("store 为 nil 时必须返回错误，否则上层会把装配缺失当成无匹配")
	}
}

// TestFindCharacters_FallsBackWhenSQLFails SQL 故障时仍须给出结果（降级路径可达）
func TestFindCharacters_FallsBackWhenSQLFails(t *testing.T) {
	// 字库由 data.Init 载入内存，纯 Go 降级路径依赖它
	setupNameServiceE2E(t)

	p := &HanziDataProvider{}
	p.SetSQLFilter(NewSQLiteHanziFilter(&errStore{err: errors.New("database is locked")}))

	// 金行 + 8~12 画：SQL 挂了也必须由 Go 端全表过滤兜住
	chars, err := p.FindCharacters(&fate.BasicCharacterQuery{WuxingIn: []string{"金"}, StrokeGTE: 8, StrokeLTE: 12})
	if err != nil {
		t.Fatalf("SQL 故障不应向上返回错误，应降级: %v", err)
	}
	if len(chars) == 0 {
		t.Fatal("SQL 故障后 Go 端降级仍应返回候选字，实际 0 —— B8 所说的降级路径不可达")
	}
	for _, c := range chars {
		if c.WuXing != "金" {
			t.Errorf("降级结果混入非金行字 %q(%s)", c.Char, c.WuXing)
		}
	}
}

// TestFindCharacters_EmptyResultIsNotError 查询成功但无匹配是合法业务结果
//
// 与上一条严格对照：故障必须降级并给出结果；真的没有匹配字则如实返回空。
// 两者都是 err == nil，区别只在结果集 —— 这正是 B8 要恢复的可区分性。
func TestFindCharacters_EmptyResultIsNotError(t *testing.T) {
	setupNameServiceE2E(t)

	p := &HanziDataProvider{}
	p.SetSQLFilter(NewSQLiteHanziFilter(&emptyStore{}))

	// SQL 权威层说「无匹配」时不应再走 Go 端兜底，否则降级会掩盖索引/数据问题
	chars, err := p.FindCharacters(&fate.BasicCharacterQuery{WuxingIn: []string{"金"}, StrokeGTE: 8, StrokeLTE: 12})
	if err != nil {
		t.Fatalf("空结果不应视为错误: %v", err)
	}
	if len(chars) != 0 {
		t.Errorf("SQL 返回无匹配时应如实返回空，实际 %d 条", len(chars))
	}
}

// TestFindCharacters_SQLAndFallbackAgree 两条路径结果一致
//
// SQL 只是粗筛加速层，最终集合必须与纯 Go 路径相同，否则会出现
// 「有时能出名字、有时出不来」这种依赖数据源的不确定性。
func TestFindCharacters_SQLAndFallbackAgree(t *testing.T) {
	setupNameServiceE2E(t)

	query := &fate.BasicCharacterQuery{WuxingIn: []string{"水"}, StrokeGTE: 6, StrokeLTE: 10}

	pure := &HanziDataProvider{}
	want, err := pure.FindCharacters(query)
	if err != nil {
		t.Fatalf("纯 Go 路径报错: %v", err)
	}
	if len(want) == 0 {
		t.Skip("内存字库无符合条件的水行字，跳过一致性对比")
	}

	fallback := &HanziDataProvider{}
	fallback.SetSQLFilter(NewSQLiteHanziFilter(&errStore{err: errors.New("down")}))
	got, err := fallback.FindCharacters(query)
	if err != nil {
		t.Fatalf("降级路径报错: %v", err)
	}
	if len(got) != len(want) {
		t.Errorf("降级路径结果数 %d 与纯 Go 路径 %d 不一致", len(got), len(want))
	}
}