package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"name/internal/materials/schema"
)

// 本文件守护精选策展名物料（docs/31 ADR-008 运行时只读 SQLite）的落库语义：
// 种子行富字段保真 + 用户自学习行不被导入清空。

// TestImportCurated_种子行入库并保留富字段 导入后 curated_names 承载 meaning/
// wuxing/styles/tags，避免运行时改读库后策展释义与风格标签丢失。
func TestImportCurated_种子行入库并保留富字段(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	report := importMinimal(t, s, dir, ImportOptions{})
	r := resultOf(report, schema.CuratedMaterial)
	if r.Err != "" {
		t.Fatalf("策展名导入失败: %s", r.Err)
	}
	if r.Rows != 3 {
		t.Errorf("策展名行数期望 3（空名应被丢弃），实得 %d", r.Rows)
	}

	var meaning, wuxing, styles, tags, source string
	err := s.db.QueryRow(`SELECT meaning, wuxing, styles, tags, source
		FROM curated_names WHERE name = '子轩'`).Scan(&meaning, &wuxing, &styles, &tags, &source)
	if err != nil {
		t.Fatalf("查询策展名失败: %v", err)
	}
	if meaning != "志向高远" {
		t.Errorf("meaning 期望「志向高远」，实得 %q", meaning)
	}
	if wuxing != "水木" {
		t.Errorf("wuxing 期望「水木」，实得 %q", wuxing)
	}
	if styles != `["古典风","大气"]` && styles != `["大气","古典风"]` {
		t.Errorf("styles 未正确落库，实得 %q", styles)
	}
	if tags != `["高频平","志向"]` && tags != `["志向","高频平"]` {
		t.Errorf("tags 未正确落库，实得 %q", tags)
	}
	if source != "seed" {
		t.Errorf("种子行 source 期望 seed，实得 %q", source)
	}

	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM curated_names WHERE name = ''").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("空名策展条目必须被丢弃")
	}
}

// TestImportCurated_保留用户自学习行 重灌不得清掉 source=user_favorite 的收藏行
func TestImportCurated_保留用户自学习行(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})

	// 用户收藏一个不在策展表里的名字
	if err := s.SaveCuratedName("欧阳明", "ou yang ming", "male", 99, userFavoriteSource); err != nil {
		t.Fatalf("写入用户收藏失败: %v", err)
	}

	importMinimal(t, s, dir, ImportOptions{Lenient: true}) // 绕过指纹短路，强制重灌

	var src string
	if err := s.db.QueryRow("SELECT source FROM curated_names WHERE name = '欧阳明'").Scan(&src); err != nil {
		t.Fatalf("用户收藏行丢失: %v", err)
	}
	if src != userFavoriteSource {
		t.Errorf("用户收藏行 source 期望 %q，实得 %q", userFavoriteSource, src)
	}
}

// TestImportCurated_种子修订覆盖旧值 源文件改内容后重灌，种子行应随之更新
func TestImportCurated_种子修订覆盖旧值(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})

	revised := `[
      {"name":"子轩","pinyin":"zi xuan","gender":"male","source":"修订","meaning":"新释义","wuxing":"木","yinyun_score":80,"styles":["新风"],"tags":["修订"]},
      {"name":"浩然","pinyin":"hao ran","gender":"male","source":"易经","meaning":"浩然正气","wuxing":"水木","yinyun_score":92,"styles":["古典风"],"tags":["阳刚"]}
    ]`
	if err := os.WriteFile(filepath.Join(dir, "curated_names.json"), []byte(revised), 0o644); err != nil {
		t.Fatalf("改写策展源失败: %v", err)
	}

	importMinimal(t, s, dir, ImportOptions{})

	var meaning string
	if err := s.db.QueryRow("SELECT meaning FROM curated_names WHERE name = '子轩'").Scan(&meaning); err != nil {
		t.Fatal(err)
	}
	if meaning != "新释义" {
		t.Errorf("策展名释义应随源修订，实得 %q", meaning)
	}
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM curated_names WHERE name = '永嘉'").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("源中已删除的种子行必须同步删除")
	}
}

// TestImportCurated_用户收藏同名时让位用户行 用户收藏的同名条目应保留其 source
func TestImportCurated_用户收藏同名时让位用户行(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})

	// 用户收藏了策展表中已存在的「永嘉」
	if err := s.SaveCuratedName("永嘉", "yong jia", "male", 100, userFavoriteSource); err != nil {
		t.Fatalf("写入用户收藏失败: %v", err)
	}

	importMinimal(t, s, dir, ImportOptions{Lenient: true})

	var src string
	if err := s.db.QueryRow("SELECT source FROM curated_names WHERE name = '永嘉'").Scan(&src); err != nil {
		t.Fatal(err)
	}
	if src != userFavoriteSource {
		t.Errorf("重灌不得覆盖用户收藏行，实得 source=%q", src)
	}
}

// TestLoadAllCuratedNames_返回富字段 运行时经持久化器装载，需拿到 meaning/wuxing/styles/tags
func TestLoadAllCuratedNames_返回富字段(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})

	entries, err := s.LoadAllCuratedNames()
	if err != nil {
		t.Fatalf("读取策展名失败: %v", err)
	}

	var found bool
	for _, e := range entries {
		if e.Name != "浩然" {
			continue
		}
		found = true
		if e.Meaning != "浩然正气" {
			t.Errorf("meaning 期望「浩然正气」，实得 %q", e.Meaning)
		}
		if e.Wuxing != "水木" {
			t.Errorf("wuxing 期望「水木」，实得 %q", e.Wuxing)
		}
		if len(e.Styles) != 2 {
			t.Errorf("styles 期望 2 项，实得 %v", e.Styles)
		}
		if len(e.Tags) != 2 {
			t.Errorf("tags 期望 2 项，实得 %v", e.Tags)
		}
	}
	if !found {
		t.Fatal("策展名「浩然」应可从库读出")
	}
}
