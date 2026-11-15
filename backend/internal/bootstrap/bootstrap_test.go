package bootstrap

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"name/internal/infrastructure/database/sqlite"
	"name/internal/paths"
	"name/internal/testutil"
)

// callerFile 定位本包源文件，用于 testutil.DataDir 探测仓库根 data/
var callerFile = func() string {
	_, file, _, _ := runtime.Caller(0)
	return file
}()

// --- 最小物料 fixture（与 sqlite 包导入测试同口径） ---

// writeMinimalMaterials 在临时目录生成一份最小可用物料集
func writeMinimalMaterials(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	raw := filepath.Join(dir, "raw")
	if err := os.MkdirAll(raw, 0o755); err != nil {
		t.Fatalf("创建 raw 目录失败: %v", err)
	}

	kangxi := "\xEF\xBB\xBF" +
		"MIT License\nCopyright (c)\nhttps://example.com/kangxi\n\n" +
		"CodePoint,Value,Character,Strokes\n" +
		"20108,27700,永,5\n" +
		"27700,27700,水,4\n"
	if err := os.WriteFile(filepath.Join(raw, "kangxi-strokecount.csv"), []byte(kangxi), 0o644); err != nil {
		t.Fatalf("写入康熙 CSV 失败: %v", err)
	}

	namer := `{
  "version": "1.0",
  "generated": "2026-01-01",
  "totalChars": 3,
  "chars": [
    {"char":"永","pinyin":"yong3","strokes":5,"radical":"水","meaning":"水流长","wuxing":"水","level":1,"gender":"男","tone":3,"positiveScore":90,"namingCategories":["品德"],"isPolyphonic":true},
    {"char":"水","pinyin":"shui3","strokes":4,"radical":"水","meaning":"水体","wuxing":"水","level":1,"gender":"female"},
    {"char":"和","pinyin":"he2","strokes":8,"radical":"禾","wuxing":"","level":2}
  ],
  "charGroups": [
    {"radical":"水","name":"水字旁","meaning":"与水有关","chars":["永","水"]}
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "namer.json"), []byte(namer), 0o644); err != nil {
		t.Fatalf("写入 namer.json 失败: %v", err)
	}

	gate := `["伙","腕","之"]`
	if err := os.WriteFile(filepath.Join(dir, "naming_quality.json"), []byte(gate), 0o644); err != nil {
		t.Fatalf("写入门禁表失败: %v", err)
	}

	forbidden := `["作死","挽尊"]`
	if err := os.WriteFile(filepath.Join(dir, "forbidden_combos.json"), []byte(forbidden), 0o644); err != nil {
		t.Fatalf("写入禁忌表失败: %v", err)
	}

	yijing := `[{"id":1,"name":"乾","number":1,"symbol":"☰","upper_trigram":7,"lower_trigram":7,"gua_ci":"元亨利贞","xiang_ci":"天行健","yao_ci":["初九","九二","九三"],"tuan_ci":"大哉乾元，万物资始","xiao_xiang":["天行健，君子以自强不息","见龙在田，德施普也","终日乾乾，反复道也"],"interpretation":"刚健中正"}]`
	if err := os.WriteFile(filepath.Join(dir, "yijing.json"), []byte(yijing), 0o644); err != nil {
		t.Fatalf("写入易经失败: %v", err)
	}

	zodiac := `[{"id":1,"name":"鼠","wuxing":"水","compatible":["龙","猴"],"conflicting":["马"],"avoid_chars":"午","lucky_number":[1,13]}]`
	if err := os.WriteFile(filepath.Join(dir, "zodiac.json"), []byte(zodiac), 0o644); err != nil {
		t.Fatalf("写入生肖失败: %v", err)
	}

	freq := `{"meta":{"total_names":100,"built_at":"2026-01-01"},"frequency":[
      {"char":"永","count":500,"rank":1,"tier":5},
      {"char":"水","count":300,"rank":2,"tier":4},
      {"char":"和","count":10,"rank":9,"tier":1}
    ]}`
	if err := os.WriteFile(filepath.Join(dir, "name_frequency.json"), []byte(freq), 0o644); err != nil {
		t.Fatalf("写入单字频率失败: %v", err)
	}

	bigram := `{"meta":{"total_names":100,"built_at":"2026-01-01"},"frequency":[
      {"char1":"永","char2":"水","count":20,"rank":3,"tier":4}
    ]}`
	if err := os.WriteFile(filepath.Join(dir, "name_bigram_frequency.json"), []byte(bigram), 0o644); err != nil {
		t.Fatalf("写入双字频率失败: %v", err)
	}

	curated := `[
      {"name":"子轩","pinyin":"zi xuan","gender":"male","source":"百家姓","meaning":"志向高远","wuxing":"水木","yinyun_score":88,"styles":["古典风","大气"],"tags":["志向","高频平"]},
      {"name":"浩然","pinyin":"hao ran","gender":"male","source":"易经","meaning":"浩然正气","wuxing":"水木","yinyun_score":92,"styles":["古典风","大气"],"tags":["阳刚","高频平"]}
    ]`
	if err := os.WriteFile(filepath.Join(dir, "curated_names.json"), []byte(curated), 0o644); err != nil {
		t.Fatalf("写入策展名失败: %v", err)
	}

	// 经典语料最小文件：保证 classics_extracts 物化出字集行（运行期 ADR-008 装载必需）。
	// shijing.json 与 chuci.json 是 LoadFromJSON 的 P0 硬依赖，缺失即导入失败。
	shijing := `[{"title":"水木","chapter":"测试","section":"永水","content":["永水长流，木秀于林。"]}]`
	if err := os.WriteFile(filepath.Join(dir, "shijing.json"), []byte(shijing), 0o644); err != nil {
		t.Fatalf("写入诗经失败: %v", err)
	}
	chuci := `[{"title":"涉江","chapter":"九章","section":"测试","content":["永水汤汤，木叶纷纷。"]}]`
	if err := os.WriteFile(filepath.Join(dir, "chuci.json"), []byte(chuci), 0o644); err != nil {
		t.Fatalf("写入楚辞失败: %v", err)
	}

	return dir
}

// --- 测试 ---

func TestResolveDataDir_优先级与兜底(t *testing.T) {
	t.Run("显式指定优先", func(t *testing.T) {
		t.Setenv("DATA_DIR", "D:\\not-exist")
		dir, err := ResolveDataDir("D:\\fixture")
		if err != nil || dir != "D:\\fixture" {
			t.Fatalf("显式指定应优先，实际: dir=%s err=%v", dir, err)
		}
	})
	t.Run("环境变量次之", func(t *testing.T) {
		t.Setenv("DATA_DIR", "D:\\from-env")
		dir, err := ResolveDataDir("")
		if err != nil || dir != "D:\\from-env" {
			t.Fatalf("DATA_DIR 应生效，实际: dir=%s err=%v", dir, err)
		}
	})
	t.Run("无任何来源返回错误", func(t *testing.T) {
		t.Setenv("DATA_DIR", "")
		saved := findDataDir
		findDataDir = func() string { return "" }
		defer func() { findDataDir = saved }()

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("获取工作目录失败: %v", err)
		}
		tmp := t.TempDir()
		if err := os.Chdir(tmp); err != nil {
			t.Fatalf("切换工作目录失败: %v", err)
		}
		defer func() { _ = os.Chdir(cwd) }()

		if _, err := ResolveDataDir(""); err == nil {
			t.Fatal("无任何来源应返回错误")
		}
	})
	t.Run("回落 cwd/data", func(t *testing.T) {
		t.Setenv("DATA_DIR", "")
		saved := findDataDir
		findDataDir = func() string { return "" }
		defer func() { findDataDir = saved }()

		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("获取工作目录失败: %v", err)
		}
		tmp := t.TempDir()
		if err := os.MkdirAll(filepath.Join(tmp, "data"), 0o755); err != nil {
			t.Fatalf("创建 data 目录失败: %v", err)
		}
		if err := os.Chdir(tmp); err != nil {
			t.Fatalf("切换工作目录失败: %v", err)
		}
		defer func() { _ = os.Chdir(cwd) }()

		dir, err := ResolveDataDir("")
		if err != nil || filepath.Base(dir) != "data" {
			t.Fatalf("应回落 cwd/data，实际: dir=%s err=%v", dir, err)
		}
	})
}

// makeExeDataLayout 造一份「exe 同级 data/ 含 namer.json」的打包布局，返回 exe 路径
func makeExeDataLayout(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatalf("创建 data 目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "data", "namer.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("写入 namer.json 失败: %v", err)
	}
	return filepath.Join(root, "namer-desktop.exe")
}

// replaceFindDataDir 临时替换 dev 仓库探测为无果（模拟打包态），返回恢复函数
func replaceFindDataDir(t *testing.T) {
	t.Helper()
	saved := findDataDir
	findDataDir = func() string { return "" }
	t.Cleanup(func() { findDataDir = saved })
}

// replaceFindDataDirFromExe 临时替换 exe 目录探测，返回恢复函数
func replaceFindDataDirFromExe(t *testing.T, exePath string) {
	t.Helper()
	saved := findDataDirFromExe
	findDataDirFromExe = func() string { return paths.FindDataDirFromExe(exePath) }
	t.Cleanup(func() { findDataDirFromExe = saved })
}

func TestResolveDataDirDetailed_来源分级(t *testing.T) {
	t.Run("显式指定", func(t *testing.T) {
		t.Setenv("DATA_DIR", "D:\\not-exist")
		dir, source, err := ResolveDataDirDetailed("D:\\fixture")
		if err != nil || dir != "D:\\fixture" || source != SourceExplicit {
			t.Fatalf("应命中显式来源，实际: dir=%s source=%v err=%v", dir, source, err)
		}
	})
	t.Run("环境变量", func(t *testing.T) {
		t.Setenv("DATA_DIR", "D:\\from-env")
		dir, source, err := ResolveDataDirDetailed("")
		if err != nil || dir != "D:\\from-env" || source != SourceEnvVar {
			t.Fatalf("应命中环境变量来源，实际: dir=%s source=%v err=%v", dir, source, err)
		}
	})
	t.Run("dev仓库探测", func(t *testing.T) {
		t.Setenv("DATA_DIR", "")
		saved := findDataDir
		findDataDir = func() string { return "D:\\repo\\data" }
		t.Cleanup(func() { findDataDir = saved })

		dir, source, err := ResolveDataDirDetailed("")
		if err != nil || dir != "D:\\repo\\data" || source != SourceDevRepo {
			t.Fatalf("应命中 dev 仓库来源，实际: dir=%s source=%v err=%v", dir, source, err)
		}
	})
	t.Run("exe同级data", func(t *testing.T) {
		t.Setenv("DATA_DIR", "")
		replaceFindDataDir(t)
		exePath := makeExeDataLayout(t)
		replaceFindDataDirFromExe(t, exePath)

		dir, source, err := ResolveDataDirDetailed("")
		if err != nil || source != SourceExeDir || filepath.Base(dir) != "data" {
			t.Fatalf("应命中 exe 同级 data 来源，实际: dir=%s source=%v err=%v", dir, source, err)
		}
	})
	t.Run("cwddata兜底", func(t *testing.T) {
		t.Setenv("DATA_DIR", "")
		replaceFindDataDir(t)
		replaceFindDataDirFromExe(t, "")
		tmp := t.TempDir()
		if err := os.MkdirAll(filepath.Join(tmp, "data"), 0o755); err != nil {
			t.Fatalf("创建 data 目录失败: %v", err)
		}
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("获取工作目录失败: %v", err)
		}
		if err := os.Chdir(tmp); err != nil {
			t.Fatalf("切换工作目录失败: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(cwd) })

		dir, source, err := ResolveDataDirDetailed("")
		if err != nil || source != SourceCwdDir || filepath.Base(dir) != "data" {
			t.Fatalf("应回落 cwd/data，实际: dir=%s source=%v err=%v", dir, source, err)
		}
	})
	t.Run("皆无报错", func(t *testing.T) {
		t.Setenv("DATA_DIR", "")
		replaceFindDataDir(t)
		replaceFindDataDirFromExe(t, "")
		tmp := t.TempDir()
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("获取工作目录失败: %v", err)
		}
		if err := os.Chdir(tmp); err != nil {
			t.Fatalf("切换工作目录失败: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(cwd) })

		if _, _, err := ResolveDataDirDetailed(""); err == nil {
			t.Fatal("无任何来源应返回错误")
		}
	})
}

func TestResolveDataDir_与Detailed同口径(t *testing.T) {
	t.Setenv("DATA_DIR", "D:\\from-env")
	dir, err := ResolveDataDir("")
	if err != nil || dir != "D:\\from-env" {
		t.Fatalf("薄包装应与 Detailed 同结果，实际: dir=%s err=%v", dir, err)
	}
}

func TestDefaultDBPath_Dev与便携两语义(t *testing.T) {
	t.Run("dev仓库：物料目录上一级（现行为不变）", func(t *testing.T) {
		got := DefaultDBPath(filepath.Join("D:", "repo", "data"), SourceDevRepo)
		if want := filepath.Join("D:", "repo", "namer.db"); got != want {
			t.Fatalf("DefaultDBPath = %q, 期望 %q", got, want)
		}
	})
	t.Run("便携态：data文件夹内（exe同级）", func(t *testing.T) {
		got := DefaultDBPath(filepath.Join("D:", "app", "data"), SourceExeDir)
		if want := filepath.Join("D:", "app", "data", "namer.db"); got != want {
			t.Fatalf("DefaultDBPath = %q, 期望 %q", got, want)
		}
	})
}

func TestDefaultLogDir_Dev与便携两语义(t *testing.T) {
	t.Run("dev仓库：物料目录上一级（现行为不变）", func(t *testing.T) {
		got := DefaultLogDir(filepath.Join("D:", "repo", "data"), SourceDevRepo)
		if want := filepath.Join("D:", "repo", "logs"); got != want {
			t.Fatalf("DefaultLogDir = %q, 期望 %q", got, want)
		}
	})
	t.Run("便携态：data文件夹内（exe同级）", func(t *testing.T) {
		got := DefaultLogDir(filepath.Join("D:", "app", "data"), SourceCwdDir)
		if want := filepath.Join("D:", "app", "data", "logs"); got != want {
			t.Fatalf("DefaultLogDir = %q, 期望 %q", got, want)
		}
	})
}

func TestRun_内存模式装配桌面应用(t *testing.T) {
	dataDir := testutil.MustDataDir(t, callerFile)

	res, err := Run(Options{DataDir: dataDir})
	if err != nil {
		t.Fatalf("内存模式装配失败: %v", err)
	}
	defer func() { _ = res.Close() }()

	if res.App == nil {
		t.Fatal("应返回装配好的桌面端 App")
	}
	groups, err := res.App.GetCharGroups()
	if err != nil || len(groups) == 0 {
		t.Fatalf("装配后偏旁分组应为非空，实际: %d 个, err=%v", len(groups), err)
	}
}

func TestRun_SQLite首跑自动播种物料(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	fixture := writeMinimalMaterials(t)

	res, err := Run(Options{DBPath: dbPath, DataDir: fixture, SeedFromJSON: true})
	if err != nil {
		t.Fatalf("SQLite 首跑装配失败: %v", err)
	}
	defer func() { _ = res.Close() }()

	s, ok := res.Store.(*sqlite.Store)
	if !ok {
		t.Fatalf("应使用 SQLite 存储，实际: %T", res.Store)
	}
	rows, err := s.HanziRowCount()
	if err != nil {
		t.Fatalf("查询字库行数失败: %v", err)
	}
	if rows == 0 {
		t.Fatal("首跑应把 JSON 物料播种进 SQLite 字库")
	}

	groups, err := res.App.GetCharGroups()
	if err != nil || len(groups) == 0 {
		t.Fatalf("播种后偏旁分组应为非空，实际: %d 个, err=%v", len(groups), err)
	}
}

func TestRun_SQLite二次启动指纹短路不重复导入(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	fixture := writeMinimalMaterials(t)

	first, err := Run(Options{DBPath: dbPath, DataDir: fixture, SeedFromJSON: true})
	if err != nil {
		t.Fatalf("首跑失败: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("首跑关闭失败: %v", err)
	}
	res, err := Run(Options{DBPath: dbPath, DataDir: fixture, SeedFromJSON: true})
	if err != nil {
		t.Fatalf("二次启动失败: %v", err)
	}
	defer func() { _ = res.Close() }()
}

func TestRun_空库未播种拒绝启动(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	fixture := writeMinimalMaterials(t)

	_, err := Run(Options{DBPath: dbPath, DataDir: fixture, SeedFromJSON: false})
	if err == nil {
		t.Fatal("空库且未播种应 fail-fast 拒绝启动")
	}
}

func TestClose_重复关闭安全(t *testing.T) {
	dataDir := testutil.MustDataDir(t, callerFile)
	res, err := Run(Options{DataDir: dataDir})
	if err != nil {
		t.Fatalf("装配失败: %v", err)
	}
	if err := res.Close(); err != nil {
		t.Fatalf("首次关闭失败: %v", err)
	}
	if err := res.Close(); err != nil {
		t.Fatalf("重复关闭应安全，实际: %v", err)
	}
}
